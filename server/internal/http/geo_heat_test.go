package httpapi_test

// The admin geo page's heat maps: travellers' rough areas are recorded only
// for opted-in travellers and never shown below 3 per area, every layer
// compares the period with the one before, and the filters narrow it. Same
// real-Postgres pattern as validate_test.go (skips without a database);
// the test data sits far out at sea so nothing else in the database is near.

import (
	"context"
	"fmt"
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

func geoTestServer(t *testing.T) (*postgres.Store, *httptest.Server) {
	store := mustConnect(t)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32)}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	t.Cleanup(srv.Close)
	return store, srv
}

func TestTravellerPresenceRecording(t *testing.T) {
	store, srv := geoTestServer(t)
	ctx := context.Background()
	purge := func() {
		store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_pres_%'`)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id = 'ven_pres'`)
	}
	purge()
	t.Cleanup(purge)
	pw, _ := auth.HashPassword("testpass123")
	execSQL(t, store, `INSERT INTO vendors (id, name, category, status) VALUES ('ven_pres', 'Pres', 'coffee', 'approved')`)
	ven := "ven_pres"
	for _, u := range []domain.User{
		{ID: "usr_pres_t", Email: "t@prestest.dev", Role: domain.RoleTraveller},
		{ID: "usr_pres_v", Email: "v@prestest.dev", Role: domain.RoleVendor, VendorID: &ven},
	} {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		must(t, store.CreateUser(ctx, u))
	}
	trav := loginAs(t, srv, "t@prestest.dev", "testpass123")
	vend := loginAs(t, srv, "v@prestest.dev", "testpass123")
	rows := func(id string) (n int, lat, lng int) {
		store.Pool.QueryRow(ctx, `SELECT count(*), COALESCE(min(cell_lat),0), COALESCE(min(cell_lng),0) FROM traveller_presence WHERE user_id=$1`, id).Scan(&n, &lat, &lng)
		return
	}
	get := func(tok, q string) {
		t.Helper()
		if code := doReqJSON(t, srv, http.MethodGet, "/places"+q, tok, nil, nil); code != http.StatusOK {
			t.Fatalf("GET /places%s: %d", q, code)
		}
	}

	get(trav, "?lat=-55.0012&lng=-100.0049")
	get(trav, "?lat=-55.0013&lng=-100.0042") // same ~1 km cell, same day
	get(trav, "")                            // no position
	if n, lat, lng := rows("usr_pres_t"); n != 1 || lat != -5500 || lng != -10000 {
		t.Fatalf("traveller: %d rows at %d,%d; want 1 at -5500,-10000", n, lat, lng)
	}
	get(vend, "?lat=-55.0012&lng=-100.0049")
	if n, _, _ := rows("usr_pres_v"); n != 0 {
		t.Errorf("vendor recorded: %d rows", n)
	}

	// Turning usage analytics off deletes the rows and stops recording, and
	// leaves the other switch alone.
	var me map[string]any
	if code := doReqJSON(t, srv, http.MethodPut, "/me/privacy", trav, map[string]any{"shareLocation": true}, &me); code != http.StatusOK {
		t.Fatalf("share on: %d", code)
	}
	if code := doReqJSON(t, srv, http.MethodPut, "/me/privacy", trav, map[string]any{"usageAnalytics": false}, &me); code != http.StatusOK {
		t.Fatalf("analytics off: %d", code)
	}
	if me["usageAnalytics"] != false || me["shareLocation"] != true {
		t.Errorf("privacy after analytics off: %v", me)
	}
	if n, _, _ := rows("usr_pres_t"); n != 0 {
		t.Errorf("rows kept after opting out: %d", n)
	}
	get(trav, "?lat=-55.0012&lng=-100.0049")
	if n, _, _ := rows("usr_pres_t"); n != 0 {
		t.Errorf("recorded while opted out: %d", n)
	}
	me = nil
	doReqJSON(t, srv, http.MethodGet, "/me", trav, nil, &me)
	if me["usageAnalytics"] != false {
		t.Errorf("/me doesn't report usageAnalytics off: %v", me["usageAnalytics"])
	}

	// Pruning drops days before the cut-off.
	execSQL(t, store, `INSERT INTO traveller_presence (user_id, day, cell_lat, cell_lng) VALUES ('usr_pres_t', (now() AT TIME ZONE 'UTC')::date - 200, -5500, -10000)`)
	if _, err := store.PruneTravellerPresence(ctx, time.Now().Add(-180*24*time.Hour)); err != nil {
		t.Fatalf("prune: %v", err)
	}
	if n, _, _ := rows("usr_pres_t"); n != 0 {
		t.Errorf("old row not pruned: %d", n)
	}
}

func TestAdminGeoHeatLayers(t *testing.T) {
	store, srv := geoTestServer(t)
	ctx := context.Background()
	purge := func() {
		for _, q := range []string{
			`DELETE FROM saved_places WHERE id LIKE 'sav_heat_%'`,
			`DELETE FROM geo_places WHERE id LIKE 'gp_heat_%'`,
			`DELETE FROM checkins WHERE place_id LIKE 'plc_heat_%'`,
			`DELETE FROM reviews WHERE place_id LIKE 'plc_heat_%'`,
			`DELETE FROM trips WHERE owner_id LIKE 'usr_heat_%'`,
			`DELETE FROM places WHERE id LIKE 'plc_heat_%'`,
			`DELETE FROM users WHERE id LIKE 'usr_heat_%'`,
			`DELETE FROM vendors WHERE id LIKE 'ven_heat_%'`,
			`DELETE FROM regions WHERE id = 'geo_heat_r'`,
		} {
			store.Pool.Exec(ctx, q)
		}
	}
	purge()
	t.Cleanup(purge)

	execSQL(t, store, `INSERT INTO regions (id, name, lat, lng) VALUES ('geo_heat_r', 'Heat Town', -50, -140)`)
	old := time.Now().Add(-200 * 24 * time.Hour)
	prev := time.Now().Add(-40 * 24 * time.Hour)
	for _, v := range []struct {
		id, status string
		lat, lng   float64
		submitted  time.Time
	}{
		{"ven_heat_a", "approved", -50.01, -140.01, old},
		{"ven_heat_new1", "pending", -50.02, -140.02, time.Now()},
		{"ven_heat_new2", "approved", -50.02, -140.03, prev},
		{"ven_heat_imp", "approved", -50.03, -140.02, time.Now()},
	} {
		execSQL(t, store, `INSERT INTO vendors (id, name, category, status, lat, lng, submitted_at) VALUES ($1, $1, 'coffee', $2, $3, $4, $5)`,
			v.id, v.status, v.lat, v.lng, v.submitted)
	}
	execSQL(t, store, `INSERT INTO places (id, name, vendor_id, data) VALUES ('plc_heat_a', 'A', 'ven_heat_a', '{"lat":-50.01,"lng":-140.01}')`)

	pw, _ := auth.HashPassword("testpass123")
	imp := "ven_heat_imp"
	users := []domain.User{
		{ID: "usr_heat_admin", Email: "admin@heattest.dev", Role: domain.RoleAdmin},
		{ID: "usr_heat_trav", Email: "trav@heattest.dev", Role: domain.RoleTraveller},
		{ID: "usr_heat_imp", Email: "import_heat@spotto.dev", Role: domain.RoleVendor, VendorID: &imp},
	}
	for i := 2; i <= 5; i++ {
		country := "Testland"
		if i == 5 {
			country = "Otherland"
		}
		users = append(users, domain.User{ID: fmt.Sprintf("usr_heat_t%d", i), Email: fmt.Sprintf("t%d@heattest.dev", i),
			Role: domain.RoleTraveller, Country: country, TravelStyle: []string{"surf"}})
	}
	for _, u := range users {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		must(t, store.CreateUser(ctx, u))
	}
	admin := loginAs(t, srv, "admin@heattest.dev", "testpass123")

	// Travellers: t2-t4 in one ~5 km cell, t5 alone in another; t2 also
	// last period.
	for _, id := range []string{"usr_heat_t2", "usr_heat_t3", "usr_heat_t4"} {
		execSQL(t, store, `INSERT INTO traveller_presence (user_id, day, cell_lat, cell_lng) VALUES ($1, (now() AT TIME ZONE 'UTC')::date, -5000, -14000)`, id)
	}
	execSQL(t, store, `INSERT INTO traveller_presence (user_id, day, cell_lat, cell_lng) VALUES ('usr_heat_t5', (now() AT TIME ZONE 'UTC')::date, -5010, -14010)`)
	execSQL(t, store, `INSERT INTO traveller_presence (user_id, day, cell_lat, cell_lng) VALUES ('usr_heat_t2', ((now() - interval '40 days') AT TIME ZONE 'UTC')::date, -5000, -14000)`)

	// Activity at place A: a check-in, a review and a save now, and a
	// check-in last period.
	execSQL(t, store, `INSERT INTO checkins (place_id, traveller_id) VALUES ('plc_heat_a', 'usr_heat_t2')`)
	execSQL(t, store, `INSERT INTO checkins (place_id, traveller_id, created_at) VALUES ('plc_heat_a', 'usr_heat_t2', now() - interval '40 days')`)
	execSQL(t, store, `INSERT INTO reviews (id, place_id, traveller_id, rating) VALUES ('rev_heat_a', 'plc_heat_a', 'usr_heat_t3', 4)`)
	execSQL(t, store, `INSERT INTO saved_places (id, user_id, place_id) VALUES ('sav_heat_p', 'usr_heat_t4', 'plc_heat_a')`)
	t.Cleanup(func() { store.Pool.Exec(ctx, `DELETE FROM reviews WHERE id = 'rev_heat_a'`) })

	// A place saved from reels that isn't on Spotto, by two travellers.
	execSQL(t, store, `INSERT INTO geo_places (id, google_place_id, name, lat, lng, city) VALUES ('gp_heat_1', 'g_heat_1', 'Hidden Beach Bar', -50.03, -140.03, 'Heat Town')`)
	execSQL(t, store, `INSERT INTO saved_places (id, user_id, geo_place_id) VALUES ('sav_heat_g1', 'usr_heat_t2', 'gp_heat_1'), ('sav_heat_g2', 'usr_heat_t3', 'gp_heat_1')`)

	// Trips: t2 started yesterday (day 0 is past, day 1 is today with a
	// located custom stop, day 2's stop has no location); t3 starts in 5
	// days at place A.
	today := time.Now().UTC()
	execSQL(t, store, `INSERT INTO trips (owner_id, name, code, start_date, days) VALUES ('usr_heat_t2', 'T', 'TRP-H1', $1, $2)`,
		today.AddDate(0, 0, -1).Format("2006-01-02"),
		`[{"id":"d0","stops":[{"id":"s","placeId":"plc_heat_a","time":540}]},
		  {"id":"d1","stops":[{"id":"s","placeId":"","time":540,"custom":{"name":"Cove","lat":-50.04,"lng":-140.04}}]},
		  {"id":"d2","stops":[{"id":"s","placeId":"","time":540,"custom":{"name":"Somewhere","lat":5.9749,"lng":80.4293}}]}]`)
	execSQL(t, store, `INSERT INTO trips (owner_id, name, code, start_date, days) VALUES ('usr_heat_t3', 'T', 'TRP-H2', $1, $2)`,
		today.AddDate(0, 0, 5).Format("2006-01-02"), `[{"id":"d0","stops":[{"id":"s","placeId":"plc_heat_a","time":540}]}]`)

	type heat struct {
		Points []struct {
			Lat, Lng float64
			W        int
		} `json:"points"`
		Total         int `json:"total"`
		PreviousTotal int `json:"previousTotal"`
		Regions       []struct {
			ID         string `json:"id"`
			Value      int    `json:"value"`
			Previous   int    `json:"previous"`
			Businesses int    `json:"businesses"`
			Hidden     bool   `json:"hidden"`
		} `json:"regions"`
		Breakdown map[string]int `json:"breakdown"`
		Wanted    []struct {
			Name   string `json:"name"`
			Savers int    `json:"savers"`
		} `json:"wanted"`
	}
	fetch := func(q string) (h heat, town struct {
		Value, Previous, Businesses int
		Hidden                      bool
	}) {
		t.Helper()
		if code := doReqJSON(t, srv, http.MethodGet, "/admin/geo/heat?"+q, admin, nil, &h); code != http.StatusOK {
			t.Fatalf("heat %s: %d", q, code)
		}
		for _, r := range h.Regions {
			if r.ID == "geo_heat_r" {
				town.Value, town.Previous, town.Businesses, town.Hidden = r.Value, r.Previous, r.Businesses, r.Hidden
			}
		}
		return
	}
	pointsNear := func(h heat) (n, w int) {
		for _, p := range h.Points {
			if p.Lat < -49 && p.Lat > -51 && p.Lng < -139 && p.Lng > -141 {
				n++
				w += p.W
			}
		}
		return
	}

	// Travellers: the lone traveller's cell isn't drawn; the town counts 4.
	h, town := fetch("layer=travellers")
	if n, w := pointsNear(h); n != 1 || w != 3 {
		t.Errorf("travellers: %d points weighing %d, want 1 weighing 3", n, w)
	}
	if town.Value != 4 || town.Hidden || town.Previous != 0 || town.Businesses != 3 {
		t.Errorf("travellers town: %+v, want 4 now, previous hidden as 0, 3 live businesses", town)
	}
	if _, town = fetch("layer=travellers&country=Otherland"); town.Value != 0 || !town.Hidden {
		t.Errorf("one traveller must be hidden: %+v", town)
	}
	if _, town = fetch("layer=travellers&country=Testland&style=surf"); town.Value != 3 {
		t.Errorf("travellers Testland/surf: %+v, want 3", town)
	}

	// Activity.
	h, town = fetch("layer=activity&country=Testland")
	if town.Value != 3 || town.Previous != 1 || h.Breakdown["checkin"] != 1 || h.Breakdown["review"] != 1 || h.Breakdown["save"] != 1 {
		t.Errorf("activity: town %+v, breakdown %v", town, h.Breakdown)
	}
	if _, town = fetch("layer=activity&types=checkin"); town.Value != 1 || town.Previous != 1 {
		t.Errorf("activity checkins: %+v", town)
	}
	if _, town = fetch("layer=activity&category=bars"); town.Value != 0 {
		t.Errorf("activity bars: %+v", town)
	}
	if _, town = fetch("layer=activity&country=Otherland"); town.Value != 0 {
		t.Errorf("activity Otherland: %+v", town)
	}
	if code := doReqJSON(t, srv, http.MethodGet, "/admin/geo/heat?layer=activity&types=nope", admin, nil, nil); code != http.StatusBadRequest {
		t.Errorf("bad type: %d", code)
	}

	// Business sign-ups: imported listings left out unless asked for.
	if _, town = fetch("layer=businesses"); town.Value != 1 || town.Previous != 1 {
		t.Errorf("sign-ups: %+v, want 1 now and 1 before", town)
	}
	if _, town = fetch("layer=businesses&imported=1"); town.Value != 2 {
		t.Errorf("sign-ups with imported: %+v", town)
	}
	if _, town = fetch("layer=businesses&status=approved"); town.Value != 0 {
		t.Errorf("approved sign-ups: %+v", town)
	}

	// Wanted places and upcoming trips.
	h, town = fetch("layer=wanted")
	if town.Value != 2 || len(h.Wanted) == 0 || h.Wanted[0].Name != "Hidden Beach Bar" || h.Wanted[0].Savers != 2 {
		t.Errorf("wanted: town %+v, list %+v", town, h.Wanted)
	}
	if _, town = fetch("layer=trips&days=30"); town.Value != 2 {
		t.Errorf("upcoming trip stops: %+v, want 2", town)
	}
	if _, town = fetch("layer=trips&days=7&country=Testland"); town.Value != 2 {
		t.Errorf("upcoming trip stops (7 days, Testland): %+v, want 2", town)
	}

	// The area around place A.
	var area map[string]any
	if code := doReqJSON(t, srv, http.MethodGet, "/admin/geo/area?lat=-50.01&lng=-140.01&radiusKm=5", admin, nil, &area); code != http.StatusOK {
		t.Fatalf("area: %d", code)
	}
	act, _ := area["activity"].(map[string]any)
	if area["travellers"] != float64(3) || area["liveBusinesses"] != float64(3) || area["newBusinesses"] != float64(1) ||
		area["wantedPlaces"] != float64(1) || area["upcomingStops"] != float64(2) || act["checkin"] != float64(1) {
		t.Errorf("area: %v", area)
	}

	var filters struct {
		Countries []struct {
			Value string
			Count int
		}
	}
	doReqJSON(t, srv, http.MethodGet, "/admin/geo/filters", admin, nil, &filters)
	found := false
	for _, c := range filters.Countries {
		found = found || (c.Value == "Testland" && c.Count == 3)
	}
	if !found {
		t.Errorf("filters: %+v", filters)
	}

	trav := loginAs(t, srv, "trav@heattest.dev", "testpass123")
	if code := doReqJSON(t, srv, http.MethodGet, "/admin/geo/heat?layer=activity", trav, nil, nil); code != http.StatusForbidden {
		t.Errorf("traveller on admin heat: %d", code)
	}
}
