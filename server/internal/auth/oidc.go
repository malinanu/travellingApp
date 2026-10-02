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
	"slices"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	appleIssuer  = "https://appleid.apple.com"
	appleKeysURL = "https://appleid.apple.com/auth/keys"

	googleKeysURL = "https://www.googleapis.com/oauth2/v3/certs"

	appleKeysTTL        = time.Hour
	appleKeysMinRefresh = time.Minute
)

// OIDCIdentity is what a verified OpenID Connect identity token proves.
type OIDCIdentity struct {
	Subject       string // the provider's stable user id
	Email         string // may be empty, or a private-relay address
	EmailVerified bool
	Name          string // Google only; Apple never puts the name in the token
}

// AppleIdentity is kept as the name Apple-specific callers use.
type AppleIdentity = OIDCIdentity

// OIDCVerifier verifies RS256 identity tokens (Sign in with Apple, Google
// Sign-In) against the provider's published public keys. Without this, the
// social-login endpoints would trust whatever id/email the client claims.
type OIDCVerifier struct {
	audience string   // the client/bundle id the token must be issued to
	issuers  []string // accepted `iss` values
	keysURL  string
	client   *http.Client

	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// AppleVerifier is the OIDCVerifier configured for Apple.
type AppleVerifier = OIDCVerifier

func newOIDCVerifier(audience, keysURL string, issuers ...string) *OIDCVerifier {
	return &OIDCVerifier{
		audience: audience,
		issuers:  issuers,
		keysURL:  keysURL,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// NewAppleVerifier returns a verifier accepting Apple tokens issued to
// audience (the iOS bundle id, e.g. com.spotto.spotto).
func NewAppleVerifier(audience string) *OIDCVerifier {
	return newOIDCVerifier(audience, appleKeysURL, appleIssuer)
}

// NewGoogleVerifier returns a verifier accepting Google ID tokens issued to
// audience (the Web OAuth client id the app passes as serverClientId).
func NewGoogleVerifier(audience string) *OIDCVerifier {
	return newOIDCVerifier(audience, googleKeysURL, "https://accounts.google.com", "accounts.google.com")
}

// Verify checks signature, issuer, audience and expiry, and returns the
// verified identity. Any failure means the token must not be trusted.
func (v *OIDCVerifier) Verify(ctx context.Context, rawToken string) (OIDCIdentity, error) {
	if rawToken == "" {
		return OIDCIdentity{}, errors.New("identity token required")
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
		jwt.WithAudience(v.audience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return OIDCIdentity{}, fmt.Errorf("invalid identity token: %w", err)
	}
	iss, _ := claims["iss"].(string)
	if !slices.Contains(v.issuers, iss) {
		return OIDCIdentity{}, errors.New("identity token has an unexpected issuer")
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return OIDCIdentity{}, errors.New("identity token has no subject")
	}
	id := OIDCIdentity{Subject: sub}
	id.Email, _ = claims["email"].(string)
	id.Name, _ = claims["name"].(string)
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
func (v *OIDCVerifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
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
	return nil, errors.New("unknown signing key")
}

func (v *OIDCVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.keysURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch signing keys: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch signing keys: status %d", res.StatusCode)
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
		return nil, fmt.Errorf("decode signing keys: %w", err)
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
		return nil, errors.New("no usable signing keys returned")
	}
	return keys, nil
}
