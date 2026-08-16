// Package geo holds small geography helpers shared across server packages
// that can't import each other directly — notably cmd/importsrilanka (a
// separate main package) and internal/http.
package geo

import (
	"math"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/seed"
)

// NearestRegion returns the seeded region whose center is closest (great-
// circle distance) to lat/lng — gives each imported place a real region
// instead of a hardcoded one.
func NearestRegion(lat, lng float64) domain.Region {
	best := seed.Regions[0]
	bestDist := HaversineKm(lat, lng, best.Lat, best.Lng)
	for _, r := range seed.Regions[1:] {
		d := HaversineKm(lat, lng, r.Lat, r.Lng)
		if d < bestDist {
			best, bestDist = r, d
		}
	}
	return best
}

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

