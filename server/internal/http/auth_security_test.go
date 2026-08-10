package httpapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

func TestAuthSecurityEnforcement(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)

	tokens := auth.NewTokens("security-test-secret")
	srv := httpapi.New(store, tokens, config.Config{})
	router := srv.Router(config.Config{})

	t.Run("Reject Invalid Login Payload", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/auth/login", bytes.NewBufferString(`{"email":"","password":""}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Errorf("expected login rejection for empty credentials, got 200")
		}
	})

	t.Run("Reject Unauthorized Protected Endpoint Call", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/vendor/profile", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized for unauthenticated route, got %d", rec.Code)
		}
	})

	t.Run("Enforce HTTP Security Headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("expected X-Content-Type-Options: nosniff")
		}
		if rec.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("expected X-Frame-Options: DENY")
		}
		if rec.Header().Get("Referrer-Policy") == "" {
			t.Errorf("expected Referrer-Policy header")
		}
	})
}
