package httpapi_test

// Integration tests for the admin vendor directory and its per-vendor stats
// pane. Two things are covered because two distinct bugs were possible:
//
//  1. GET /admin/vendors with no status must return *every* vendor â€”
//     including suspended ones and ones with no map coordinates, which the
//     approval queue (pending only) and geo map (coords only) each hide.
//  2. GET /admin/vendors/{id}/stats must aggregate the right rows â€” notably
//     checkins, which carry no vendor_id and join via places.vendor_id.
//
// Same real-Postgres pattern as validate_test.go; skips if it can't connect.

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

func TestAdminVendorDirectoryAndStats(t *testing.T) {
	store := mustConnect(t)
	// Registered before the purge cleanup below so it runs *after* it â€”
	// t.Cleanup is LIFO, and a plain `defer store.Close()` would close the
	// pool out from under the purge.
	t.Cleanup(store.Close)
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	testTOTPKey := make([]byte, 32)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: testTOTPKey, VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	// The seed helpers below are plain INSERTs (CreateUser/CreateClaim/
	// AddCheckin), so leftovers from a previous run would
	// either collide on a primary key or silently inflate the counts.
	// Purge before seeding and again on the way out.
	purgeVendorStatsTestData(t, store, ctx)
	t.Cleanup(func() { purgeVendorStatsTestData(t, store, ctx) })

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")

	// The subject vendor: approved, with coordinates and a full spread of
	// activity to aggregate.
	subject := domain.Vendor{
		ID: "ven_stats_subject", Name: "Stats Subject Cafe", Category: "cafe", Town: "Ella",
		Status: domain.VendorApproved, Tier: "Premium", Flags: []string{}, SubmittedAt: now,
		Lat: 6.8667, Lng: 81.0466,
	}
	must(t, store.UpsertVendor(ctx, subject))

	// A suspended vendor and a coordinate-less one â€” the two cases that were
	// invisible in every pre-existing admin surface.
	suspended := domain.Vendor{
		ID: "ven_stats_suspended", Name: "Stats Suspended Inn", Category: "hotel", Town: "Kandy",
		Status: domain.VendorSuspended, Tier: "Free", Flags: []string{}, SubmittedAt: now,
		Lat: 7.2906, Lng: 80.6337,
	}
	must(t, store.UpsertVendor(ctx, suspended))
	noCoords := domain.Vendor{
		ID: "ven_stats_nocoords", Name: "Stats No Coords Stall", Category: "cafe", Town: "Galle",
		Status: domain.VendorPending, Tier: "Free", Flags: []string{}, SubmittedAt: now,
		// Lat/Lng deliberately zero.
	}
	must(t, store.UpsertVendor(ctx, noCoords))

	must(t, store.CreateUser(ctx, domain.User{ID: "usr_stats_admin", Email: "admin@statstest.dev", PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: now}))
	adminToken := loginAs(t, srv, "admin@statstest.dev", "testpass123")

	t.Run("directory returns every vendor regardless of status or coords", func(t *testing.T) {
		var all []domain.Vendor
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors", adminToken, nil, &all); got != http.StatusOK {
			t.Fatalf("want 200, got %d", got)
		}
		for _, want := range []string{subject.ID, suspended.ID, noCoords.ID} {
			if !containsVendor(all, want) {
				t.Errorf("directory is missing vendor %s", want)
			}
		}
	})

	t.Run("status filter still narrows the queue", func(t *testing.T) {
		var pending []domain.Vendor
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors?status=pending", adminToken, nil, &pending); got != http.StatusOK {
			t.Fatalf("want 200, got %d", got)
		}
		if containsVendor(pending, suspended.ID) {
			t.Error("status=pending leaked a suspended vendor")
		}
		if !containsVendor(pending, noCoords.ID) {
			t.Error("status=pending dropped a pending vendor")
		}
	})

	// Seed activity for the subject vendor only, so the counts below are
	// unambiguous even if other tests leave rows behind.
	seedVendorActivity(t, store, ctx, subject, now)

	t.Run("stats aggregate the subject vendor's activity", func(t *testing.T) {
		var st struct {
			Claimed      int     `json:"claimed"`
			Redeemed     int     `json:"redeemed"`
			Redeemed30d  int     `json:"redeemed30d"`
			Checkins     int     `json:"checkins"`
			Checkins30d  int     `json:"checkins30d"`
			Reviews      int     `json:"reviews"`
			AvgRating    float64 `json:"avgRating"`
			BadReviews   int     `json:"badReviews"`
			Flagged      bool    `json:"flagged"`
			Offers       int     `json:"offers"`
			OffersLive   int     `json:"offersLive"`
			OffersDraft  int     `json:"offersDraft"`
			TeamTotal    int     `json:"teamTotal"`
			TeamManagers int     `json:"teamManagers"`
			TeamStaff    int     `json:"teamStaff"`
			SubTier      *string `json:"subTier"`
			SubMRR       *int    `json:"subMrr"`
		}
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+subject.ID+"/stats", adminToken, nil, &st); got != http.StatusOK {
			t.Fatalf("want 200, got %d", got)
		}

		// 3 claims seeded, 2 of them redeemed (both within 30 days).
		if st.Claimed != 3 {
			t.Errorf("claimed: want 3, got %d", st.Claimed)
		}
		if st.Redeemed != 2 {
			t.Errorf("redeemed: want 2, got %d", st.Redeemed)
		}
		if st.Redeemed30d != 2 {
			t.Errorf("redeemed30d: want 2, got %d", st.Redeemed30d)
		}
		// 2 check-ins, reached only via places.vendor_id.
		if st.Checkins != 2 {
			t.Errorf("checkins: want 2 (joined via places), got %d", st.Checkins)
		}
		if st.Checkins30d != 2 {
			t.Errorf("checkins30d: want 2, got %d", st.Checkins30d)
		}
		// Ratings 5, 4, 1 â†’ avg 3.33, one bad review, below the flag threshold of 5.
		if st.Reviews != 3 {
			t.Errorf("reviews: want 3, got %d", st.Reviews)
		}
		if st.AvgRating < 3.32 || st.AvgRating > 3.34 {
			t.Errorf("avgRating: want ~3.33, got %v", st.AvgRating)
		}
		if st.BadReviews != 1 {
			t.Errorf("badReviews: want 1, got %d", st.BadReviews)
		}
		if st.Flagged {
			t.Error("flagged: want false (1 bad review is under the threshold)")
		}
		if st.Offers != 3 || st.OffersLive != 2 || st.OffersDraft != 1 {
			t.Errorf("offers: want 3 total/2 live/1 draft, got %d/%d/%d", st.Offers, st.OffersLive, st.OffersDraft)
		}
		if st.TeamTotal != 3 || st.TeamManagers != 1 || st.TeamStaff != 1 {
			t.Errorf("team: want 3 total/1 manager/1 staff, got %d/%d/%d", st.TeamTotal, st.TeamManagers, st.TeamStaff)
		}
		if st.SubTier == nil || *st.SubTier != "Premium" {
			t.Errorf("subTier: want Premium, got %v", st.SubTier)
		}
		if st.SubMRR == nil || *st.SubMRR != 49 {
			t.Errorf("subMrr: want 49, got %v", st.SubMRR)
		}
	})

	t.Run("a vendor with no activity reports zeroes, not an error", func(t *testing.T) {
		var st map[string]any
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors/"+noCoords.ID+"/stats", adminToken, nil, &st); got != http.StatusOK {
			t.Fatalf("want 200, got %d", got)
		}
		if st["redeemed"] != float64(0) || st["reviews"] != float64(0) {
			t.Errorf("want zeroed stats, got %v", st)
		}
		// No subscriptions row â€” the field should be omitted, not zero.
		if _, present := st["subTier"]; present {
			t.Error("subTier should be omitted for a vendor with no subscription")
		}
	})

	t.Run("stats for an unknown vendor 404", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodGet, "/admin/vendors/ven_does_not_exist/stats", adminToken, nil); got != http.StatusNotFound {
			t.Errorf("want 404, got %d", got)
		}
	})

	t.Run("summary aggregates network-wide vendor activity", func(t *testing.T) {
		// The shared local DB holds vendors beyond this test's seeds, so every
		// assertion is a lower bound: the seeded rows must be *included*.
		var sm struct {
			Total      int     `json:"total"`
			Pending    int     `json:"pending"`
			Approved   int     `json:"approved"`
			Suspended  int     `json:"suspended"`
			Flagged    int     `json:"flagged"`
			Redeemed   int     `json:"redeemed"`
			Checkins   int     `json:"checkins"`
			Reviews    int     `json:"reviews"`
			AvgRating  float64 `json:"avgRating"`
			BadReviews int     `json:"badReviews"`
			Offers     int     `json:"offers"`
			OffersLive int     `json:"offersLive"`
			ActiveSubs int     `json:"activeSubs"`
			MRR        int     `json:"mrr"`
		}
		if got := doReqJSON(t, srv, http.MethodGet, "/admin/vendors/summary", adminToken, nil, &sm); got != http.StatusOK {
			t.Fatalf("want 200, got %d", got)
		}
		// 3 seeded vendors (approved/suspended/pending) with 2 redemptions,
		// 2 check-ins, 3 reviews (1 bad), 3 offers (2 live), 1 active $49 sub.
		checks := []struct {
			name       string
			got, floor int
		}{
			{"total", sm.Total, 3},
			{"pending", sm.Pending, 1},
			{"approved", sm.Approved, 1},
			{"suspended", sm.Suspended, 1},
			{"redeemed", sm.Redeemed, 2},
			{"checkins", sm.Checkins, 2},
			{"reviews", sm.Reviews, 3},
			{"badReviews", sm.BadReviews, 1},
			{"offers", sm.Offers, 3},
			{"offersLive", sm.OffersLive, 2},
			{"activeSubs", sm.ActiveSubs, 1},
			{"mrr", sm.MRR, 49},
		}
		for _, c := range checks {
			if c.got < c.floor {
				t.Errorf("%s: want >= %d, got %d", c.name, c.floor, c.got)
			}
		}
		if sm.AvgRating < 1 || sm.AvgRating > 5 {
			t.Errorf("avgRating out of range: %v", sm.AvgRating)
		}
	})
}

// purgeVendorStatsTestData removes every row this test seeds, keyed by the
// `ven_stats_`/`usr_stats_`/`plc_stats_` prefixes it owns, so the test is
// re-runnable against a persistent local database.
func purgeVendorStatsTestData(t *testing.T, store *postgres.Store, ctx context.Context) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM checkins WHERE place_id LIKE 'plc_stats_%'`,
		`DELETE FROM claims WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM reviews WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM offers WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM subscriptions WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM vendor_invites WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM vendor_blobs WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM vendor_notifications WHERE vendor_id LIKE 'ven_stats_%'`,
		`DELETE FROM feed_items WHERE vendor IN ('Stats Subject Cafe', 'Stats Suspended Inn', 'Stats No Coords Stall')`,
		`DELETE FROM places WHERE id LIKE 'plc_stats_%'`,
		`DELETE FROM users WHERE id LIKE 'usr_stats_%'`,
		`DELETE FROM vendors WHERE id LIKE 'ven_stats_%'`,
	} {
		if _, err := store.Pool.Exec(ctx, q); err != nil {
			t.Fatalf("purge (%s): %v", q, err)
		}
	}
}

func containsVendor(vs []domain.Vendor, id string) bool {
	for _, v := range vs {
		if v.ID == id {
			return true
		}
	}
	return false
}

// seedVendorActivity writes one of every activity row type the stats query
// aggregates, with deliberately distinct counts per type so a mis-wired
// subquery shows up as a specific wrong number rather than a plausible one.
func seedVendorActivity(t *testing.T, store *postgres.Store, ctx context.Context, v domain.Vendor, now time.Time) {
	t.Helper()

	placeID := "plc_stats_subject"
	must(t, store.UpsertPlace(ctx, domain.Place{ID: placeID, Name: v.Name, VendorID: &v.ID, Data: json.RawMessage(`{}`)}))

	// 3 claims, 2 redeemed.
	redeemedAt := now.Add(-2 * time.Hour)
	for i, c := range []struct {
		id, code string
		redeemed *time.Time
	}{
		{"clm_stats_1", "SPOTTO-ST01", &redeemedAt},
		{"clm_stats_2", "SPOTTO-ST02", &redeemedAt},
		{"clm_stats_3", "SPOTTO-ST03", nil},
	} {
		status := domain.ClaimRedeemed
		if c.redeemed == nil {
			status = domain.ClaimClaimed
		}
		must(t, store.CreateClaim(ctx, domain.Claim{
			ID: c.id, Code: c.code, PlaceID: placeID, VendorID: &v.ID,
			TravellerID: "usr_stats_traveller", Status: status,
			ClaimedAt: now.Add(-time.Duration(i+3) * time.Hour), RedeemedAt: c.redeemed,
		}))
	}

	// 2 check-ins â€” these hang off the place, not the vendor.
	must(t, store.AddCheckin(ctx, placeID, "usr_stats_traveller"))
	must(t, store.AddCheckin(ctx, placeID, "usr_stats_traveller2"))

	// 3 reviews: 5, 4, 1 stars.
	for _, rv := range []struct {
		id, traveller string
		rating        int
	}{
		{"rev_stats_1", "usr_stats_traveller", 5},
		{"rev_stats_2", "usr_stats_traveller2", 4},
		{"rev_stats_3", "usr_stats_traveller3", 1},
	} {
		must(t, store.UpsertReview(ctx, domain.Review{
			ID: rv.id, PlaceID: placeID + "_" + rv.id, VendorID: &v.ID,
			TravellerID: rv.traveller, Rating: rv.rating, Text: "seeded", CreatedAt: now,
		}))
	}

	// 3 offers: 2 live, 1 draft.
	for _, o := range []struct {
		id     string
		status domain.OfferStatus
	}{
		{"off_stats_1", domain.OfferLive},
		{"off_stats_2", domain.OfferLive},
		{"off_stats_3", domain.OfferDraft},
	} {
		must(t, store.UpsertOffer(ctx, domain.Offer{ID: o.id, VendorID: v.ID, Title: "Seeded offer", Status: o.status}))
	}

	must(t, store.UpsertSubscription(ctx, domain.Subscription{
		ID: "sub_stats_subject", VendorID: &v.ID, Vendor: v.Name, Town: v.Town,
		Tier: "Premium", MRR: 49, State: "active", Since: "2026-01-01",
	}))

	// Team: 1 owner + 1 manager + 1 staff.
	pw, _ := auth.HashPassword("testpass123")
	for _, u := range []struct {
		id, email string
		role      string
	}{
		{"usr_stats_owner", "owner@statstest.dev", domain.VendorRoleOwner},
		{"usr_stats_manager", "manager@statstest.dev", domain.VendorRoleManager},
		{"usr_stats_staff", "staff@statstest.dev", domain.VendorRoleStaff},
	} {
		must(t, store.CreateUser(ctx, domain.User{
			ID: u.id, Email: u.email, PasswordHash: pw, Role: domain.RoleVendor,
			VendorID: &v.ID, VendorRole: u.role, CreatedAt: now,
		}))
	}
}
