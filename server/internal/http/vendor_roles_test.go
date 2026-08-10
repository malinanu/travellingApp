package httpapi_test

// Integration test verifying the staff/manager/owner permission matrix is
// actually enforced â€” see requireBusinessProfileAccess (vendor.go) and
// RequireVendorManagerOrAbove (auth.go). Same real-Postgres pattern as
// validate_test.go; skips automatically if it can't connect.

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
)

// doReq makes an authenticated request and returns the status code.
func doReq(t *testing.T, srv *httptest.Server, method, path, token string, body any) int {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, srv.URL+path, reader)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	return res.StatusCode
}

func TestVendorRolePermissions(t *testing.T) {
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
	vendor := domain.Vendor{ID: "ven_roles_test", Name: "Roles Test Vendor", Category: "cafe", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendor))

	must(t, store.CreateUser(ctx, domain.User{ID: "usr_roles_owner", Email: "owner@rolestest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_roles_manager", Email: "manager@rolestest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleManager, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_roles_staff", Email: "staff@rolestest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleStaff, CreatedAt: now}))

	offer := domain.Offer{ID: "off_roles_test", VendorID: vendor.ID, Title: "Test offer", Type: "Percentage", Value: "10%", Status: domain.OfferDraft, Limit: 10}
	must(t, store.UpsertOffer(ctx, offer))

	ownerToken := loginAs(t, srv, "owner@rolestest.dev", "testpass123")
	managerToken := loginAs(t, srv, "manager@rolestest.dev", "testpass123")
	staffToken := loginAs(t, srv, "staff@rolestest.dev", "testpass123")

	t.Run("staff blocked from offer mutations", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPost, "/vendor/offers", staffToken, map[string]any{"title": "x", "type": "Percentage", "value": "5%", "limit": 5}); got != http.StatusForbidden {
			t.Errorf("POST /vendor/offers: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPatch, "/vendor/offers/"+offer.ID+"/status", staffToken, map[string]any{"status": "live"}); got != http.StatusForbidden {
			t.Errorf("PATCH offer status: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodDelete, "/vendor/offers/"+offer.ID, staffToken, nil); got != http.StatusForbidden {
			t.Errorf("DELETE offer: want 403, got %d", got)
		}
	})

	t.Run("staff blocked from menu/growth/reports", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPut, "/vendor/menu", staffToken, map[string]any{"sections": []any{}}); got != http.StatusForbidden {
			t.Errorf("PUT /vendor/menu: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodGet, "/vendor/growth", staffToken, nil); got != http.StatusForbidden {
			t.Errorf("GET /vendor/growth: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPost, "/vendor/growth/flash-sale", staffToken, map[string]any{}); got != http.StatusForbidden {
			t.Errorf("POST flash-sale: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodGet, "/vendor/reports", staffToken, nil); got != http.StatusForbidden {
			t.Errorf("GET /vendor/reports: want 403, got %d", got)
		}
	})

	t.Run("staff blocked from business profile writes", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPut, "/vendor/business", staffToken, map[string]any{"blurb": "nope"}); got != http.StatusForbidden {
			t.Errorf("PUT /vendor/business: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/category", staffToken, map[string]any{"category": "bar"}); got != http.StatusForbidden {
			t.Errorf("PATCH category: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", staffToken, map[string]any{"name": "x", "town": "y"}); got != http.StatusForbidden {
			t.Errorf("PATCH identity: want 403, got %d", got)
		}
	})

	t.Run("staff blocked from team management", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPost, "/vendor/team/invite", staffToken, map[string]any{"role": "staff"}); got != http.StatusForbidden {
			t.Errorf("POST team invite: want 403, got %d", got)
		}
	})

	t.Run("staff allowed on reads and validate", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodGet, "/vendor/offers", staffToken, nil); got != http.StatusOK {
			t.Errorf("GET /vendor/offers: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodGet, "/vendor/business", staffToken, nil); got != http.StatusOK {
			t.Errorf("GET /vendor/business: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodGet, "/vendor/dashboard", staffToken, nil); got != http.StatusOK {
			t.Errorf("GET /vendor/dashboard: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPost, "/vendor/validate", staffToken, map[string]any{"code": "SPOTTO-NOPE"}); got != http.StatusOK {
			t.Errorf("POST /vendor/validate: want 200, got %d", got)
		}
	})

	t.Run("manager allowed on offers/menu/growth mutations", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPatch, "/vendor/offers/"+offer.ID+"/status", managerToken, map[string]any{"status": "live"}); got != http.StatusOK {
			t.Errorf("PATCH offer status: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPut, "/vendor/menu", managerToken, map[string]any{"sections": []any{}}); got != http.StatusOK {
			t.Errorf("PUT /vendor/menu: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodGet, "/vendor/growth", managerToken, nil); got != http.StatusOK {
			t.Errorf("GET /vendor/growth: want 200, got %d", got)
		}
	})

	t.Run("manager blocked from team management", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPost, "/vendor/team/invite", managerToken, map[string]any{"role": "staff"}); got != http.StatusForbidden {
			t.Errorf("POST team invite: want 403, got %d", got)
		}
	})

	t.Run("manager blocked from business profile writes without a grant", func(t *testing.T) {
		got := doReq(t, srv, http.MethodPatch, "/vendor/business/category", managerToken, map[string]any{"category": "bar"})
		if got != http.StatusForbidden {
			t.Errorf("PATCH category: want 403, got %d", got)
		}
	})

	t.Run("owner allowed on everything, including business profile with no grant", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", ownerToken, map[string]any{"name": "Roles Test Vendor", "town": "Weligama"}); got != http.StatusOK {
			t.Errorf("PATCH identity: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPost, "/vendor/team/invite", ownerToken, map[string]any{"role": "staff"}); got != http.StatusCreated {
			t.Errorf("POST team invite: want 201, got %d", got)
		}
	})

	// cleanup so repeat runs stay idempotent
	store.Pool.Exec(ctx, `DELETE FROM vendor_invites WHERE vendor_id='ven_roles_test'`)
	store.Pool.Exec(ctx, `DELETE FROM offers WHERE id='off_roles_test'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id IN ('usr_roles_owner','usr_roles_manager','usr_roles_staff')`)
	store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id='ven_roles_test'`)
}
