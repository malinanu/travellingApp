package auth

import (
	"testing"

	"github.com/spotto/server/internal/domain"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("spotto123")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "spotto123") {
		t.Error("correct password rejected")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("wrong password accepted")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	tokens := NewTokens("test-secret")
	vendorID := "ven_driftwood"
	u := domain.User{ID: "usr_1", Role: domain.RoleVendor, VendorID: &vendorID}

	tok, err := tokens.Issue(u)
	if err != nil {
		t.Fatal(err)
	}
	c, err := tokens.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Subject != "usr_1" || c.Role != domain.RoleVendor || c.VendorID != "ven_driftwood" {
		t.Errorf("claims mismatch: %+v", c)
	}

	if _, err := NewTokens("other-secret").Verify(tok); err == nil {
		t.Error("token verified with wrong secret")
	}
}

func TestEncryptSecretRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	plaintext := "JBSWY3DPEHPK3PXP" // a realistic base32 TOTP secret

	encrypted, err := EncryptSecret(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == plaintext {
		t.Fatal("EncryptSecret returned the plaintext unchanged")
	}

	decrypted, err := DecryptSecret(key, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != plaintext {
		t.Errorf("round-trip mismatch: got %q, want %q", decrypted, plaintext)
	}

	// Two encryptions of the same plaintext must differ (random nonce) —
	// otherwise identical secrets would be distinguishable by ciphertext.
	encrypted2, err := EncryptSecret(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == encrypted2 {
		t.Error("two encryptions of the same plaintext produced identical ciphertext (nonce reuse?)")
	}

	// Decrypting with the wrong key must fail, not silently return garbage.
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(255 - i)
	}
	if _, err := DecryptSecret(wrongKey, encrypted); err == nil {
		t.Error("decrypted successfully with the wrong key")
	}
}

func TestDecryptSecretEmptyString(t *testing.T) {
	key := make([]byte, 32)
	got, err := DecryptSecret(key, "")
	if err != nil || got != "" {
		t.Errorf("expected (\"\", nil) for empty ciphertext, got (%q, %v)", got, err)
	}
}
