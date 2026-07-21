package httpapi_test

// Integration tests for the advanced admin vendor-detail surface: the
// read-only detail endpoints (offers / reviews / team / audit), the admin
// override-edit (PATCH), and the emergency takedown (which must pull the
// vendor's place from the traveller feed). Same real-Postgres pattern as the
// other integration tests; skips without a local database.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/store/postgres"
)

func TestAdminVendorDetailAndControls(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	purgeVendorDetailTestData(t, store, ctx)
	t.Cleanup(func() { purgeVendorDetailTestData(t, store, ctx) })

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")

	vendor := domain.Vendor{
		ID: "ven_det_test", Name: "Detail Test Cafe", Category: "cafe", Town: "Ella",
		Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now,
		Lat: 6.87, Lng: 81.05,
	}
	must(t, store.UpsertVendor(ctx, vendor))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_det_admin", Email: "admin@dettest.dev", PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_det_traveller", Email: "traveller@dettest.dev", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_det_owner", Email: "owner@dettest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))

	// A curated (vendorless) place must stay visible through a takedown.
	must(t, store.UpsertPlace(ctx, domain.Place{ID: "plc_det_curated", Name: "Curated Viewpoint", Data: json.RawMessage(`{}`)}))

	// Seed offers + reviews for the detail tabs.
	must(t, store.UpsertOffer(ctx, domain.Offer{ID: "off_det_1", VendorID: vendor.ID, Title: "Free coffee", Type: "Freebie", Value: "1", Status: domain.OfferLive}))
	must(t, store.UpsertReview(ctx, domain.Review{ID: "rev_det_good", PlaceID: "plc_det_r1", VendorID: &vendor.ID, TravellerID: "usr_det_traveller", Rating: 5, Text: "great", CreatedAt: now}))
	must(t, store.UpsertReview(ctx, domain.Review{ID: "rev_det_bad", PlaceID: "plc_det_r2", VendorID: &vendor.ID, TravellerID: "usr_det_traveller", Rating: 1, Text: "bad", CreatedAt: now}))

	adminToken := loginAs(t, srv, "admin@dettest.dev", "testpass123")
	travellerToken := loginAs(t, srv, "traveller@dettest.dev", "testpass123")

	t.Run("detail endpoints return seeded data", func(t *testing.T) {
		var offers []domain.Offer
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+vendor.ID+"/offers", adminToken, nil, &offers); got != http.StatusOK || len(offers) != 1 {
			t.Fatalf("offers: got status %d, %d offers", got, len(offers))
		}
		var allReviews, badReviews []domain.Review
		doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+vendor.ID+"/reviews?all=1", adminToken, nil, &allReviews)
		doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+vendor.ID+"/reviews", adminToken, nil, &badReviews)
		if len(allReviews) != 2 {
			t.Errorf("reviews?all=1: want 2, got %d", len(allReviews))
		}
		if len(badReviews) != 1 {
			t.Errorf("reviews (bad-only default): want 1, got %d", len(badReviews))
		}
		var team struct {
			Members []domain.User `json:"members"`
		}
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+vendor.ID+"/team", adminToken, nil, &team); got != http.StatusOK {
			t.Fatalf("team: got %d", got)
		}
		if len(team.Members) < 1 {
			t.Errorf("team members: want >=1 (owner), got %d", len(team.Members))
		}
	})

	t.Run("admin edit persists and syncs to the traveller place", func(t *testing.T) {
		body := map[string]any{"name": "Renamed By Admin", "note": "corrected", "profile": map[string]any{"blurb": "admin blurb"}}
		if got := doReq(t, srv, http.MethodPatch, "/admin/vendors/"+vendor.ID, adminToken, body); got != http.StatusOK {
			t.Fatalf("PATCH: want 200, got %d", got)
		}
		// Persisted on the vendor row.
		var all []domain.Vendor
		doReqJSON(t, srv, http.MethodGet, "/admin/vendors", adminToken, nil, &all)
		if !hasVendorNamed(all, vendor.ID, "Renamed By Admin") {
			t.Error("edited name did not persist on the vendor")
		}
		// Synced into the traveller feed.
		if name := placeNameInFeed(t, srv, travellerToken, vendor.ID); name != "Renamed By Admin" {
			t.Errorf("edited name not synced to /places: got %q", name)
		}
	})

	t.Run("takedown suspends, hides the place, and logs the reason", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPost, "/admin/vendors/"+vendor.ID+"/takedown", adminToken, map[string]any{"reason": ""}); got != http.StatusBadRequest {
			t.Errorf("empty reason: want 400, got %d", got)
		}
		if got := doReq(t, srv, http.MethodPost, "/admin/vendors/"+vendor.ID+"/takedown", adminToken, map[string]any{"reason": "fraud report"}); got != http.StatusOK {
			t.Fatalf("takedown: want 200, got %d", got)
		}
		// Place gone from the traveller feed, curated place still present.
		if name := placeNameInFeed(t, srv, travellerToken, vendor.ID); name != "" {
			t.Errorf("suspended vendor's place still in /places: %q", name)
		}
		if !curatedInFeed(t, srv, travellerToken, "plc_det_curated") {
			t.Error("curated vendorless place was wrongly hidden")
		}
		// Audit trail records the takedown.
		var audit []domain.AdminAuditLog
		doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+vendor.ID+"/audit", adminToken, nil, &audit)
		if !hasAction(audit, "vendor.takedown") {
			t.Error("takedown not found in vendor audit history")
		}
	})
}

func hasVendorNamed(vs []domain.Vendor, id, name string) bool {
	for _, v := range vs {
		if v.ID == id && v.Name == name {
			return true
		}
	}
	return false
}

func hasAction(entries []domain.AdminAuditLog, action string) bool {
	for _, e := range entries {
		if e.Action == action {
			return true
		}
	}
	return false
}

// placeNameInFeed returns the traveller-facing name of the vendor's place, or
// "" if it isn't in GET /places.
func placeNameInFeed(t *testing.T, srv *httptest.Server, token, vendorID string) string {
	t.Helper()
	var places []map[string]any
	doReqJSON(t, srv, http.MethodGet, "/places", token, nil, &places)
	for _, p := range places {
		if p["vendorId"] == vendorID {
			name, _ := p["name"].(string)
			return name
		}
	}
	return ""
}

func curatedInFeed(t *testing.T, srv *httptest.Server, token, placeID string) bool {
	t.Helper()
	var places []map[string]any
	doReqJSON(t, srv, http.MethodGet, "/places", token, nil, &places)
	for _, p := range places {
		if p["id"] == placeID {
			return true
		}
	}
	return false
}

func purgeVendorDetailTestData(t *testing.T, store *postgres.Store, ctx context.Context) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM admin_audit_log WHERE target_id = 'ven_det_test'`,
		`DELETE FROM reviews WHERE vendor_id = 'ven_det_test'`,
		`DELETE FROM offers WHERE vendor_id = 'ven_det_test'`,
		`DELETE FROM places WHERE vendor_id = 'ven_det_test' OR id = 'plc_det_curated'`,
		`DELETE FROM users WHERE id LIKE 'usr_det_%'`,
		`DELETE FROM vendors WHERE id = 'ven_det_test'`,
	} {
		if _, err := store.Pool.Exec(ctx, q); err != nil {
			t.Fatalf("purge (%s): %v", q, err)
		}
	}
}
