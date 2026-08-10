package httpapi_test

// Integration tests for the temporary Business-Profile access grant system
// (vendor_grants.go): create/redeem/expiry/revoke/rate-limit, plus admin
// extend/shorten/revoke. Same real-Postgres pattern as validate_test.go;
// skips automatically if it can't connect.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

// doReqJSON is doReq but also decodes the JSON response body.
func doReqJSON(t *testing.T, srv *httptest.Server, method, path, token string, body any, out any) int {
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
	if out != nil {
		json.NewDecoder(res.Body).Decode(out)
	}
	return res.StatusCode
}

// TestAdminVendorGrantsNeverNull guards the /access-grants crash: an empty
// grant list must serialise to [] (not null), or the admin page's
// grants.filter() throws "Cannot read properties of null".
func TestAdminVendorGrantsNeverNull(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_grantsnull_admin", Email: "admin@grantsnull.dev", PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: now}))
	t.Cleanup(func() { store.Pool.Exec(ctx, `DELETE FROM users WHERE id = 'usr_grantsnull_admin'`) })
	adminToken := loginAs(t, srv, "admin@grantsnull.dev", "testpass123")

	// A status value no grant ever computes to keeps the result empty
	// regardless of what the shared DB holds, so this exercises the empty
	// path specifically.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/vendor-grants?status=__none__", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	res, err := http.DefaultClient.Do(req)
	must(t, err)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("want 200, got %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	got := strings.TrimSpace(string(body))
	if got == "null" {
		t.Fatal("empty grant list serialised to null â€” this crashes the /access-grants page")
	}
	if got != "[]" {
		t.Errorf("want [], got %s", got)
	}
}

func TestVendorProfileGrants(t *testing.T) {
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
	vendor := domain.Vendor{ID: "ven_grants_test", Name: "Grants Test Vendor", Category: "cafe", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendor))

	must(t, store.CreateUser(ctx, domain.User{ID: "usr_grants_owner", Email: "owner@grantstest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_grants_manager1", Email: "manager1@grantstest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleManager, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_grants_manager2", Email: "manager2@grantstest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleManager, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_grants_admin", Email: "admin@grantstest.dev", PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: now}))

	ownerToken := loginAs(t, srv, "owner@grantstest.dev", "testpass123")
	manager1Token := loginAs(t, srv, "manager1@grantstest.dev", "testpass123")
	manager2Token := loginAs(t, srv, "manager2@grantstest.dev", "testpass123")
	adminToken := loginAs(t, srv, "admin@grantstest.dev", "testpass123")

	identityBody := map[string]any{"name": "Grants Test Vendor", "town": "Weligama"}

	// createGrant creates a fresh grant and registers its revocation as test
	// cleanup â€” ActiveVendorProfileGrant checks "does *any* active grant
	// exist for this grantee", so a grant left active by an earlier subtest
	// would otherwise leak through and mask what a later subtest is
	// actually testing.
	createGrant := func(t *testing.T, granteeID string) (id, pin string) {
		t.Helper()
		var out struct {
			ID  string `json:"id"`
			Pin string `json:"pin"`
		}
		status := doReqJSON(t, srv, http.MethodPost, "/vendor/profile-grants", ownerToken,
			map[string]any{"granteeId": granteeID, "durationMins": 15, "deliver": "copy"}, &out)
		if status != http.StatusCreated {
			t.Fatalf("create grant: want 201, got %d", status)
		}
		t.Cleanup(func() {
			doReq(t, srv, http.MethodDelete, "/vendor/profile-grants/"+out.ID, ownerToken, nil)
		})
		return out.ID, out.Pin
	}

	t.Run("manager2 cannot redeem manager1's grant", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager2Token, map[string]any{"pin": pin})
		if got != http.StatusForbidden {
			t.Errorf("want 403, got %d", got)
		}
	})

	t.Run("wrong PIN rejected, correct PIN redeems and unlocks writes", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")

		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", manager1Token, identityBody); got != http.StatusForbidden {
			t.Fatalf("expected locked before redemption, got %d", got)
		}

		if got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": "000000"}); got != http.StatusForbidden {
			t.Errorf("wrong pin: want 403, got %d", got)
		}

		var redeemOut struct {
			AccessExpiresAt time.Time `json:"accessExpiresAt"`
		}
		if got := doReqJSON(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin}, &redeemOut); got != http.StatusOK {
			t.Fatalf("correct pin: want 200, got %d", got)
		}
		if !redeemOut.AccessExpiresAt.After(time.Now()) {
			t.Errorf("accessExpiresAt should be in the future, got %v", redeemOut.AccessExpiresAt)
		}

		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", manager1Token, identityBody); got != http.StatusOK {
			t.Errorf("expected unlocked after redemption, got %d", got)
		}
	})

	t.Run("expired PIN rejected even with correct value", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		store.Pool.Exec(ctx, `UPDATE vendor_profile_grants SET pin_expires_at = now() - interval '1 minute' WHERE id=$1`, id)
		got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin})
		if got != http.StatusForbidden {
			t.Errorf("want 403, got %d", got)
		}
	})

	t.Run("access-expired grant blocks writes", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		if got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin}); got != http.StatusOK {
			t.Fatalf("redeem: want 200, got %d", got)
		}
		store.Pool.Exec(ctx, `UPDATE vendor_profile_grants SET access_expires_at = now() - interval '1 minute' WHERE id=$1`, id)
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", manager1Token, identityBody); got != http.StatusForbidden {
			t.Errorf("want 403 after access expiry, got %d", got)
		}
	})

	t.Run("owner revoke blocks a previously active grant", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		if got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin}); got != http.StatusOK {
			t.Fatalf("redeem: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", manager1Token, identityBody); got != http.StatusOK {
			t.Fatalf("expected unlocked, got %d", got)
		}
		if got := doReq(t, srv, http.MethodDelete, "/vendor/profile-grants/"+id, ownerToken, nil); got != http.StatusOK {
			t.Fatalf("revoke: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", manager1Token, identityBody); got != http.StatusForbidden {
			t.Errorf("want 403 after revoke, got %d", got)
		}
	})

	t.Run("admin extend pushes access_expires_at forward", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		if got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin}); got != http.StatusOK {
			t.Fatalf("redeem: want 200, got %d", got)
		}
		var before time.Time
		store.Pool.QueryRow(ctx, `SELECT access_expires_at FROM vendor_profile_grants WHERE id=$1`, id).Scan(&before)

		if got := doReq(t, srv, http.MethodPost, "/admin/vendor-grants/"+id+"/extend", adminToken, map[string]any{"minutes": 30}); got != http.StatusOK {
			t.Fatalf("extend: want 200, got %d", got)
		}
		var after time.Time
		store.Pool.QueryRow(ctx, `SELECT access_expires_at FROM vendor_profile_grants WHERE id=$1`, id).Scan(&after)
		if !after.After(before) {
			t.Errorf("expected access_expires_at to move forward: before=%v after=%v", before, after)
		}
	})

	t.Run("admin shorten a pending PIN causes it to expire immediately", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		if got := doReq(t, srv, http.MethodPost, "/admin/vendor-grants/"+id+"/extend", adminToken, map[string]any{"minutes": -100000}); got != http.StatusOK {
			t.Fatalf("shorten: want 200, got %d", got)
		}
		got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin})
		if got != http.StatusForbidden {
			t.Errorf("want 403 after admin shorten, got %d", got)
		}
	})

	t.Run("admin revoke blocks an active grant", func(t *testing.T) {
		id, pin := createGrant(t, "usr_grants_manager1")
		if got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": pin}); got != http.StatusOK {
			t.Fatalf("redeem: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPost, "/admin/vendor-grants/"+id+"/revoke", adminToken, nil); got != http.StatusOK {
			t.Fatalf("admin revoke: want 200, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", manager1Token, identityBody); got != http.StatusForbidden {
			t.Errorf("want 403 after admin revoke, got %d", got)
		}
	})

	t.Run("rate limited after 5 wrong-PIN attempts", func(t *testing.T) {
		id, _ := createGrant(t, "usr_grants_manager1")
		for i := 0; i < 5; i++ {
			got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": "111111"})
			if got != http.StatusForbidden {
				t.Fatalf("attempt %d: want 403, got %d", i+1, got)
			}
		}
		got := doReq(t, srv, http.MethodPost, "/vendor/profile-grants/"+id+"/redeem", manager1Token, map[string]any{"pin": "111111"})
		if got != http.StatusTooManyRequests {
			t.Errorf("6th attempt: want 429, got %d", got)
		}
	})

	// cleanup so repeat runs stay idempotent
	store.Pool.Exec(ctx, `DELETE FROM vendor_profile_grants WHERE vendor_id='ven_grants_test'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id IN ('usr_grants_owner','usr_grants_manager1','usr_grants_manager2','usr_grants_admin')`)
	store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id='ven_grants_test'`)
}
