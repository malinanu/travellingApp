package httpapi_test

import (
	"context"
	"encoding/json"
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

func doJSONArray(t *testing.T, method, url, token string) (int, []map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out []map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestPrivacyExportSessions(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	run := time.Now().UnixNano()
	email := fmt.Sprintf("priv%d@edgecase.test", run)
	pw, _ := auth.HashPassword("testpass123")
	u := domain.User{
		ID:           fmt.Sprintf("usr_priv_%d", run),
		Email:        email,
		PasswordHash: pw,
		Role:         domain.RoleTraveller,
		Name:         "Privacy Tester",
		CreatedAt:    time.Now(),
	}
	must(t, store.CreateUser(ctx, u))

	// Login to get access token & session
	code, login := doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{"email": email, "password": "testpass123"})
	if code != http.StatusOK {
		t.Fatalf("login failed: %d %v", code, login)
	}
	token := login["token"].(string)

	t.Run("PUT /me/privacy updates all 5 privacy settings", func(t *testing.T) {
		tglTrue := true
		tglFalse := false
		req := map[string]any{
			"shareLocation":      tglTrue,
			"usageAnalytics":     tglFalse,
			"visibleToGroup":     tglFalse,
			"countVibeVotes":     tglFalse,
			"personalisedOffers": tglTrue,
		}
		code, out := doJSON(t, http.MethodPut, srv.URL+"/me/privacy", token, req)
		if code != http.StatusOK {
			t.Fatalf("PUT /me/privacy: %d %v", code, out)
		}
		if out["shareLocation"] != true || out["usageAnalytics"] != false ||
			out["visibleToGroup"] != false || out["countVibeVotes"] != false ||
			out["personalisedOffers"] != true {
			t.Fatalf("privacy response unexpected: %v", out)
		}

		// Verify UserByID matches
		user, err := store.UserByID(ctx, u.ID)
		must(t, err)
		if !user.ShareLocation || user.UsageAnalytics || user.VisibleToGroup || user.CountVibeVotes || !user.PersonalisedOffers {
			t.Fatalf("store values mismatch: %+v", user)
		}

		// Verify GET /me includes them
		code, meResp := doJSON(t, http.MethodGet, srv.URL+"/me", token, nil)
		if code != http.StatusOK {
			t.Fatalf("GET /me: %d %v", code, meResp)
		}
		if meResp["visibleToGroup"] != false || meResp["countVibeVotes"] != false || meResp["personalisedOffers"] != true {
			t.Fatalf("GET /me missing new privacy fields: %v", meResp)
		}
	})

	t.Run("GET /me/export returns comprehensive user data", func(t *testing.T) {
		code, export := doJSON(t, http.MethodGet, srv.URL+"/me/export", token, nil)
		if code != http.StatusOK {
			t.Fatalf("GET /me/export: %d %v", code, export)
		}
		for _, key := range []string{
			"user", "saved_places", "inspiration_links", "trips", "reviews",
			"checkins", "vibe_checkins", "quest_completions", "claims",
			"guide_conversations", "user_sessions", "exported_at",
		} {
			if _, exists := export[key]; !exists {
				t.Fatalf("GET /me/export missing required key: %s", key)
			}
		}
	})

	t.Run("GET /me/sessions and DELETE /me/sessions/{id}", func(t *testing.T) {
		// Create a second session for this user
		_, login2 := doJSON(t, http.MethodPost, srv.URL+"/auth/login", "", map[string]string{"email": email, "password": "testpass123"})
		token2 := login2["token"].(string)

		// List sessions using token2
		code, sessions := doJSONArray(t, http.MethodGet, srv.URL+"/me/sessions", token2)
		if code != http.StatusOK || len(sessions) < 2 {
			t.Fatalf("expected at least 2 sessions: %d %v", code, sessions)
		}

		// One of them must be current
		var otherID string
		for _, s := range sessions {
			if s["isCurrent"] == true {
				// current session
			} else {
				otherID = s["id"].(string)
			}
		}
		if otherID == "" {
			t.Fatalf("expected other session id, got none: %v", sessions)
		}

		// Delete the other session
		code, delResp := doJSON(t, http.MethodDelete, srv.URL+"/me/sessions/"+otherID, token2, nil)
		if code != http.StatusOK {
			t.Fatalf("DELETE /me/sessions/%s: %d %v", otherID, code, delResp)
		}

		// Re-fetch sessions: other session should no longer be in active sessions
		code, sessions = doJSONArray(t, http.MethodGet, srv.URL+"/me/sessions", token2)
		if code != http.StatusOK {
			t.Fatalf("GET /me/sessions after delete: %d", code)
		}
		for _, s := range sessions {
			if s["id"] == otherID {
				t.Fatalf("deleted session %s still returned in active sessions: %v", otherID, sessions)
			}
		}
	})
}
