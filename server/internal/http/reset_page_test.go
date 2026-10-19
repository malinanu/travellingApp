package httpapi_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

// The traveller reset page must set the password through the same checks
// as POST /auth/reset-password, and a token must only work once.
func TestResetPasswordPage(t *testing.T) {
	store := mustConnect(t)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000", PublicAPIURL: "https://api.example.test"}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	defer srv.Close()

	email := fmt.Sprintf("reset-page-%d@example.com", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{"name": "Reset Tester", "email": email, "password": "Original-Pass-123"})
	res, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register: status %d", res.StatusCode)
	}
	u, err := store.UserByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}
	raw := fmt.Sprintf("raw-token-%d", time.Now().UnixNano())
	sum := sha256.Sum256([]byte(raw))
	if err := store.CreatePasswordResetToken(context.Background(), fmt.Sprintf("prt_test_%d", time.Now().UnixNano()), u.ID, hex.EncodeToString(sum[:]), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	get, err := http.Get(srv.URL + "/reset-password?token=" + url.QueryEscape(raw))
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(get.Body)
	get.Body.Close()
	if get.StatusCode != http.StatusOK || !strings.Contains(string(page), `name="token" value="`+raw+`"`) {
		t.Fatalf("GET page: status %d, token field missing", get.StatusCode)
	}
	if csp := get.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "form-action 'self'") {
		t.Errorf("GET page CSP = %q", csp)
	}

	submit := func(pw, confirm string) (int, string) {
		t.Helper()
		res, err := http.PostForm(srv.URL+"/reset-password", url.Values{"token": {raw}, "password": {pw}, "confirm": {confirm}})
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}

	if st, b := submit("Brand-New-Pass-456", "Different-Pass-789"); st != http.StatusBadRequest || !strings.Contains(b, "don&#39;t match") {
		t.Errorf("mismatch: status %d", st)
	}
	if st, _ := submit("short", "short"); st != http.StatusBadRequest {
		t.Errorf("short password: status %d, want 400", st)
	}
	if st, b := submit("Brand-New-Pass-456", "Brand-New-Pass-456"); st != http.StatusOK || !strings.Contains(b, "Password updated") {
		t.Fatalf("valid reset: status %d", st)
	}
	loginAs(t, srv, email, "Brand-New-Pass-456")
	if st, b := submit("Another-Pass-000", "Another-Pass-000"); st != http.StatusBadRequest || !strings.Contains(b, "already used") {
		t.Errorf("reused token: status %d", st)
	}
}
