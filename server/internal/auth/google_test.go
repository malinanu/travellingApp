package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestGoogleVerifierIssuers(t *testing.T) {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := newTestVerifier(t, &key.PublicKey)
	v.issuers = NewGoogleVerifier(testAud).issuers
	ctx := context.Background()

	for _, iss := range []string{"https://accounts.google.com", "accounts.google.com"} {
		c := validClaims()
		c["iss"] = iss
		c["name"] = "Ada Lovelace"
		c["email_verified"] = true
		id, err := v.Verify(ctx, signToken(t, key, "k1", c))
		if err != nil {
			t.Fatalf("issuer %q: %v", iss, err)
		}
		if id.Name != "Ada Lovelace" || !id.EmailVerified {
			t.Errorf("issuer %q: unexpected identity %+v", iss, id)
		}
	}

	// A token Apple issued must not be accepted as a Google token.
	if _, err := v.Verify(ctx, signToken(t, key, "k1", jwt.MapClaims{
		"iss": appleIssuer, "aud": testAud, "sub": "x", "exp": validClaims()["exp"],
	})); err == nil {
		t.Fatal("apple-issued token accepted by the google verifier")
	}
}
