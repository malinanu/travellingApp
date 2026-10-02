package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

// /auth/google/mobile must only trust a verified Google ID token. These
// cases fail before the handler writes anything, but New() needs a store, so
// it skips without a local Postgres like the other handler tests.
func TestGoogleMobileAuthRequiresVerifiedToken(t *testing.T) {
	post := func(t *testing.T, clientID, body string) int {
		t.Helper()
		cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000", GoogleClientID: clientID}
		srv := httptest.NewServer(httpapi.New(mustConnect(t), auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
		defer srv.Close()
		res, err := http.Post(srv.URL+"/auth/google/mobile", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}

	if got := post(t, "", `{"idToken":"x"}`); got != http.StatusNotImplemented {
		t.Errorf("unconfigured: status %d, want 501", got)
	}
	if got := post(t, "web-client.apps.googleusercontent.com", `{"email":"victim@example.com"}`); got != http.StatusBadRequest {
		t.Errorf("no token: status %d, want 400", got)
	}
	if got := post(t, "web-client.apps.googleusercontent.com", `{"idToken":"not.a.jwt","email":"victim@example.com"}`); got != http.StatusUnauthorized {
		t.Errorf("garbage token: status %d, want 401", got)
	}
}
