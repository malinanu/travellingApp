package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleIssuer  = "https://appleid.apple.com"
	appleKeysURL = "https://appleid.apple.com/auth/keys"

	appleKeysTTL        = time.Hour
	appleKeysMinRefresh = time.Minute
)

// AppleIdentity is what a verified Sign in with Apple identity token proves.
type AppleIdentity struct {
	Subject       string // Apple's stable, app-scoped user id
	Email         string // may be empty, or a private-relay address
	EmailVerified bool
}

// AppleVerifier verifies Sign in with Apple identity tokens (RS256 JWTs signed
// by Apple) against Apple's published public keys. Without this, /auth/apple
// would trust whatever id/email the client claims.
type AppleVerifier struct {
	audience string // the app's bundle id
	keysURL  string
	client   *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// NewAppleVerifier returns a verifier accepting tokens issued to audience
// (the iOS bundle id, e.g. com.spotto.spotto).
func NewAppleVerifier(audience string) *AppleVerifier {
	return &AppleVerifier{
		audience: audience,
		keysURL:  appleKeysURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Verify checks signature, issuer, audience and expiry, and returns the
// verified identity. Any failure means the token must not be trusted.
func (v *AppleVerifier) Verify(ctx context.Context, rawToken string) (AppleIdentity, error) {
	if rawToken == "" {
		return AppleIdentity{}, errors.New("identity token required")
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token has no key id")
		}
		return v.key(ctx, kid)
	},
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithIssuer(appleIssuer),
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return AppleIdentity{}, fmt.Errorf("invalid apple identity token: %w", err)
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return AppleIdentity{}, errors.New("apple identity token has no subject")
	}
	id := AppleIdentity{Subject: sub}
	id.Email, _ = claims["email"].(string)
	// Apple sends email_verified as either a bool or the string "true".
	switch ev := claims["email_verified"].(type) {
	case bool:
		id.EmailVerified = ev
	case string:
		id.EmailVerified = ev == "true"
	}
	return id, nil
}

// key returns Apple's public key for kid, refreshing the cached key set when
// it's stale or when kid is unknown (Apple rotates keys), rate-limited so a
// stream of forged kids can't make us hammer Apple.
func (v *AppleVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	age := time.Since(v.fetchedAt)
	if k, ok := v.keys[kid]; ok && age < appleKeysTTL {
		return k, nil
	}
	if v.keys == nil || age >= appleKeysMinRefresh {
		keys, err := v.fetchKeys(ctx)
		if err != nil {
			return nil, err
		}
		v.keys, v.fetchedAt = keys, time.Now()
	}
	if k, ok := v.keys[kid]; ok {
		return k, nil
	}
	return nil, errors.New("unknown apple signing key")
}

func (v *AppleVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.keysURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch apple keys: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch apple keys: status %d", res.StatusCode)
	}
	var set struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&set); err != nil {
		return nil, fmt.Errorf("decode apple keys: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			continue
		}
		keys[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	if len(keys) == 0 {
		return nil, errors.New("apple returned no usable keys")
	}
	return keys, nil
}
