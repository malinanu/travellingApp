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
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

func TestVerifyResetTokenEndpoint(t *testing.T) {
	store := mustConnect(t)
	cfg := config.Config{
		CORSOrigins:       []string{"http://localhost:3000"},
		TOTPEncryptionKey: make([]byte, 32),
		VendorAppURL:      "http://localhost:3000",
		PublicAPIURL:      "https://api.example.test",
	}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	defer srv.Close()

	// 1. Missing token param -> 400
	res, err := http.Get(srv.URL + "/auth/verify-reset-token")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("missing token: got status %d, want 400", res.StatusCode)
	}

	// 2. Non-existent token -> 400
	res, err = http.Get(srv.URL + "/auth/verify-reset-token?token=nonexistenttoken123")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("nonexistent token: got status %d, want 400", res.StatusCode)
	}

	// Register a test user
	email := fmt.Sprintf("verify-token-%d@example.com", time.Now().UnixNano())
	body, _ := json.Marshal(map[string]string{"name": "Token Tester", "email": email, "password": "Original-Pass-123"})
	regRes, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	regRes.Body.Close()
	if regRes.StatusCode != http.StatusCreated {
		t.Fatalf("register: status %d", regRes.StatusCode)
	}

	u, err := store.UserByEmail(context.Background(), email)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Valid token -> 200 {"valid": true}
	rawToken1 := fmt.Sprintf("raw-token-1-%d", time.Now().UnixNano())
	sum1 := sha256.Sum256([]byte(rawToken1))
	tokenHash1 := hex.EncodeToString(sum1[:])
	if err := store.CreatePasswordResetToken(context.Background(), fmt.Sprintf("prt_test_1_%d", time.Now().UnixNano()), u.ID, tokenHash1, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	res, err = http.Get(srv.URL + "/auth/verify-reset-token?token=" + url.QueryEscape(rawToken1))
	if err != nil {
		t.Fatal(err)
	}
	respBytes, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("valid token: status %d, body: %s", res.StatusCode, string(respBytes))
	}

	// 4. Create a second token for the same user -> should purge the first token
	rawToken2 := fmt.Sprintf("raw-token-2-%d", time.Now().UnixNano())
	sum2 := sha256.Sum256([]byte(rawToken2))
	tokenHash2 := hex.EncodeToString(sum2[:])
	if err := store.CreatePasswordResetToken(context.Background(), fmt.Sprintf("prt_test_2_%d", time.Now().UnixNano()), u.ID, tokenHash2, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	// First token must now be invalid/purged
	res, err = http.Get(srv.URL + "/auth/verify-reset-token?token=" + url.QueryEscape(rawToken1))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("superseded token1: got status %d, want 400", res.StatusCode)
	}

	// Second token must be valid
	res, err = http.Get(srv.URL + "/auth/verify-reset-token?token=" + url.QueryEscape(rawToken2))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("latest token2: got status %d, want 200", res.StatusCode)
	}

	// 5. Expired token -> should be purged on verification and return 400
	rawExpired := fmt.Sprintf("raw-token-expired-%d", time.Now().UnixNano())
	sumExp := sha256.Sum256([]byte(rawExpired))
	tokenHashExp := hex.EncodeToString(sumExp[:])
	// Directly insert expired token into DB
	if err := store.CreatePasswordResetToken(context.Background(), fmt.Sprintf("prt_test_exp_%d", time.Now().UnixNano()), u.ID, tokenHashExp, time.Now().Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	res, err = http.Get(srv.URL + "/auth/verify-reset-token?token=" + url.QueryEscape(rawExpired))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expired token: got status %d, want 400", res.StatusCode)
	}
}

