package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testAud = "com.spotto.spotto"

func newTestVerifier(t *testing.T, key *rsa.PublicKey) *AppleVerifier {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": "k1", "kty": "RSA",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	t.Cleanup(srv.Close)
	v := NewAppleVerifier(testAud)
	v.keysURL = srv.URL
	return v
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss": appleIssuer, "aud": testAud, "sub": "001234.abcd",
		"email": "a@b.com", "email_verified": "true",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
}

func TestAppleVerifier(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newTestVerifier(t, &key.PublicKey)
	ctx := context.Background()

	t.Run("accepts a valid token", func(t *testing.T) {
		id, err := v.Verify(ctx, signToken(t, key, "k1", validClaims()))
		if err != nil {
			t.Fatal(err)
		}
		if id.Subject != "001234.abcd" || id.Email != "a@b.com" || !id.EmailVerified {
			t.Errorf("unexpected identity: %+v", id)
		}
	})

	reject := map[string]func() string{
		"wrong audience": func() string { c := validClaims(); c["aud"] = "com.evil.app"; return signToken(t, key, "k1", c) },
		"wrong issuer":   func() string { c := validClaims(); c["iss"] = "https://evil.example"; return signToken(t, key, "k1", c) },
		"expired":        func() string { c := validClaims(); c["exp"] = time.Now().Add(-time.Hour).Unix(); return signToken(t, key, "k1", c) },
		"no expiry":      func() string { c := validClaims(); delete(c, "exp"); return signToken(t, key, "k1", c) },
		"no subject":     func() string { c := validClaims(); delete(c, "sub"); return signToken(t, key, "k1", c) },
		"bad signature":  func() string { return signToken(t, other, "k1", validClaims()) },
		"unknown kid":    func() string { return signToken(t, key, "nope", validClaims()) },
		"alg none": func() string {
			s, _ := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims()).SignedString(jwt.UnsafeAllowNoneSignatureType)
			return s
		},
		"empty":   func() string { return "" },
		"garbage": func() string { return "not.a.jwt" },
	}
	for name, mk := range reject {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, err := v.Verify(ctx, mk()); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
