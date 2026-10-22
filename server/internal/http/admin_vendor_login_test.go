package httpapi_test

// Admin hand-over of a vendor login (admin_vendor_login.go): who may do
// it, which accounts it can touch, and that each mode really cuts off the
// previous holder. Same real-Postgres pattern as validate_test.go; it skips
// when it can't connect.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestAdminVendorLoginHandover(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	cfg := config.Config{
		CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32),
		VendorAppURL: "http://localhost:3000", SuperAdminEmails: []string{"super@vlhtest.dev"},
	}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-"+t.Name()), cfg).Router(cfg))
	t.Cleanup(srv.Close)

	purge := func() {
		store.Pool.Exec(ctx, `DELETE FROM admin_audit_log WHERE target_id IN ('ven_vlh_a','usr_vlh_admin')`)
		store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_vlh_%'`)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id IN ('ven_vlh_a','ven_vlh_b')`)
	}
	purge()
	t.Cleanup(purge)

	cats, err := store.GetVendorCategories(ctx)
	if err != nil || len(cats) == 0 {
		t.Skipf("skipping: no vendor categories in the local DB (%v)", err)
	}
	now := time.Now()
	for _, id := range []string{"ven_vlh_a", "ven_vlh_b"} {
		must(t, store.UpsertVendor(ctx, domain.Vendor{ID: id, Name: "Handover " + id, Category: cats[0].ID, Town: "Ella",
			Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}))
	}
	pw, _ := auth.HashPassword("testpass123")
	venA, venB := "ven_vlh_a", "ven_vlh_b"
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_vlh_super", Email: "super@vlhtest.dev", PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_vlh_admin", Email: "admin@vlhtest.dev", PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_vlh_owner", Email: "owner@vlhtest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &venA, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_vlh_other", Email: "other@vlhtest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &venB, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))

	superTok := loginAs(t, srv, "super@vlhtest.dev", "testpass123")
	adminTok := loginAs(t, srv, "admin@vlhtest.dev", "testpass123")
	ownerTok := loginAs(t, srv, "owner@vlhtest.dev", "testpass123")

	handover := func(tok, vendorID, userID string, body map[string]any) (int, map[string]any) {
		var out map[string]any
		code := doReqJSON(t, srv, http.MethodPost, "/admin/vendors/"+vendorID+"/members/"+userID+"/login", tok, body, &out)
		return code, out
	}
	reason := "Real owner verified in person"

	// A plain admin can't, until a super-admin grants it.
	if code, _ := handover(adminTok, venA, "usr_vlh_owner", map[string]any{"mode": "password", "password": "Brand-new-pass-1", "reason": reason}); code != http.StatusForbidden {
		t.Fatalf("ungranted admin: want 403, got %d", code)
	}
	if code := doReqJSON(t, srv, http.MethodPut, "/admin/users/usr_vlh_admin/permissions", adminTok, map[string]any{"vendorLogins": true, "reason": reason}, nil); code != http.StatusForbidden {
		t.Fatalf("admin granting itself: want 403, got %d", code)
	}
	if code := doReqJSON(t, srv, http.MethodPut, "/admin/users/usr_vlh_admin/permissions", superTok, map[string]any{"vendorLogins": true, "reason": reason}, nil); code != http.StatusOK {
		t.Fatalf("super-admin grant: want 200, got %d", code)
	}

	// Wrong vendor, or a non-vendor account, is never reachable.
	if code, _ := handover(adminTok, venA, "usr_vlh_other", map[string]any{"mode": "none", "email": "x@vlhtest.dev", "reason": reason}); code != http.StatusNotFound {
		t.Errorf("other vendor's member: want 404, got %d", code)
	}
	if code, _ := handover(adminTok, venA, "usr_vlh_super", map[string]any{"mode": "none", "email": "x@vlhtest.dev", "reason": reason}); code != http.StatusNotFound {
		t.Errorf("admin account: want 404, got %d", code)
	}
	if code, _ := handover(adminTok, venA, "usr_vlh_owner", map[string]any{"mode": "none", "email": "other@vlhtest.dev", "reason": reason}); code != http.StatusConflict {
		t.Errorf("taken email: want 409, got %d", code)
	}
	if code, _ := handover(adminTok, venA, "usr_vlh_owner", map[string]any{"mode": "password", "password": "1234567890", "reason": reason}); code != http.StatusBadRequest {
		t.Errorf("weak password: want 400, got %d", code)
	}

	// Password mode: new email + password work, the old ones and the old
	// session don't.
	if code, _ := handover(adminTok, venA, "usr_vlh_owner", map[string]any{"mode": "password", "email": "Real.Owner@vlhtest.dev", "password": "Brand-new-pass-1", "reason": reason}); code != http.StatusOK {
		t.Fatalf("password handover: want 200, got %d", code)
	}
	if code := doReqJSON(t, srv, http.MethodGet, "/me", ownerTok, nil, nil); code != http.StatusUnauthorized {
		t.Errorf("old session: want 401, got %d", code)
	}
	loginAs(t, srv, "real.owner@vlhtest.dev", "Brand-new-pass-1")
	if code := doReqJSON(t, srv, http.MethodPost, "/auth/login", "", map[string]string{"email": "owner@vlhtest.dev", "password": "testpass123"}, nil); code == http.StatusOK {
		t.Error("old email and password still log in")
	}

	// Link mode: the password stops working, and the link sets a new one.
	code, out := handover(adminTok, venA, "usr_vlh_owner", map[string]any{"mode": "link", "email": "final@vlhtest.dev", "reason": reason})
	if code != http.StatusOK {
		t.Fatalf("link handover: want 200, got %d", code)
	}
	if code := doReqJSON(t, srv, http.MethodPost, "/auth/login", "", map[string]string{"email": "final@vlhtest.dev", "password": "Brand-new-pass-1"}, nil); code == http.StatusOK {
		t.Error("previous password still works after link hand-over")
	}
	link, _ := out["setupLink"].(string) // no SMTP in tests, so the link comes back
	u, err := url.Parse(link)
	if err != nil || u.Query().Get("resetToken") == "" {
		t.Fatalf("want a setup link without SMTP, got %q", link)
	}
	if code := doReqJSON(t, srv, http.MethodPost, "/auth/reset-password", "", map[string]string{"token": u.Query().Get("resetToken"), "newPassword": "Owner-chosen-pass-9"}, nil); code != http.StatusOK {
		t.Fatalf("set password via link: want 200, got %d", code)
	}
	loginAs(t, srv, "final@vlhtest.dev", "Owner-chosen-pass-9")

	// Revoking the grant takes the power away again.
	must(t, store.SetAdminPermission(ctx, "usr_vlh_admin", httpapi.PermVendorLogins, false))
	if code, _ := handover(adminTok, venA, "usr_vlh_owner", map[string]any{"mode": "none", "email": "again@vlhtest.dev", "reason": reason}); code != http.StatusForbidden {
		t.Errorf("revoked admin: want 403, got %d", code)
	}

	// The audit trail records the hand-over but never a password.
	var leaked bool
	must(t, store.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM admin_audit_log WHERE target_id='ven_vlh_a' AND metadata::text LIKE '%Brand-new-pass%')`).Scan(&leaked))
	if leaked {
		t.Error("password written to the audit log")
	}
}
