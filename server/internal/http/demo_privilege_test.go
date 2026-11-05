package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func purgeDemoPrivilegeTestData(store *postgres.Store) {
	ctx := context.Background()
	store.Pool.Exec(ctx, `DELETE FROM admin_audit_log WHERE actor_id LIKE 'usr_demo_%' OR target_id LIKE 'usr_demo_%'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_demo_%'`)
}

func TestAdminDemoPrivilege(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()
	purgeDemoPrivilegeTestData(store)
	t.Cleanup(func() { purgeDemoPrivilegeTestData(store) })

	pw, _ := auth.HashPassword("testpass123")
	for _, u := range []domain.User{
		{ID: "usr_demo_admin", Email: "admin@demotest.dev", Role: domain.RoleAdmin},
		{ID: "usr_demo_dev", Email: "dev@demotest.dev", Role: domain.RoleTraveller},
	} {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		must(t, store.CreateUser(ctx, u))
	}

	srv, _ := newMediaTestServer(t, store, true)
	t.Cleanup(srv.Close)
	admin := loginAs(t, srv, "admin@demotest.dev", "testpass123")
	dev := loginAs(t, srv, "dev@demotest.dev", "testpass123")

	call := func(tok, method, path string, body any) (int, map[string]any) {
		t.Helper()
		var out map[string]any
		code := doReqJSON(t, srv, method, path, tok, body, &out)
		return code, out
	}

	// 1. Non-admin cannot grant demo privilege (403)
	code, _ := call(dev, http.MethodPut, "/admin/users/usr_demo_dev/demo-privilege", map[string]any{
		"demoPrivilege": true,
		"reason":        "self grant attempt",
	})
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-admin, got %d", code)
	}

	// 2. Admin cannot grant without a reason (400)
	code, _ = call(admin, http.MethodPut, "/admin/users/usr_demo_dev/demo-privilege", map[string]any{
		"demoPrivilege": true,
	})
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request without reason, got %d", code)
	}

	// 3. Admin grants demo privilege
	code, out := call(admin, http.MethodPut, "/admin/users/usr_demo_dev/demo-privilege", map[string]any{
		"demoPrivilege": true,
		"location":      "mirissa",
		"reason":        "staging investor pitch",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200 OK granting demo privilege, got %d (%v)", code, out)
	}
	if out["demoPrivilege"] != true || out["location"] != "mirissa" {
		t.Fatalf("unexpected response body: %v", out)
	}

	// 4. Check audit log
	var audited int
	store.Pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_log WHERE action='user.demo_privilege' AND target_id='usr_demo_dev'`).Scan(&audited)
	if audited != 1 {
		t.Errorf("expected 1 audit log entry, got %d", audited)
	}

	// 5. Developer checks /me
	var me map[string]any
	doReqJSON(t, srv, http.MethodGet, "/me", dev, nil, &me)
	if me["demoPrivilege"] != true {
		t.Errorf("expected me[demoPrivilege] == true, got %v", me["demoPrivilege"])
	}
	if me["demoLocation"] != "mirissa" {
		t.Errorf("expected me[demoLocation] == 'mirissa', got %v", me["demoLocation"])
	}

	// 6. SearchUsers includes demo_privilege
	summaries, err := store.SearchUsers(ctx, "dev@demotest.dev", 10)
	if err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}
	if len(summaries) == 0 || !summaries[0].DemoPrivilege {
		t.Errorf("expected SearchUsers to show DemoPrivilege=true, got summaries: %+v", summaries)
	}

	// 7. Revoke demo privilege
	code, out = call(admin, http.MethodPut, "/admin/users/usr_demo_dev/demo-privilege", map[string]any{
		"demoPrivilege": false,
		"reason":        "pitch ended",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200 OK revoking demo privilege, got %d (%v)", code, out)
	}

	doReqJSON(t, srv, http.MethodGet, "/me", dev, nil, &me)
	if me["demoPrivilege"] == true {
		t.Errorf("expected me[demoPrivilege] == false after revocation, got %v", me["demoPrivilege"])
	}
}

