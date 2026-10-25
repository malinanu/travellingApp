package httpapi

// Admin geo page: heat-map layers (where travellers are, what they do,
// where businesses sign up, gaps and demand), the filters' options, and a
// summary of the area around a clicked point. Every layer compares the
// chosen period with the one before it and ranks towns (nearestRegion).

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/store/postgres"
)

// geoMinTravellers is the fewest travellers an area may show; smaller
// counts are hidden so nobody can be singled out.
const geoMinTravellers = 3

var geoPeriods = map[int]bool{7: true, 30: true, 90: true}

var geoStatuses = map[string]bool{
	"": true, string(domain.VendorPending): true, string(domain.VendorApproved): true,
	string(domain.VendorEdits): true, string(domain.VendorSuspended): true, string(domain.VendorRejected): true,
}

type geoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	W   int     `json:"w"`
}

type geoRegionStat struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	Value      int     `json:"value"`
	Previous   int     `json:"previous"`
	Businesses int     `json:"businesses"`
	// Hidden is set when a traveller count was too small to show.
	Hidden bool `json:"hidden,omitempty"`
}

type geoHeatResp struct {
	Layer         string                 `json:"layer"`
	Days          int                    `json:"days"`
	Points        []geoPoint             `json:"points"`
	Total         int                    `json:"total"`
	PreviousTotal int                    `json:"previousTotal"`
	Regions       []geoRegionStat        `json:"regions"`
	Breakdown     map[string]int         `json:"breakdown,omitempty"`
	Wanted        []postgres.WantedPlace `json:"wanted,omitempty"`
}

// parseGeoFilter reads the period (days, default 30) and filters shared by
// the geo endpoints.
func parseGeoFilter(r *http.Request) (postgres.GeoFilter, int, string) {
	q := r.URL.Query()
	days, _ := strconv.Atoi(q.Get("days"))
	if !geoPeriods[days] {
		days = 30
	}
	now := time.Now().UTC()
	period := time.Duration(days) * 24 * time.Hour
	f := postgres.GeoFilter{
		Until:           now,
		Since:           now.Add(-period),
		PrevSince:       now.Add(-2 * period),
		Country:         strings.TrimSpace(q.Get("country")),
		Style:           strings.TrimSpace(q.Get("style")),
		Category:        strings.TrimSpace(q.Get("category")),
		Status:          q.Get("status"),
		IncludeImported: q.Get("imported") == "1",
	}
	if !geoStatuses[f.Status] {
		return f, days, "unknown status"
	}
	if t := q.Get("types"); t != "" {
		known := map[string]bool{}
		for _, k := range postgres.ActivityTypes {
			known[k] = true
		}
		for _, k := range strings.Split(t, ",") {
			if !known[k] {
				return f, days, "unknown activity type " + k
			}
			f.Types = append(f.Types, k)
		}
	}
	return f, days, ""
}

// handleAdminGeoHeat returns one heat-map layer: weighted points for the
// map, the period's total and the previous period's, and every town's value
// (with its live businesses) for the rankings.
func (s *Server) handleAdminGeoHeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	f, days, bad := parseGeoFilter(r)
	if bad != "" {
		writeErr(w, http.StatusBadRequest, bad)
		return
	}
	layer := r.URL.Query().Get("layer")
	regions, err := s.Store.Regions(ctx)
	if err != nil {
		storeErr(w, err)
		return
	}
	stats := make([]geoRegionStat, len(regions))
	for i, rg := range regions {
		stats[i] = geoRegionStat{ID: rg.ID, Name: rg.Name, Lat: rg.Lat, Lng: rg.Lng}
	}
	live, err := s.Store.LiveBusinesses(ctx, f.Category)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, p := range live {
		if i := nearestRegion(regions, p.Lat, p.Lng); i >= 0 {
			stats[i].Businesses++
		}
	}
	resp := geoHeatResp{Layer: layer, Days: days, Points: []geoPoint{}}

	// addCounts sums located current/previous counts into points, towns and totals.
	addCounts := func(lat, lng float64, cur, prev int) {
		if cur > 0 {
			resp.Points = append(resp.Points, geoPoint{Lat: lat, Lng: lng, W: cur})
		}
		resp.Total += cur
		resp.PreviousTotal += prev
		if i := nearestRegion(regions, lat, lng); i >= 0 {
			stats[i].Value += cur
			stats[i].Previous += prev
		}
	}

	switch layer {
	case "travellers":
		cells, err := s.Store.TravellerPresence(ctx, f)
		if err != nil {
			storeErr(w, err)
			return
		}
		type key struct{ lat, lng float64 }
		cellCur := map[key]map[string]bool{}
		regionCur := make([]map[string]bool, len(regions))
		regionPrev := make([]map[string]bool, len(regions))
		allCur, allPrev := map[string]bool{}, map[string]bool{}
		add := func(m map[string]bool, id string) map[string]bool {
			if m == nil {
				m = map[string]bool{}
			}
			m[id] = true
			return m
		}
		for _, c := range cells {
			i := nearestRegion(regions, c.Lat, c.Lng)
			if c.Cur {
				k := key{c.Lat, c.Lng}
				cellCur[k] = add(cellCur[k], c.UserID)
				allCur[c.UserID] = true
				if i >= 0 {
					regionCur[i] = add(regionCur[i], c.UserID)
				}
			}
			if c.Prev {
				allPrev[c.UserID] = true
				if i >= 0 {
					regionPrev[i] = add(regionPrev[i], c.UserID)
				}
			}
		}
		for k, users := range cellCur {
			if len(users) >= geoMinTravellers {
				resp.Points = append(resp.Points, geoPoint{Lat: k.lat, Lng: k.lng, W: len(users)})
			}
		}
		for i := range stats {
			stats[i].Value, stats[i].Hidden = shownTravellers(len(regionCur[i]))
			stats[i].Previous, _ = shownTravellers(len(regionPrev[i]))
		}
		resp.Total, resp.PreviousTotal = len(allCur), len(allPrev)

	case "activity", "opportunity":
		counts, err := s.Store.PlaceActivity(ctx, f)
		if err != nil {
			storeErr(w, err)
			return
		}
		resp.Breakdown = map[string]int{}
		for _, c := range counts {
			addCounts(c.Lat, c.Lng, c.Cur, c.Prev)
			resp.Breakdown[c.Kind] += c.Cur
		}
		if layer == "opportunity" {
			resp.Points = []geoPoint{} // drawn as town circles
		}

	case "businesses":
		counts, err := s.Store.BusinessSignups(ctx, f)
		if err != nil {
			storeErr(w, err)
			return
		}
		for _, c := range counts {
			addCounts(c.Lat, c.Lng, c.Cur, c.Prev)
		}

	case "wanted":
		wanted, err := s.Store.WantedPlaces(ctx, f)
		if err != nil {
			storeErr(w, err)
			return
		}
		for _, p := range wanted {
			addCounts(p.Lat, p.Lng, p.Savers, p.Previous)
		}
		resp.Wanted = []postgres.WantedPlace{}
		for _, p := range wanted {
			if p.Savers > 0 && len(resp.Wanted) < 50 {
				resp.Wanted = append(resp.Wanted, p)
			}
		}

	case "trips":
		// Trips look ahead: the next `days` days, nothing to compare with.
		today := time.Now().UTC()
		stops, err := s.Store.UpcomingTripStops(ctx, f, today, today.AddDate(0, 0, days))
		if err != nil {
			storeErr(w, err)
			return
		}
		for _, st := range stops {
			addCounts(st.Lat, st.Lng, 1, 0)
		}

	default:
		writeErr(w, http.StatusBadRequest, "layer must be travellers, activity, opportunity, businesses, wanted or trips")
		return
	}
	resp.Regions = stats
	writeJSON(w, http.StatusOK, resp)
}

// shownTravellers hides a traveller count below geoMinTravellers.
func shownTravellers(n int) (int, bool) {
	if n > 0 && n < geoMinTravellers {
		return 0, true
	}
	return n, false
}

// handleAdminGeoFilters lists the values the geo page's filters offer.
func (s *Server) handleAdminGeoFilters(w http.ResponseWriter, r *http.Request) {
	opts, err := s.Store.GeoFilters(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"countries":     opts.Countries,
		"styles":        opts.Styles,
		"activityTypes": postgres.ActivityTypes,
	})
}

type geoAreaResp struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	RadiusKm float64 `json:"radiusKm"`
	Days     int     `json:"days"`
	// Travellers is 0 with TravellersHidden set when too few to show.
	Travellers       int            `json:"travellers"`
	TravellersHidden bool           `json:"travellersHidden,omitempty"`
	Activity         map[string]int `json:"activity"`
	LiveBusinesses   int            `json:"liveBusinesses"`
	NewBusinesses    int            `json:"newBusinesses"`
	WantedPlaces     int            `json:"wantedPlaces"`
	UpcomingStops    int            `json:"upcomingStops"`
}

// handleAdminGeoArea sums every layer within radiusKm (default 5, max 50)
// of a clicked point, for the chosen period and filters.
func (s *Server) handleAdminGeoArea(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	lat, err1 := strconv.ParseFloat(q.Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(q.Get("lng"), 64)
	if err1 != nil || err2 != nil {
		writeErr(w, http.StatusBadRequest, "lat and lng are required")
		return
	}
	radius := 5.0
	if v, err := strconv.ParseFloat(q.Get("radiusKm"), 64); err == nil && v > 0 && v <= 50 {
		radius = v
	}
	f, days, bad := parseGeoFilter(r)
	if bad != "" {
		writeErr(w, http.StatusBadRequest, bad)
		return
	}
	near := func(pLat, pLng float64) bool { return geo.HaversineKm(lat, lng, pLat, pLng) <= radius }
	resp := geoAreaResp{Lat: lat, Lng: lng, RadiusKm: radius, Days: days, Activity: map[string]int{}}

	cells, err := s.Store.TravellerPresence(ctx, f)
	if err != nil {
		storeErr(w, err)
		return
	}
	travellers := map[string]bool{}
	for _, c := range cells {
		if c.Cur && near(c.Lat, c.Lng) {
			travellers[c.UserID] = true
		}
	}
	resp.Travellers, resp.TravellersHidden = shownTravellers(len(travellers))

	activity, err := s.Store.PlaceActivity(ctx, f)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, c := range activity {
		if c.Cur > 0 && near(c.Lat, c.Lng) {
			resp.Activity[c.Kind] += c.Cur
		}
	}
	live, err := s.Store.LiveBusinesses(ctx, f.Category)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, p := range live {
		if near(p.Lat, p.Lng) {
			resp.LiveBusinesses++
		}
	}
	signups, err := s.Store.BusinessSignups(ctx, f)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, c := range signups {
		if near(c.Lat, c.Lng) {
			resp.NewBusinesses += c.Cur
		}
	}
	wanted, err := s.Store.WantedPlaces(ctx, f)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, p := range wanted {
		if p.Savers > 0 && near(p.Lat, p.Lng) {
			resp.WantedPlaces++
		}
	}
	today := time.Now().UTC()
	stops, err := s.Store.UpcomingTripStops(ctx, f, today, today.AddDate(0, 0, days))
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, st := range stops {
		if near(st.Lat, st.Lng) {
			resp.UpcomingStops++
		}
	}
	writeJSON(w, http.StatusOK, resp)
}
