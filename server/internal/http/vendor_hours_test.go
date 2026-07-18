package httpapi_test

// Integration tests for per-day Business Hours and the public-holiday
// closure prompt (vendor.go's todayHoursAndOpenNow, handleUpcomingVendorHolidays,
// handleVendorHolidayDecision). Same real-Postgres pattern as
// validate_test.go; skips automatically if it can't connect.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestVendorHoursAndHolidays(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	testTOTPKey := make([]byte, 32)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: testTOTPKey, VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	vendor := domain.Vendor{ID: "ven_hours_test", Name: "Hours Test Vendor", Category: "cafe", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendor))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_hours_owner", Email: "owner@hourstest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_hours_manager", Email: "manager@hourstest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleManager, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_hours_staff", Email: "staff@hourstest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleStaff, CreatedAt: now}))

	ownerToken := loginAs(t, srv, "owner@hourstest.dev", "testpass123")
	managerToken := loginAs(t, srv, "manager@hourstest.dev", "testpass123")
	staffToken := loginAs(t, srv, "staff@hourstest.dev", "testpass123")

	weekdayKeys := [...]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	todayKey := weekdayKeys[now.Weekday()]
	openTime := now.Add(-1 * time.Hour).Format("15:04")
	closeTime := now.Add(1 * time.Hour).Format("15:04")

	t.Run("per-day hours reflect in the synced place", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/business", ownerToken, map[string]any{
			"hours": map[string]any{
				todayKey: map[string]any{"open": openTime, "close": closeTime},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/business: want 200, got %d", status)
		}

		var placeID string
		if err := store.Pool.QueryRow(ctx, `SELECT id FROM places WHERE vendor_id=$1`, vendor.ID).Scan(&placeID); err != nil {
			t.Fatalf("place should have been lazily created: %v", err)
		}
		var out struct {
			Hours   string `json:"hours"`
			OpenNow bool   `json:"openNow"`
		}
		if got := doReqJSON(t, srv, http.MethodGet, "/places/"+placeID, ownerToken, nil, &out); got != http.StatusOK {
			t.Fatalf("GET /places/%s: want 200, got %d", placeID, got)
		}
		if !out.OpenNow {
			t.Errorf("expected openNow=true for today's %s..%s window (now=%s), got hours=%q openNow=%v", openTime, closeTime, now.Format("15:04"), out.Hours, out.OpenNow)
		}
	})

	t.Run("legacy single open/close pair still works after the per-day hours change", func(t *testing.T) {
		// Simulates a vendor whose profile predates per-day hours — direct
		// SQL write bypasses the API's own (now per-day-only) validation,
		// matching what's actually sitting in the real database today.
		store.Pool.Exec(ctx, `UPDATE vendors SET profile = profile || jsonb_build_object('hours', jsonb_build_object('open', $2::text, 'close', $3::text)) WHERE id=$1`,
			vendor.ID, openTime, closeTime)
		// Any Business Profile write re-syncs the Place from the vendor row
		// as it now stands in the DB — use a harmless identity no-op.
		if got := doReq(t, srv, http.MethodPatch, "/vendor/business/identity", ownerToken, map[string]any{"name": vendor.Name, "town": "Weligama"}); got != http.StatusOK {
			t.Fatalf("identity no-op: want 200, got %d", got)
		}
		var placeID string
		store.Pool.QueryRow(ctx, `SELECT id FROM places WHERE vendor_id=$1`, vendor.ID).Scan(&placeID)
		var out struct {
			Hours   string `json:"hours"`
			OpenNow bool   `json:"openNow"`
		}
		doReqJSON(t, srv, http.MethodGet, "/places/"+placeID, ownerToken, nil, &out)
		if out.Hours == "" {
			t.Error("legacy {open,close} hours should still produce a non-empty display string, got empty")
		}
		if !out.OpenNow {
			t.Errorf("expected openNow=true for a 06:00-23:59 window covering now, got hours=%q openNow=%v", out.Hours, out.OpenNow)
		}
	})

	t.Run("holiday override closes today regardless of regular hours", func(t *testing.T) {
		todayDate := now.Format("2006-01-02")
		if got := doReq(t, srv, http.MethodPost, "/vendor/holidays/decision", ownerToken, map[string]any{
			"date": todayDate, "name": "Test Holiday", "closed": true,
		}); got != http.StatusOK {
			t.Fatalf("POST /vendor/holidays/decision: want 200, got %d", got)
		}

		var placeID string
		store.Pool.QueryRow(ctx, `SELECT id FROM places WHERE vendor_id=$1`, vendor.ID).Scan(&placeID)
		var out struct {
			Hours   string `json:"hours"`
			OpenNow bool   `json:"openNow"`
		}
		doReqJSON(t, srv, http.MethodGet, "/places/"+placeID, ownerToken, nil, &out)
		if out.OpenNow {
			t.Errorf("expected openNow=false on a closed holiday override, got hours=%q openNow=%v", out.Hours, out.OpenNow)
		}
	})

	t.Run("manager can answer a holiday decision without a profile grant", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPost, "/vendor/holidays/decision", managerToken, map[string]any{
			"date": "2027-12-25", "name": "Christmas Day", "closed": false,
		}); got != http.StatusOK {
			t.Errorf("manager POST /vendor/holidays/decision: want 200, got %d", got)
		}
		// Sanity check the RBAC lock from the earlier feature is untouched:
		// a manager still can't write the rest of Business Profile without a grant.
		if got := doReq(t, srv, http.MethodPut, "/vendor/business", managerToken, map[string]any{"blurb": "nope"}); got != http.StatusForbidden {
			t.Errorf("manager PUT /vendor/business should still be locked: want 403, got %d", got)
		}
	})

	t.Run("staff blocked from both holiday endpoints", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodGet, "/vendor/holidays/upcoming", staffToken, nil); got != http.StatusForbidden {
			t.Errorf("staff GET upcoming: want 403, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPost, "/vendor/holidays/decision", staffToken, map[string]any{
			"date": "2027-01-01", "name": "New Year's Day", "closed": true,
		}); got != http.StatusForbidden {
			t.Errorf("staff POST decision: want 403, got %d", got)
		}
	})

	// cleanup so repeat runs stay idempotent
	store.Pool.Exec(ctx, `DELETE FROM places WHERE vendor_id='ven_hours_test'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id IN ('usr_hours_owner','usr_hours_manager','usr_hours_staff')`)
	store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id='ven_hours_test'`)
}
