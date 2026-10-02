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

// /auth/apple must only trust a verified Apple identity token. These cases
// all fail before the handler touches the database, but New() needs a store, so it skips without a local Postgres like the other handler tests.
func TestAppleAuthRequiresVerifiedToken(t *testing.T) {
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(mustConnect(t), auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	defer srv.Close()

	cases := []struct {
		name, body string
		want       int
	}{
		// The original exploit: claim an existing user's email, no proof.
		{"client-claimed id and email, no token", `{"appleId":"x","email":"victim@example.com"}`, http.StatusBadRequest},
		{"empty token", `{"identityToken":""}`, http.StatusBadRequest},
		{"garbage token", `{"identityToken":"not.a.jwt","appleId":"x","email":"victim@example.com"}`, http.StatusUnauthorized},
	}
	for _, c := range cases {
		res, err := http.Post(srv.URL+"/auth/apple", "application/json", strings.NewReader(c.body))
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != c.want {
			t.Errorf("%s: status %d, want %d", c.name, res.StatusCode, c.want)
		}
	}
}
