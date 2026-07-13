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
