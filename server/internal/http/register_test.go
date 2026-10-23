package httpapi_test

// Integration tests for vendor sign-up (POST /auth/register and
// /auth/register/check) against the real Postgres store. Skips when no local
// postgres is available, like the other tests in this package.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/seed"
)

func postJSON(t *testing.T, url string, body any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(body)
	res, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestVendorRegistration(t *testing.T) {
	store := mustConnect(t)
	ctx := context.Background()
	// Registration validates the category against the in-memory taxonomy,
	// which the API loads at boot (cmd/api/main.go).
	if _, err := store.EnsureVendorCategories(ctx, seed.InitialCategories); err != nil {
		t.Fatal(err)
	}
	cats, err := store.GetVendorCategories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	domain.InitVendorCategories(cats)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32)}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	defer srv.Close()
	stamp := time.Now().UnixNano()

	vendorReq := func(email string) map[string]any {
		return map[string]any{
			"email": email, "password": "Register-Test-123", "name": "Test Owner", "role": "vendor",
			"businessName": "Register Test Cafe", "category": "coffee", "town": "Weligama",
			"phone": "+94 761234567", "lat": 5.97, "lng": 80.43,
		}
	}

	t.Run("phone is stored on the account", func(t *testing.T) {
		email := fmt.Sprintf("register-phone-%d@example.com", stamp)
		status, body := postJSON(t, srv.URL+"/auth/register", vendorReq(email))
		if status != http.StatusCreated {
			t.Fatalf("register: status %d %v", status, body)
		}
		u, err := store.UserByEmail(ctx, email)
		if err != nil {
			t.Fatal(err)
		}
		if u.Phone != "+94 761234567" {
			t.Fatalf("user phone = %q, want %q", u.Phone, "+94 761234567")
		}
		if u.VendorID == nil {
			t.Fatal("vendor account has no vendorId")
		}
		v, err := store.VendorByID(ctx, *u.VendorID)
		if err != nil {
			t.Fatal(err)
		}
		if v.Status != domain.VendorPending || v.Lat != 5.97 || v.Lng != 80.43 {
			t.Fatalf("vendor = status %s at %v,%v; want pending at 5.97,80.43", v.Status, v.Lat, v.Lng)
		}
	})

	t.Run("an already-listed Google place still registers, flagged", func(t *testing.T) {
		placeID := fmt.Sprintf("test-place-%d", stamp)
		existing := domain.Vendor{
			ID: fmt.Sprintf("ven_listed_%d", stamp), Name: "Already Listed", Category: "coffee",
			Status: domain.VendorApproved, Priority: "normal", Flags: []string{}, Tier: "Free",
			SubmittedAt: time.Now(), GooglePlaceID: &placeID,
		}
		if err := store.UpsertVendor(ctx, existing); err != nil {
			t.Fatal(err)
		}
		req := vendorReq(fmt.Sprintf("register-dup-%d@example.com", stamp))
		req["googlePlaceId"] = placeID
		status, body := postJSON(t, srv.URL+"/auth/register", req)
		if status != http.StatusCreated {
			t.Fatalf("register: status %d %v", status, body)
		}
		u, err := store.UserByEmail(ctx, req["email"].(string))
		if err != nil {
			t.Fatal(err)
		}
		v, err := store.VendorByID(ctx, *u.VendorID)
		if err != nil {
			t.Fatal(err)
		}
		if v.GooglePlaceID != nil {
			t.Fatalf("new vendor took the listed place ID %q", *v.GooglePlaceID)
		}
		if len(v.Flags) != 1 || !strings.Contains(v.Flags[0], existing.ID) {
			t.Fatalf("flags = %v, want one naming %s", v.Flags, existing.ID)
		}
	})

	t.Run("a malformed email is rejected", func(t *testing.T) {
		for _, path := range []string{"/auth/register/check", "/auth/register"} {
			status, body := postJSON(t, srv.URL+path, vendorReq("not-an-email"))
			if status != http.StatusBadRequest {
				t.Fatalf("%s: status %d %v, want 400", path, status, body)
			}
		}
	})

	t.Run("a taken email is a conflict", func(t *testing.T) {
		email := fmt.Sprintf("register-phone-%d@example.com", stamp)
		for _, path := range []string{"/auth/register/check", "/auth/register"} {
			status, body := postJSON(t, srv.URL+path, vendorReq(email))
			if status != http.StatusConflict {
				t.Fatalf("%s: status %d %v, want 409", path, status, body)
			}
		}
	})
}
