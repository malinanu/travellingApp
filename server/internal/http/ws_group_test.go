package httpapi_test

// The group live-location socket must refuse anonymous callers and
// signed-in users who aren't in the group. It used to trust a userId
// query param, so anyone holding a group code could read members'
// positions. Same real-Postgres pattern as validate_test.go; it skips
// when it can't connect.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestGroupWSRequiresMembership(t *testing.T) {
	store := mustConnect(t)
	// A Cleanup, not a defer: cleanups run after the function returns,
	// so a deferred Close would shut the pool before the row deletes below.
	t.Cleanup(store.Close)
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	for _, id := range []string{"usr_wsg_owner", "usr_wsg_member", "usr_wsg_stranger"} {
		must(t, store.CreateUser(ctx, domain.User{ID: id, Email: id + "@wsgtest.dev", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	}
	t.Cleanup(func() {
		store.Pool.Exec(ctx, `DELETE FROM groups WHERE owner_id = 'usr_wsg_owner'`)
		store.Pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@wsgtest.dev'`)
	})
	must(t, store.UpsertGroup(ctx, domain.Group{OwnerID: "usr_wsg_owner", Name: "Trip", Code: "WSGTEST", MemberIDs: []string{"usr_wsg_member"}}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/group?groupId=WSGTEST"
	dial := func(token string) int {
		h := http.Header{}
		if token != "" {
			h.Set("Authorization", "Bearer "+token)
		}
		conn, res, err := websocket.DefaultDialer.Dial(wsURL, h)
		if conn != nil {
			conn.Close()
		}
		if res == nil {
			t.Fatalf("dial: %v", err)
		}
		return res.StatusCode
	}

	if got := dial(""); got != http.StatusUnauthorized {
		t.Errorf("no token: want 401, got %d", got)
	}
	if got := dial(loginAs(t, srv, "usr_wsg_stranger@wsgtest.dev", "testpass123")); got != http.StatusForbidden {
		t.Errorf("non-member: want 403, got %d", got)
	}
	for _, who := range []string{"usr_wsg_owner", "usr_wsg_member"} {
		if got := dial(loginAs(t, srv, who+"@wsgtest.dev", "testpass123")); got != http.StatusSwitchingProtocols {
			t.Errorf("%s: want 101, got %d", who, got)
		}
	}
}
