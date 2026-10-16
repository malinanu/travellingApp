package httpapi_test

// Integration tests for refresh-token sessions against the real Postgres
// store (self-skips without a local DB, like validate_test.go).

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestRefreshSessions(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	run := time.Now().UnixNano()
	email := fmt.Sprintf("sess%d@edgecase.test", run)
	pw, _ := auth.HashPassword("testpass123")
	u := domain.User{ID: fmt.Sprintf("usr_sess_%d", run), Email: email, PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: time.Now()}
	must(t, store.CreateUser(ctx, u))

	code, login := doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{"email": email, "password": "testpass123"})
	refresh1, _ := login["refreshToken"].(string)
	if code != http.StatusOK || refresh1 == "" {
		t.Fatalf("login should return a refresh token: %d %v", code, login)
	}

	t.Run("refresh rotates the token", func(t *testing.T) {
		code, out := doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh1})
		refresh2, _ := out["refreshToken"].(string)
		if code != http.StatusOK || refresh2 == "" || refresh2 == refresh1 {
			t.Fatalf("refresh = %d %v", code, out)
		}
		if code, _ := doJSON(t, http.MethodGet, srv.URL+"/me", out["token"].(string), nil); code != http.StatusOK {
			t.Fatalf("new access token rejected: %d", code)
		}

		// The replaced token right away is a race, not theft: rejected, but
		// the session survives and the new token still works.
		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh1}); code != http.StatusUnauthorized {
			t.Fatalf("replaced token accepted: %d", code)
		}
		code, out = doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh2})
		refresh3, _ := out["refreshToken"].(string)
		if code != http.StatusOK {
			t.Fatalf("session ended by an in-grace replay: %d %v", code, out)
		}

		// Replaying an old token well after it was replaced ends the session.
		_, err := store.Pool.Exec(ctx, `UPDATE user_sessions SET rotated_at = now() - interval '1 hour' WHERE user_id = $1`, u.ID)
		must(t, err)
		doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh2})
		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh3}); code != http.StatusUnauthorized {
			t.Fatalf("session survived a late replay: %d", code)
		}
	})

	t.Run("logout ends the session", func(t *testing.T) {
		_, login := doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{"email": email, "password": "testpass123"})
		refresh := login["refreshToken"].(string)
		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/logout", "", map[string]string{"refreshToken": refresh}); code != http.StatusOK {
			t.Fatalf("logout: %d", code)
		}
		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh}); code != http.StatusUnauthorized {
			t.Fatalf("refresh after logout: %d", code)
		}
	})

	t.Run("admin sign-out-everywhere ends every token", func(t *testing.T) {
		adminEmail := fmt.Sprintf("sessadmin%d@edgecase.test", run)
		must(t, store.CreateUser(ctx, domain.User{ID: fmt.Sprintf("usr_sessadmin_%d", run), Email: adminEmail, PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: time.Now()}))
		admin := loginAs(t, srv, adminEmail, "testpass123")

		_, login := doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{"email": email, "password": "testpass123"})
		access, refresh := login["token"].(string), login["refreshToken"].(string)

		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/admin/users/"+u.ID+"/revoke-sessions", admin, map[string]string{}); code != http.StatusBadRequest {
			t.Fatalf("revoke without a reason = %d, want 400", code)
		}
		code, out := doJSON(t, http.MethodPost, srv.URL+"/admin/users/"+u.ID+"/revoke-sessions", admin, map[string]string{"reason": "lost phone"})
		if code != http.StatusOK {
			t.Fatalf("revoke = %d %v", code, out)
		}
		if code, _ := doJSON(t, http.MethodGet, srv.URL+"/me", access, nil); code != http.StatusUnauthorized {
			t.Fatalf("access token still works after revoke: %d", code)
		}
		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/refresh", "", map[string]string{"refreshToken": refresh}); code != http.StatusUnauthorized {
			t.Fatalf("refresh token still works after revoke: %d", code)
		}
		entries, err := store.AuditLogsByTarget(ctx, "user", u.ID, 10)
		must(t, err)
		if len(entries) == 0 || entries[0].Action != "user.revoke_sessions" {
			t.Fatalf("no user.revoke_sessions audit entry: %+v", entries)
		}
	})

	t.Run("a pre-session token can be upgraded once", func(t *testing.T) {
		current, err := store.UserByID(ctx, u.ID)
		must(t, err)
		legacy, err := tokens.Issue(current)
		must(t, err)
		code, out := doJSON(t, http.MethodPost, srv.URL+"/auth/session/upgrade", legacy, nil)
		if code != http.StatusOK || out["refreshToken"] == nil {
			t.Fatalf("upgrade = %d %v", code, out)
		}
		if code, _ := doJSON(t, http.MethodPost, srv.URL+"/auth/session/upgrade", out["token"].(string), nil); code != http.StatusConflict {
			t.Fatalf("upgrading a session token again = %d, want 409", code)
		}
	})
}
