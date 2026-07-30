// Package geo holds small geography helpers shared across server packages
// that can't import each other directly — notably cmd/importsrilanka (a
// separate main package) and internal/http.
package geo

import "math"

// HaversineKm is the great-circle distance in km between two lat/lng
// points — same formula as the SQL version used by VendorsNear/
// NearbyTravellers, just applied in Go for callers whose lat/lng live
// inside a JSONB data blob rather than dedicated columns.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	x := math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Cos(toRad(lng2)-toRad(lng1)) +
		math.Sin(toRad(lat1))*math.Sin(toRad(lat2))
	return 6371 * math.Acos(math.Min(1, math.Max(-1, x)))
}

// Region is a coarse geographic bucket for the imported Sri Lanka spots
// dataset. This duplicates the identifying fields of the 12 regions
// seeded into the `regions` table (internal/seed/seed.go, for the admin
// map widget) as a static list rather than a DB read: importsrilanka runs
// automatically on every deploy before the API starts, but `seed` (which
// populates that table) is a manual, on-demand command — the regions
// table isn't guaranteed to be populated yet when importsrilanka runs.
type Region struct {
	ID   string
	Name string
	Lat  float64
	Lng  float64
}

var Regions = []Region{
	{ID: "colombo", Name: "Colombo", Lat: 6.9271, Lng: 79.8612},
	{ID: "negombo", Name: "Negombo", Lat: 7.2083, Lng: 79.8358},
	{ID: "kandy", Name: "Kandy", Lat: 7.2906, Lng: 80.6337},
	{ID: "sigiriya", Name: "Sigiriya", Lat: 7.9570, Lng: 80.7603},
	{ID: "ella", Name: "Ella", Lat: 6.8667, Lng: 81.0466},
	{ID: "nuwara", Name: "Nuwara Eliya", Lat: 6.9497, Lng: 80.7891},
	{ID: "galle", Name: "Galle", Lat: 6.0535, Lng: 80.2210},
	{ID: "weligama", Name: "Weligama", Lat: 5.9724, Lng: 80.4292},
	{ID: "mirissa", Name: "Mirissa", Lat: 5.9483, Lng: 80.4589},
	{ID: "arugam", Name: "Arugam Bay", Lat: 6.8404, Lng: 81.8360},
	{ID: "trinco", Name: "Trincomalee", Lat: 8.5874, Lng: 81.2152},
	{ID: "jaffna", Name: "Jaffna", Lat: 9.6615, Lng: 80.0255},
}

// NearestRegion returns the seeded region whose center is closest (great-
// circle distance) to lat/lng — gives each imported place a real region
// instead of a hardcoded one.
func NearestRegion(lat, lng float64) Region {
	best := Regions[0]
	bestDist := HaversineKm(lat, lng, best.Lat, best.Lng)
	for _, r := range Regions[1:] {
		d := HaversineKm(lat, lng, r.Lat, r.Lng)
		if d < bestDist {
			best, bestDist = r, d
		}
	}
	return best
}
