package postgres

// Queries behind the admin geo page's heat maps (GET /admin/geo/heat and
// /admin/geo/area). Each layer returns located rows split into the chosen
// period and the one before it, so the page can show growth; the handler
// buckets them into map points and towns.

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// GeoFilter narrows a heat-map layer. The current period is [Since, Until)
// and the previous one [PrevSince, Since).
type GeoFilter struct {
	Since, Until, PrevSince time.Time
	// Country and Style filter by the traveller's home country and travel
	// style; empty means any.
	Country, Style string
	// Category is a business category id; empty means any.
	Category string
	// Types picks activity kinds (see ActivityTypes); empty means all.
	Types []string
	// Status is a business status for sign-ups; empty means any.
	Status string
	// IncludeImported counts businesses created by the Sri Lanka importer
	// as sign-ups too.
	IncludeImported bool
}

// ActivityTypes are the traveller actions the activity layer can show.
var ActivityTypes = []string{"checkin", "vibe", "review", "save", "redemption", "booking"}

// GeoCount is a located count for the current and the previous period.
type GeoCount struct {
	Lat, Lng  float64
	Cur, Prev int
	Kind      string // activity type, for the activity layer
}

// PresenceCell is one traveller seen in one ~5 km cell, and in which of the
// two periods.
type PresenceCell struct {
	Lat, Lng  float64 // cell centre
	UserID    string
	Cur, Prev bool
}

// travellerFilterSQL matches users u against GeoFilter's country ($n) and
// style ($n+1).
const travellerFilterSQL = `($%[1]d = '' OR u.country = $%[1]d) AND ($%[2]d = '' OR u.travel_style ? $%[2]d)`

func travellerFilter(first int) string {
	return fmt.Sprintf(travellerFilterSQL, first, first+1)
}

// TravellerPresence lists, per ~5 km cell (0.05°), each traveller seen
// there in either period. Callers count distinct travellers and must hide
// small counts.
func (s *Store) TravellerPresence(ctx context.Context, f GeoFilter) ([]PresenceCell, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT (floor(tp.cell_lat / 5.0) * 5 + 2.5) / 100.0, (floor(tp.cell_lng / 5.0) * 5 + 2.5) / 100.0,
			tp.user_id, bool_or(tp.day >= $1::date), bool_or(tp.day < $1::date)
		FROM traveller_presence tp JOIN users u ON u.id = tp.user_id
		WHERE tp.day >= $2::date AND tp.day < $3::date AND `+travellerFilter(4)+`
		GROUP BY 1, 2, 3`,
		dateOf(f.Since), dateOf(f.PrevSince), dateOf(f.Until.Add(24*time.Hour)), f.Country, f.Style)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PresenceCell
	for rows.Next() {
		var c PresenceCell
		if err := rows.Scan(&c.Lat, &c.Lng, &c.UserID, &c.Cur, &c.Prev); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// activitySources is one SELECT per activity type: place, traveller, when.
var activitySources = map[string]string{
	"checkin":    `SELECT 'checkin', place_id, traveller_id, created_at FROM checkins`,
	"vibe":       `SELECT 'vibe', place_id, traveller_id, created_at FROM vibe_checkins`,
	"review":     `SELECT 'review', place_id, traveller_id, created_at FROM reviews`,
	"save":       `SELECT 'save', place_id, user_id, created_at FROM saved_places WHERE place_id IS NOT NULL`,
	"redemption": `SELECT 'redemption', place_id, traveller_id, redeemed_at FROM claims WHERE status = 'redeemed' AND redeemed_at IS NOT NULL`,
	"booking":    `SELECT 'booking', place_id, traveller_id, created_at FROM bookings WHERE place_id <> ''`,
}

// PlaceActivity counts traveller actions per place and type, in both
// periods, at the place's location. Category matches the place's business
// category (or its own category when it has no business).
func (s *Store) PlaceActivity(ctx context.Context, f GeoFilter) ([]GeoCount, error) {
	kinds := f.Types
	if len(kinds) == 0 {
		kinds = ActivityTypes
	}
	union := ""
	for _, k := range kinds {
		src, ok := activitySources[k]
		if !ok {
			continue
		}
		if union != "" {
			union += " UNION ALL "
		}
		union += src
	}
	if union == "" {
		return nil, nil
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT (p.data->>'lat')::float8, (p.data->>'lng')::float8, a.kind,
			count(*) FILTER (WHERE a.at >= $1)::int, count(*) FILTER (WHERE a.at < $1)::int
		FROM (`+union+`) a(kind, place_id, user_id, at)
		JOIN places p ON p.id = a.place_id
		LEFT JOIN vendors v ON v.id = p.vendor_id
		JOIN users u ON u.id = a.user_id
		WHERE a.at >= $2 AND a.at < $3
			AND jsonb_typeof(p.data->'lat') = 'number' AND jsonb_typeof(p.data->'lng') = 'number'
			AND ($4 = '' OR COALESCE(v.category, p.data->>'categoryId', '') = $4)
			AND `+travellerFilter(5)+`
		GROUP BY 1, 2, 3`,
		f.Since, f.PrevSince, f.Until, f.Category, f.Country, f.Style)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GeoCount
	for rows.Next() {
		var c GeoCount
		if err := rows.Scan(&c.Lat, &c.Lng, &c.Kind, &c.Cur, &c.Prev); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// importedVendorSQL is true for a business the Sri Lanka importer created
// (its owner account is an import_…@spotto.dev placeholder).
const importedVendorSQL = `EXISTS (SELECT 1 FROM users iu WHERE iu.vendor_id = v.id AND iu.email LIKE 'import\_%@spotto.dev')`

// BusinessSignups lists business sign-ups (by submitted_at) in both periods
// at the business's pin. Imported listings are left out unless asked for.
func (s *Store) BusinessSignups(ctx context.Context, f GeoFilter) ([]GeoCount, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT v.lat, v.lng,
			count(*) FILTER (WHERE v.submitted_at >= $1)::int, count(*) FILTER (WHERE v.submitted_at < $1)::int
		FROM vendors v
		WHERE (v.lat <> 0 OR v.lng <> 0) AND v.submitted_at >= $2 AND v.submitted_at < $3
			AND ($4 = '' OR v.category = $4) AND ($5 = '' OR v.status = $5)
			AND ($6 OR NOT `+importedVendorSQL+`)
		GROUP BY 1, 2`,
		f.Since, f.PrevSince, f.Until, f.Category, f.Status, f.IncludeImported)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GeoCount
	for rows.Next() {
		var c GeoCount
		if err := rows.Scan(&c.Lat, &c.Lng, &c.Cur, &c.Prev); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LiveBusinesses lists approved businesses with a location, optionally of
// one category — the supply side of the opportunity view.
func (s *Store) LiveBusinesses(ctx context.Context, category string) ([]Point, error) {
	return collectPoints(ctx, s, `
		SELECT lat, lng, 1 FROM vendors
		WHERE status = 'approved' AND (lat <> 0 OR lng <> 0) AND ($1 = '' OR category = $1)`, category)
}

// WantedPlace is a place travellers saved from a reel that isn't a Spotto
// listing — a lead for sales.
type WantedPlace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	Kind      string    `json:"kind"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	MapsURL   string    `json:"mapsUrl"`
	Savers    int       `json:"savers"`
	Previous  int       `json:"previous"`
	LastSaved time.Time `json:"lastSaved"`
}

// WantedPlaces lists places saved from reels that aren't on Spotto, with
// distinct savers in both periods, most saved first.
func (s *Store) WantedPlaces(ctx context.Context, f GeoFilter) ([]WantedPlace, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT gp.id, gp.name, gp.city, gp.kind, gp.lat, gp.lng, gp.maps_url,
			count(DISTINCT sp.user_id) FILTER (WHERE sp.created_at >= $1)::int,
			count(DISTINCT sp.user_id) FILTER (WHERE sp.created_at < $1)::int,
			max(sp.created_at)
		FROM saved_places sp
		JOIN geo_places gp ON gp.id = sp.geo_place_id
		JOIN users u ON u.id = sp.user_id
		WHERE sp.created_at >= $2 AND sp.created_at < $3 AND `+travellerFilter(4)+`
		GROUP BY gp.id
		ORDER BY 8 DESC, 10 DESC`,
		f.Since, f.PrevSince, f.Until, f.Country, f.Style)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WantedPlace
	for rows.Next() {
		var w WantedPlace
		if err := rows.Scan(&w.ID, &w.Name, &w.City, &w.Kind, &w.Lat, &w.Lng, &w.MapsURL, &w.Savers, &w.Previous, &w.LastSaved); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// UpcomingTripStop is one stop of a traveller's current trip, on its day.
type UpcomingTripStop struct {
	Lat, Lng float64
	Day      time.Time
}

// noLocationLat/Lng is the app's default position for a typed-in stop that
// has no real location (CustomStopInfo in the app); such stops are skipped.
const noLocationLat, noLocationLng = 5.9749, 80.4293

// UpcomingTripStops lists the stops of travellers' trips that fall in
// [from, to): a trip's day i is its start date plus i days. Stops resolve to
// their place's location, or their own when typed in with one.
func (s *Store) UpcomingTripStops(ctx context.Context, f GeoFilter, from, to time.Time) ([]UpcomingTripStop, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT t.start_date, t.days FROM trips t JOIN users u ON u.id = t.owner_id
		WHERE t.start_date ~ '^\d{4}-\d{2}-\d{2}$' AND t.start_date::date < $1::date
			AND t.start_date::date + jsonb_array_length(t.days) > $2::date
			AND jsonb_typeof(t.days) = 'array' AND `+travellerFilter(3),
		dateOf(to), dateOf(from), f.Country, f.Style)
	if err != nil {
		return nil, err
	}
	type stop struct {
		PlaceID string `json:"placeId"`
		Custom  *struct {
			Lat *float64 `json:"lat"`
			Lng *float64 `json:"lng"`
		} `json:"custom"`
	}
	type pending struct {
		placeID  string
		lat, lng float64
		day      time.Time
	}
	var stops []pending
	placeIDs := map[string]bool{}
	for rows.Next() {
		var start string
		var raw []byte
		if err := rows.Scan(&start, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		startDay, err := time.Parse("2006-01-02", start)
		if err != nil {
			continue
		}
		var days []struct {
			Stops []stop `json:"stops"`
		}
		if json.Unmarshal(raw, &days) != nil {
			continue
		}
		for i, d := range days {
			day := startDay.AddDate(0, 0, i)
			if day.Before(dateOnly(from)) || !day.Before(dateOnly(to)) {
				continue
			}
			for _, st := range d.Stops {
				switch {
				case st.Custom != nil && st.Custom.Lat != nil && st.Custom.Lng != nil:
					if *st.Custom.Lat == noLocationLat && *st.Custom.Lng == noLocationLng {
						continue
					}
					stops = append(stops, pending{lat: *st.Custom.Lat, lng: *st.Custom.Lng, day: day})
				case st.PlaceID != "":
					stops = append(stops, pending{placeID: st.PlaceID, day: day})
					placeIDs[st.PlaceID] = true
				}
			}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	locs := map[string][2]float64{}
	if len(placeIDs) > 0 {
		ids := make([]string, 0, len(placeIDs))
		for id := range placeIDs {
			ids = append(ids, id)
		}
		prow, err := s.Pool.Query(ctx, `
			SELECT id, (data->>'lat')::float8, (data->>'lng')::float8 FROM places
			WHERE id = ANY($1) AND jsonb_typeof(data->'lat') = 'number' AND jsonb_typeof(data->'lng') = 'number'`, ids)
		if err != nil {
			return nil, err
		}
		for prow.Next() {
			var id string
			var lat, lng float64
			if err := prow.Scan(&id, &lat, &lng); err != nil {
				prow.Close()
				return nil, err
			}
			locs[id] = [2]float64{lat, lng}
		}
		prow.Close()
	}
	var out []UpcomingTripStop
	for _, p := range stops {
		if p.placeID != "" {
			l, ok := locs[p.placeID]
			if !ok {
				continue
			}
			p.lat, p.lng = l[0], l[1]
		}
		if p.lat == 0 && p.lng == 0 {
			continue
		}
		out = append(out, UpcomingTripStop{Lat: p.lat, Lng: p.lng, Day: p.day})
	}
	return out, nil
}

// GeoFilterOptions are the values the geo page's filters offer.
type GeoFilterOptions struct {
	Countries []GeoFilterCount `json:"countries"`
	Styles    []GeoFilterCount `json:"styles"`
}

type GeoFilterCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

// GeoFilters lists travellers' home countries and travel styles, most
// common first.
func (s *Store) GeoFilters(ctx context.Context) (GeoFilterOptions, error) {
	out := GeoFilterOptions{Countries: []GeoFilterCount{}, Styles: []GeoFilterCount{}}
	for _, q := range []struct {
		sql string
		dst *[]GeoFilterCount
	}{
		{`SELECT country, count(*)::int FROM users WHERE role = 'traveller' AND country <> '' GROUP BY 1 ORDER BY 2 DESC, 1`, &out.Countries},
		{`SELECT st, count(*)::int FROM users, jsonb_array_elements_text(CASE WHEN jsonb_typeof(travel_style) = 'array' THEN travel_style ELSE '[]'::jsonb END) st
			WHERE role = 'traveller' GROUP BY 1 ORDER BY 2 DESC, 1`, &out.Styles},
	} {
		rows, err := s.Pool.Query(ctx, q.sql)
		if err != nil {
			return out, err
		}
		for rows.Next() {
			var c GeoFilterCount
			if err := rows.Scan(&c.Value, &c.Count); err != nil {
				rows.Close()
				return out, err
			}
			*q.dst = append(*q.dst, c)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return out, err
		}
	}
	return out, nil
}

func dateOnly(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func dateOf(t time.Time) string { return t.UTC().Format("2006-01-02") }
