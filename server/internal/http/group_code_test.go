package httpapi_test

// PUT /me/group must never let two groups share a code: the code keys the
// live-location socket, and older app builds gave every user the same
// sample code. Same real-Postgres pattern as validate_test.go; it skips
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

func TestPutGroupKeepsCodesUnique(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	purge := func() {
		store.Pool.Exec(ctx, `DELETE FROM groups WHERE owner_id LIKE 'usr_grpcode_%'`)
		store.Pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@grpcodetest.dev'`)
	}
	purge()
	t.Cleanup(purge)

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	for _, id := range []string{"usr_grpcode_a", "usr_grpcode_b"} {
		must(t, store.CreateUser(ctx, domain.User{ID: id, Email: id + "@grpcodetest.dev", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	}
	tokA := loginAs(t, srv, "usr_grpcode_a@grpcodetest.dev", "testpass123")
	tokB := loginAs(t, srv, "usr_grpcode_b@grpcodetest.dev", "testpass123")

	put := func(token, code string) string {
		t.Helper()
		status, out := doJSON(t, http.MethodPut, srv.URL+"/me/group", token,
			map[string]any{"name": "Spotto Crew", "code": code, "memberIds": []string{}})
		if status != http.StatusOK {
			t.Fatalf("PUT /me/group: want 200, got %d %v", status, out)
		}
		got, _ := out["code"].(string)
		return got
	}

	// A's own code is kept, also on a second save.
	if got := put(tokA, "GRPCODETEST"); got != "GRPCODETEST" {
		t.Fatalf("first owner: want GRPCODETEST, got %q", got)
	}
	if got := put(tokA, "GRPCODETEST"); got != "GRPCODETEST" {
		t.Fatalf("re-save by owner: want GRPCODETEST, got %q", got)
	}

	// B sending A's code gets a fresh one, and can't open A's socket.
	codeB := put(tokB, "GRPCODETEST")
	if codeB == "GRPCODETEST" || !strings.HasPrefix(codeB, "GRP-") {
		t.Fatalf("taken code: want a fresh GRP- code, got %q", codeB)
	}
	g, err := store.GroupByOwner(ctx, "usr_grpcode_b")
	must(t, err)
	if g.Code != codeB {
		t.Errorf("stored code %q, response said %q", g.Code, codeB)
	}
	h := http.Header{}
	h.Set("Authorization", "Bearer "+tokB)
	conn, res, _ := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws/group?groupId=GRPCODETEST", h)
	if conn != nil {
		conn.Close()
	}
	if res == nil || res.StatusCode != http.StatusForbidden {
		t.Errorf("B on A's socket: want 403, got %v", res)
	}

	// An empty code gets one too.
	if got := put(tokB, ""); !strings.HasPrefix(got, "GRP-") {
		t.Errorf("empty code: want a GRP- code, got %q", got)
	}

	// An older app build saving its made-up sample group gets it cleaned.
	status, out := doJSON(t, http.MethodPut, srv.URL+"/me/group", tokB,
		map[string]any{"name": "Weligama Crew", "code": "WLG-4K2", "memberIds": []string{"mia", "theo", "ines", "usr_grpcode_a"}})
	if status != http.StatusOK {
		t.Fatalf("sample group: want 200, got %d %v", status, out)
	}
	g, err = store.GroupByOwner(ctx, "usr_grpcode_b")
	must(t, err)
	if g.Name != "Spotto Crew" || g.Code == "WLG-4K2" || !strings.HasPrefix(g.Code, "GRP-") ||
		len(g.MemberIDs) != 1 || g.MemberIDs[0] != "usr_grpcode_a" {
		t.Errorf("sample group not cleaned: %+v", g)
	}
}
