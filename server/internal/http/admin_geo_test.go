package httpapi_test

// Geo-analytics data: businesses stuck in the catch-all "uncategorized"
// category get their real one, and the Opportunity leaderboard's numbers
// are live (businesses and recent traveller activity per region).

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
)

func TestRecategorizeUncategorizedVendors(t *testing.T) {
	store := mustConnect(t)
	loadCategories(t, store)
	ctx := context.Background()
	purge := func() {
		store.Pool.Exec(ctx, `DELETE FROM places WHERE id LIKE 'plc_recat_%'`)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id LIKE 'ven_recat_%'`)
	}
	purge()
	t.Cleanup(purge)
	execSQL(t, store, `INSERT INTO vendor_categories (id, label, segment, icon) VALUES ('uncategorized', 'Uncategorized', 'General', 'storefront') ON CONFLICT (id) DO NOTHING`)
	for _, id := range []string{"ven_recat_tags", "ven_recat_id", "ven_recat_noplace"} {
		execSQL(t, store, `INSERT INTO vendors (id, name, category, status) VALUES ($1, $1, 'uncategorized', 'approved')`, id)
	}
	execSQL(t, store, `INSERT INTO places (id, name, vendor_id, data) VALUES ('plc_recat_tags', 'Tags', 'ven_recat_tags', '{"type":"food","tags":["Cafe"]}')`)
	execSQL(t, store, `INSERT INTO places (id, name, vendor_id, data) VALUES ('plc_recat_id', 'Id', 'ven_recat_id', '{"categoryId":"bars"}')`)

	n, err := store.RecategorizeUncategorizedVendors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n < 2 {
		t.Fatalf("moved %d businesses, want at least 2", n)
	}
	for id, want := range map[string]string{"ven_recat_tags": "coffee", "ven_recat_id": "bars", "ven_recat_noplace": "uncategorized"} {
		var got string
		store.Pool.QueryRow(ctx, `SELECT category FROM vendors WHERE id=$1`, id).Scan(&got)
		if got != want {
			t.Errorf("%s: category %q, want %q", id, got, want)
		}
	}
	var data map[string]any
	var raw []byte
	store.Pool.QueryRow(ctx, `SELECT data FROM places WHERE id='plc_recat_tags'`).Scan(&raw)
	json.Unmarshal(raw, &data)
	if data["categoryId"] != "coffee" || data["color"] != "coral" {
		t.Errorf("place data not brought in line: %v", data)
	}
	// Running again changes nothing.
	if n, err := store.RecategorizeUncategorizedVendors(ctx); err != nil || n != 0 {
		t.Fatalf("second run: %d, %v", n, err)
	}
}

func TestAdminRegionsAreLive(t *testing.T) {
	store := mustConnect(t)
	ctx := context.Background()
	purge := func() {
		store.Pool.Exec(ctx, `DELETE FROM checkins WHERE place_id LIKE 'plc_geo_%'`)
		store.Pool.Exec(ctx, `DELETE FROM reviews WHERE place_id LIKE 'plc_geo_%'`)
		store.Pool.Exec(ctx, `DELETE FROM saved_places WHERE place_id LIKE 'plc_geo_%'`)
		store.Pool.Exec(ctx, `DELETE FROM places WHERE id LIKE 'plc_geo_%'`)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id LIKE 'ven_geo_%'`)
		store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_geo_%'`)
		store.Pool.Exec(ctx, `DELETE FROM regions WHERE id LIKE 'geo_test_%'`)
	}
	purge()
	t.Cleanup(purge)

	// Two regions far out at sea, so nothing else in the database is
	// nearer to them than to a real town.
	execSQL(t, store, `INSERT INTO regions (id, name, lat, lng) VALUES ('geo_test_a', 'Test A', -60, -150), ('geo_test_b', 'Test B', -60, -120)`)
	for _, v := range []struct {
		id, status string
		lat, lng   float64
	}{
		{"ven_geo_a1", "approved", -60.01, -150.01},
		{"ven_geo_a2", "approved", -59.99, -149.99},
		{"ven_geo_a3", "pending", -60.02, -150.02}, // not live: doesn't count
		{"ven_geo_b1", "approved", -60.01, -120.01},
	} {
		execSQL(t, store, `INSERT INTO vendors (id, name, category, status, lat, lng) VALUES ($1, $1, 'coffee', $2, $3, $4)`, v.id, v.status, v.lat, v.lng)
	}
	execSQL(t, store, `INSERT INTO places (id, name, data) VALUES
		('plc_geo_a', 'A', '{"lat":-60.01,"lng":-150.01}'),
		('plc_geo_b', 'B', '{"lat":-60.01,"lng":-120.01}')`)
	pw, _ := auth.HashPassword("testpass123")
	for _, u := range []domain.User{
		{ID: "usr_geo_admin", Email: "admin@geotest.dev", Role: domain.RoleAdmin},
		{ID: "usr_geo_t1", Email: "t1@geotest.dev", Role: domain.RoleTraveller},
		{ID: "usr_geo_t2", Email: "t2@geotest.dev", Role: domain.RoleTraveller},
	} {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		must(t, store.CreateUser(ctx, u))
	}
	// Region A: two check-ins and a review this month, plus an old
	// check-in that's outside the 30 days. Region B: one save.
	execSQL(t, store, `INSERT INTO checkins (place_id, traveller_id) VALUES ('plc_geo_a', 'usr_geo_t1'), ('plc_geo_a', 'usr_geo_t2')`)
	execSQL(t, store, `INSERT INTO checkins (place_id, traveller_id, created_at) VALUES ('plc_geo_a', 'usr_geo_t1', now() - interval '40 days')`)
	execSQL(t, store, `INSERT INTO reviews (id, place_id, traveller_id, rating) VALUES ('rev_geo_a', 'plc_geo_a', 'usr_geo_t1', 5)`)
	execSQL(t, store, `INSERT INTO saved_places (id, user_id, place_id) VALUES ('sav_geo_b', 'usr_geo_t2', 'plc_geo_b')`)

	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32)}
	srv := httptest.NewServer(httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg).Router(cfg))
	t.Cleanup(srv.Close)
	tok := loginAs(t, srv, "admin@geotest.dev", "testpass123")

	var regions []domain.Region
	if code := doReqJSON(t, srv, http.MethodGet, "/admin/regions", tok, nil, &regions); code != http.StatusOK {
		t.Fatalf("GET /admin/regions: %d", code)
	}
	got := map[string]domain.Region{}
	for _, r := range regions {
		got[r.ID] = r
	}
	if a := got["geo_test_a"]; a.Vendors != 2 || a.Travelers != 3 {
		t.Errorf("region A: %d businesses, %d traveller actions; want 2 and 3", a.Vendors, a.Travelers)
	}
	if b := got["geo_test_b"]; b.Vendors != 1 || b.Travelers != 1 {
		t.Errorf("region B: %d businesses, %d traveller actions; want 1 and 1", b.Vendors, b.Travelers)
	}
}
