package httpapi_test

// Super-admin account controls (admin_superadmin.go): changing an account
// between traveller and vendor, and permanently deleting a user or a
// business. Same real-Postgres pattern as admin_vendor_login_test.go; it
// skips when it can't connect.

import (
	"context"
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

func TestSuperAdminAccountControls(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()
	loadCategories(t, store)

	cfg := config.Config{
		CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32),
		VendorAppURL: "http://localhost:3000", SuperAdminEmails: []string{"super@sactest.dev"},
	}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-"+t.Name()), cfg).Router(cfg))
	t.Cleanup(srv.Close)

	purge := func() {
		store.Pool.Exec(ctx, `DELETE FROM admin_audit_log WHERE target_id LIKE 'usr_sac_%' OR target_id LIKE 'ven_sac_%'`)
		store.Pool.Exec(ctx, `DELETE FROM daily_quests WHERE vendor_id LIKE 'ven_sac_%'`)
		store.Pool.Exec(ctx, `DELETE FROM places WHERE id LIKE 'plc_sac_%'`)
		store.Pool.Exec(ctx, `DELETE FROM vendor_invites WHERE id LIKE 'inv_sac_%'`)
		store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_sac_%'`)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id LIKE 'ven_sac_%' OR owner = 'SAC Traveller'`)
	}
	purge()
	t.Cleanup(purge)

	now := time.Now()
	cat := "coffee"
	for _, id := range []string{"ven_sac_a", "ven_sac_b", "ven_sac_c"} {
		must(t, store.UpsertVendor(ctx, domain.Vendor{ID: id, Name: "SAC " + id, Category: cat, Town: "Ella",
			Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}))
	}
	pw, _ := auth.HashPassword("testpass123")
	venA, venB, venC := "ven_sac_a", "ven_sac_b", "ven_sac_c"
	users := []domain.User{
		{ID: "usr_sac_super", Email: "super@sactest.dev", Role: domain.RoleAdmin},
		{ID: "usr_sac_admin", Email: "admin@sactest.dev", Role: domain.RoleAdmin},
		{ID: "usr_sac_trav", Email: "trav@sactest.dev", Name: "SAC Traveller", Role: domain.RoleTraveller},
		{ID: "usr_sac_trav2", Email: "trav2@sactest.dev", Role: domain.RoleTraveller},
		{ID: "usr_sac_ownerA", Email: "ownera@sactest.dev", Role: domain.RoleVendor, VendorID: &venA, VendorRole: domain.VendorRoleOwner},
		{ID: "usr_sac_staffA", Email: "staffa@sactest.dev", Name: "Staff A", Role: domain.RoleVendor, VendorID: &venA, VendorRole: domain.VendorRoleStaff},
		{ID: "usr_sac_ownerB", Email: "ownerb@sactest.dev", Role: domain.RoleVendor, VendorID: &venB, VendorRole: domain.VendorRoleOwner},
		{ID: "usr_sac_memberC", Email: "memberc@sactest.dev", Role: domain.RoleVendor, VendorID: &venC, VendorRole: domain.VendorRoleManager},
	}
	for _, u := range users {
		u.PasswordHash, u.CreatedAt = pw, now
		must(t, store.CreateUser(ctx, u))
	}
	superTok := loginAs(t, srv, "super@sactest.dev", "testpass123")
	adminTok := loginAs(t, srv, "admin@sactest.dev", "testpass123")
	reason := "Confirmed with the owner by phone"

	post := func(tok, path string, body map[string]any) (int, map[string]any) {
		t.Helper()
		var out map[string]any
		code := doReqJSON(t, srv, http.MethodPost, path, tok, body, &out)
		return code, out
	}
	userRow := func(id string) domain.User {
		t.Helper()
		u, err := store.UserByID(ctx, id)
		if err != nil {
			t.Fatalf("load %s: %v", id, err)
		}
		return u
	}

	t.Run("only super admins, never on admin accounts", func(t *testing.T) {
		for _, path := range []string{"/admin/users/usr_sac_trav/role", "/admin/users/usr_sac_trav/delete", "/admin/vendors/ven_sac_c/delete"} {
			if code, _ := post(adminTok, path, map[string]any{"reason": reason}); code != http.StatusForbidden {
				t.Errorf("plain admin %s: want 403, got %d", path, code)
			}
		}
		if code, _ := post(superTok, "/admin/users/usr_sac_admin/role", map[string]any{"role": "traveller", "reason": reason}); code != http.StatusBadRequest {
			t.Errorf("admin target role change: want 400, got %d", code)
		}
		if code, _ := post(superTok, "/admin/users/usr_sac_admin/delete", map[string]any{"reason": reason, "confirm": "admin@sactest.dev"}); code != http.StatusBadRequest {
			t.Errorf("admin target delete: want 400, got %d", code)
		}
		if code, _ := post(superTok, "/admin/users/usr_sac_trav/role", map[string]any{"role": "vendor", "vendorId": venB, "vendorRole": "staff"}); code != http.StatusBadRequest {
			t.Errorf("missing reason: want 400, got %d", code)
		}
	})

	t.Run("traveller becomes staff of an existing business", func(t *testing.T) {
		travTok := loginAs(t, srv, "trav2@sactest.dev", "testpass123")
		code, body := post(superTok, "/admin/users/usr_sac_trav2/role", map[string]any{"role": "vendor", "vendorId": venB, "vendorRole": "staff", "reason": reason})
		if code != http.StatusOK {
			t.Fatalf("want 200, got %d %v", code, body)
		}
		u := userRow("usr_sac_trav2")
		if u.Role != domain.RoleVendor || u.VendorID == nil || *u.VendorID != venB || u.VendorRole != domain.VendorRoleStaff {
			t.Fatalf("got role=%s vendor=%v vendorRole=%s", u.Role, u.VendorID, u.VendorRole)
		}
		if code := doReqJSON(t, srv, http.MethodGet, "/me", travTok, nil, nil); code != http.StatusUnauthorized {
			t.Errorf("old traveller token: want 401, got %d", code)
		}
	})

	t.Run("traveller gets a new pending business", func(t *testing.T) {
		code, body := post(superTok, "/admin/users/usr_sac_trav/role", map[string]any{
			"role": "vendor", "reason": reason, "newBusiness": map[string]string{"name": "SAC New Cafe", "category": cat, "town": "Galle"}})
		if code != http.StatusOK {
			t.Fatalf("want 200, got %d %v", code, body)
		}
		u := userRow("usr_sac_trav")
		if u.Role != domain.RoleVendor || u.VendorID == nil || u.VendorRole != domain.VendorRoleOwner {
			t.Fatalf("got role=%s vendor=%v vendorRole=%s", u.Role, u.VendorID, u.VendorRole)
		}
		v, err := store.VendorByID(ctx, *u.VendorID)
		if err != nil || v.Status != domain.VendorPending || v.Name != "SAC New Cafe" || v.Owner != "SAC Traveller" {
			t.Fatalf("new business = %+v, %v", v, err)
		}
		if code, _ := post(superTok, "/admin/users/usr_sac_trav/role", map[string]any{"role": "vendor", "vendorId": venB, "vendorRole": "staff", "reason": reason}); code != http.StatusBadRequest {
			t.Errorf("already a vendor on a business: want 400, got %d", code)
		}
	})

	t.Run("an owner with a team needs a successor", func(t *testing.T) {
		if code, _ := post(superTok, "/admin/users/usr_sac_ownerA/role", map[string]any{"role": "traveller", "reason": reason}); code != http.StatusBadRequest {
			t.Fatalf("no new owner chosen: want 400, got %d", code)
		}
		if code, _ := post(superTok, "/admin/users/usr_sac_ownerA/role", map[string]any{"role": "traveller", "reason": reason, "newOwnerId": "usr_sac_ownerB"}); code != http.StatusBadRequest {
			t.Fatalf("new owner from another business: want 400, got %d", code)
		}
		code, body := post(superTok, "/admin/users/usr_sac_ownerA/role", map[string]any{"role": "traveller", "reason": reason, "newOwnerId": "usr_sac_staffA"})
		if code != http.StatusOK {
			t.Fatalf("want 200, got %d %v", code, body)
		}
		if u := userRow("usr_sac_ownerA"); u.Role != domain.RoleTraveller || u.VendorID != nil {
			t.Fatalf("ex-owner: role=%s vendor=%v", u.Role, u.VendorID)
		}
		if u := userRow("usr_sac_staffA"); u.VendorRole != domain.VendorRoleOwner {
			t.Fatalf("successor vendorRole = %s, want owner", u.VendorRole)
		}
		if v, _ := store.VendorByID(ctx, venA); v.Owner != "Staff A" {
			t.Errorf("vendor owner display = %q, want Staff A", v.Owner)
		}
	})

	t.Run("a sole owner leaves the business ownerless", func(t *testing.T) {
		// ven_sac_b has its owner plus trav2 (staff, added above); delete
		// trav2 first so the owner is alone.
		if code, _ := post(superTok, "/admin/users/usr_sac_trav2/delete", map[string]any{"reason": reason, "confirm": "nope@sactest.dev"}); code != http.StatusBadRequest {
			t.Fatalf("wrong confirmation: want 400, got %d", code)
		}
		if code, body := post(superTok, "/admin/users/usr_sac_trav2/delete", map[string]any{"reason": reason, "confirm": "TRAV2@sactest.dev"}); code != http.StatusOK {
			t.Fatalf("delete: want 200, got %d %v", code, body)
		}
		if _, err := store.UserByID(ctx, "usr_sac_trav2"); err == nil {
			t.Fatal("deleted user still exists")
		}
		if code, body := post(superTok, "/admin/users/usr_sac_ownerB/role", map[string]any{"role": "traveller", "reason": reason}); code != http.StatusOK {
			t.Fatalf("sole owner to traveller: want 200, got %d %v", code, body)
		}
		var entries int
		store.Pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_log WHERE (target_id='usr_sac_trav2' AND action='user.delete') OR (target_id='usr_sac_ownerB' AND action='user.role_change')`).Scan(&entries)
		if entries != 2 {
			t.Errorf("audit entries = %d, want 2", entries)
		}
	})

	t.Run("deleting a business removes it and its spot, keeps the team", func(t *testing.T) {
		memberTok := loginAs(t, srv, "memberc@sactest.dev", "testpass123")
		today := time.Now().UTC()
		execSQL(t, store, `INSERT INTO offers (id, vendor_id, title) VALUES ('off_sac_c', $1, 'Free coffee')`, venC)
		execSQL(t, store, `INSERT INTO places (id, name, vendor_id) VALUES ('plc_sac_c', 'SAC spot', $1)`, venC)
		execSQL(t, store, `INSERT INTO vendor_invites (id, vendor_id, code, created_by, expires_at) VALUES ('inv_sac_c', $1, 'SACCODE1', 'usr_sac_memberC', now() + interval '1 day')`, venC)
		execSQL(t, store, `INSERT INTO daily_quests (date, vendor_id) VALUES ($1, $2)`, today.AddDate(0, 0, 3).Format("2006-01-02"), venC)

		if code, _ := post(superTok, "/admin/vendors/ven_sac_c/delete", map[string]any{"reason": reason, "confirm": "wrong name"}); code != http.StatusBadRequest {
			t.Fatalf("wrong confirmation: want 400, got %d", code)
		}
		if code, body := post(superTok, "/admin/vendors/ven_sac_c/delete", map[string]any{"reason": reason, "confirm": "SAC ven_sac_c"}); code != http.StatusConflict {
			t.Fatalf("future quest sponsor: want 409, got %d %v", code, body)
		}
		execSQL(t, store, `UPDATE daily_quests SET date=$2 WHERE vendor_id=$1`, venC, today.AddDate(0, 0, -400).Format("2006-01-02"))

		code, body := post(superTok, "/admin/vendors/ven_sac_c/delete", map[string]any{"reason": reason, "confirm": "sac VEN_SAC_C"})
		if code != http.StatusOK {
			t.Fatalf("delete: want 200, got %d %v", code, body)
		}
		for table, id := range map[string]string{"vendors": venC, "places": "plc_sac_c", "offers": "off_sac_c", "vendor_invites": "inv_sac_c"} {
			var n int
			store.Pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE id=$1`, id).Scan(&n)
			if n != 0 {
				t.Errorf("%s %s still exists", table, id)
			}
		}
		m := userRow("usr_sac_memberC")
		if m.VendorID != nil || m.Role != domain.RoleVendor {
			t.Errorf("team member: role=%s vendor=%v, want vendor with no business", m.Role, m.VendorID)
		}
		if code := doReqJSON(t, srv, http.MethodGet, "/me", memberTok, nil, nil); code != http.StatusUnauthorized {
			t.Errorf("team member's old token: want 401, got %d", code)
		}
	})
}

func execSQL(t *testing.T, store *postgres.Store, sql string, args ...any) {
	t.Helper()
	if _, err := store.Pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
