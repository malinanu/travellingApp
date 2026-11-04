package httpapi_test

import (
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
)

func TestClaimStatusAndMyClaims(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	testTOTPKey := make([]byte, 32)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: testTOTPKey, VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")

	vendor := domain.Vendor{ID: "ven_status_test", Name: "Status Vendor", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendor))

	place := domain.Place{ID: "plc_status_test", Name: "Status Spot", VendorID: &vendor.ID, Data: []byte(`{"category":"cafe"}`)}
	must(t, store.UpsertPlace(ctx, place))

	must(t, store.CreateUser(ctx, domain.User{
		ID:           "usr_status_traveller",
		Email:        "statustraveller@test.com",
		PasswordHash: pw,
		Role:         domain.RoleTraveller,
		CreatedAt:    now,
	}))

	claimActive := domain.Claim{
		ID:          "clm_status_active",
		PlaceID:     place.ID,
		Code:        "SPOTTO-STAT1",
		VendorID:    &vendor.ID,
		TravellerID: "usr_status_traveller",
		Status:      domain.ClaimClaimed,
		ClaimedAt:   now,
	}
	claimRedeemed := domain.Claim{
		ID:          "clm_status_redeemed",
		PlaceID:     "plc_status_other",
		Code:        "SPOTTO-STAT2",
		VendorID:    &vendor.ID,
		TravellerID: "usr_status_traveller",
		Status:      domain.ClaimRedeemed,
		ClaimedAt:   now,
		RedeemedAt:  &now,
	}
	must(t, store.CreateClaim(ctx, claimActive))
	must(t, store.CreateClaim(ctx, claimRedeemed))

	defer func() {
		store.Pool.Exec(ctx, `DELETE FROM claims WHERE id IN ('clm_status_active','clm_status_redeemed')`)
		store.Pool.Exec(ctx, `DELETE FROM places WHERE id = 'plc_status_test'`)
		store.Pool.Exec(ctx, `DELETE FROM users WHERE id = 'usr_status_traveller'`)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id = 'ven_status_test'`)
	}()

	travellerToken := loginAs(t, srv, "statustraveller@test.com", "testpass123")

	t.Run("claim status not found", func(t *testing.T) {
		res, err := http.Get(srv.URL + "/claims/SPOTTO-NONEXISTENT/status")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusNotFound {
			t.Fatalf("want 404, got %d", res.StatusCode)
		}
	})

	t.Run("claim status active", func(t *testing.T) {
		res, err := http.Get(srv.URL + "/claims/SPOTTO-STAT1/status")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", res.StatusCode)
		}
		var out map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		if out["status"] != "claimed" {
			t.Fatalf("want status claimed, got %v", out["status"])
		}
		if out["code"] != "SPOTTO-STAT1" {
			t.Fatalf("want code SPOTTO-STAT1, got %v", out["code"])
		}
	})

	t.Run("claim status redeemed", func(t *testing.T) {
		res, err := http.Get(srv.URL + "/claims/SPOTTO-STAT2/status")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", res.StatusCode)
		}
		var out map[string]any
		json.NewDecoder(res.Body).Decode(&out)
		if out["status"] != "redeemed" {
			t.Fatalf("want status redeemed, got %v", out["status"])
		}
		if out["redeemedAt"] == nil {
			t.Fatal("expected non-nil redeemedAt")
		}
	})

	t.Run("my claims returns only active unredeemed claims", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/me/claims", nil)
		req.Header.Set("Authorization", "Bearer "+travellerToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("want 200, got %d", res.StatusCode)
		}
		var claims []domain.Claim
		json.NewDecoder(res.Body).Decode(&claims)
		if len(claims) != 1 {
			t.Fatalf("expected 1 active claim, got %d", len(claims))
		}
		if claims[0].Code != "SPOTTO-STAT1" {
			t.Fatalf("expected claim SPOTTO-STAT1, got %s", claims[0].Code)
		}
		if claims[0].Status != domain.ClaimClaimed {
			t.Fatalf("expected claim status claimed, got %s", claims[0].Status)
		}
	})
}
