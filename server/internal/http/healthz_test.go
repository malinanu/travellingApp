package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

func TestHealthAndReadinessEndpoints(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)

	tokens := auth.NewTokens("health-test-secret")
	srv := httpapi.New(store, tokens, config.Config{})
	router := srv.Router(config.Config{})

	t.Run("GET /healthz", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for /healthz, got %d", rec.Code)
		}
	})

	t.Run("GET /readyz", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/readyz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK for /readyz, got %d", rec.Code)
		}
	})
}
