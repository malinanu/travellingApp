package httpapi_test

// Integration test for the vendor validate endpoint's edge cases against the
// real Postgres store (the domain has no mockable interface — see store.go's
// decision to skip per-entity interfaces since the project is Postgres-only).
// Requires the local shared-postgres container with a `spotto` database;
// skips automatically if it can't connect.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/store/postgres"
)

func mustConnect(t *testing.T) *postgres.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	store, err := postgres.Connect(ctx, "postgres://postgres:admin123@localhost:5432/spotto?sslmode=disable")
	if err != nil {
		t.Skipf("skipping: no local postgres available: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func loginAs(t *testing.T, srv *httptest.Server, email, password string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	res, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d", res.StatusCode)
	}
	var out struct {
		Token string `json:"token"`
	}
	json.NewDecoder(res.Body).Decode(&out)
	return out.Token
}

func validate(t *testing.T, srv *httptest.Server, token, code string) map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"code": code})
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/vendor/validate", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return out
}

// TestValidateEdgeCases covers unknown / already_redeemed / wrong_vendor,
// seeding two vendors and two claims directly against the store so the test
// is independent of cmd/seed's dataset.
func TestValidateEdgeCases(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	testTOTPKey := make([]byte, 32) // zero key is fine for this test — no 2FA flows exercised here
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: testTOTPKey, VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")

	vendorA := domain.Vendor{ID: "ven_test_a", Name: "Vendor A", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	vendorB := domain.Vendor{ID: "ven_test_b", Name: "Vendor B", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendorA))
	must(t, store.UpsertVendor(ctx, vendorB))

	must(t, store.CreateUser(ctx, domain.User{ID: "usr_test_vendor_a", Email: "vendora@edgecase.test", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendorA.ID, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_test_traveller", Email: "traveller@edgecase.test", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))

	unredeemed := domain.Claim{ID: "clm_test_unredeemed", Code: "SPOTTO-TEST1", VendorID: &vendorA.ID, TravellerID: "usr_test_traveller", Status: domain.ClaimClaimed, ClaimedAt: now}
	otherVendor := domain.Claim{ID: "clm_test_other", Code: "SPOTTO-TEST2", VendorID: &vendorB.ID, TravellerID: "usr_test_traveller", Status: domain.ClaimClaimed, ClaimedAt: now}
	must(t, store.CreateClaim(ctx, unredeemed))
	must(t, store.CreateClaim(ctx, otherVendor))

	vendorAToken := loginAs(t, srv, "vendora@edgecase.test", "testpass123")

	t.Run("unknown code", func(t *testing.T) {
		out := validate(t, srv, vendorAToken, "SPOTTO-NOPE")
		if out["result"] != "unknown" {
			t.Errorf("want unknown, got %v", out["result"])
		}
	})

	t.Run("wrong vendor", func(t *testing.T) {
		out := validate(t, srv, vendorAToken, "SPOTTO-TEST2")
		if out["result"] != "wrong_vendor" {
			t.Errorf("want wrong_vendor, got %v", out["result"])
		}
	})

	t.Run("valid then already redeemed", func(t *testing.T) {
		first := validate(t, srv, vendorAToken, "SPOTTO-TEST1")
		if first["result"] != "valid" {
			t.Fatalf("want valid, got %v", first["result"])
		}
		second := validate(t, srv, vendorAToken, "SPOTTO-TEST1")
		if second["result"] != "already_redeemed" {
			t.Errorf("want already_redeemed, got %v", second["result"])
		}
		if second["redeemedAt"] == nil {
			t.Error("already_redeemed response missing redeemedAt")
		}
	})

	// cleanup so repeat runs stay idempotent
	store.Pool.Exec(ctx, `DELETE FROM claims WHERE id IN ('clm_test_unredeemed','clm_test_other')`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id IN ('usr_test_vendor_a','usr_test_traveller')`)
	store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id IN ('ven_test_a','ven_test_b')`)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
