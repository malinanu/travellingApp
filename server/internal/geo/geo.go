// Package geo holds small geography helpers shared across server packages
// that can't import each other directly — notably cmd/importsrilanka (a
// separate main package) and internal/http.
package geo

import (
	"math"
	"strings"

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

// FindRegion searches seeded regions by name or ID (case-insensitive, space/punctuation-tolerant).
func FindRegion(query string) *domain.Region {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	clean := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
	qClean := clean(q)
	if qClean == "" {
		return nil
	}

	// 1. Exact match on raw or clean name/id
	for i := range seed.Regions {
		r := &seed.Regions[i]
		rName := strings.ToLower(r.Name)
		rID := strings.ToLower(r.ID)
		if q == rName || q == rID || qClean == clean(rName) || qClean == clean(rID) {
			return r
		}
	}

	// 2. Query contains region name or region clean name (prefer longest match to avoid sub-word false positives)
	var bestMatch *domain.Region
	bestLen := 0
	for i := range seed.Regions {
		r := &seed.Regions[i]
		rName := strings.ToLower(r.Name)
		rID := strings.ToLower(r.ID)
		rClean := clean(rName)
		idClean := clean(rID)

		matched := false
		matchLen := 0
		if strings.Contains(q, rName) {
			matched = true
			matchLen = len(rName)
		} else if len(rClean) >= 4 && strings.Contains(qClean, rClean) {
			matched = true
			matchLen = len(rClean)
		} else if len(rID) >= 4 && (strings.Contains(q, rID) || strings.Contains(qClean, idClean)) {
			matched = true
			matchLen = len(rID)
		}

		if matched && matchLen > bestLen {
			bestMatch = r
			bestLen = matchLen
		}
	}
	return bestMatch
}

// HaversineKm is the great-circle distance in km between two lat/lng
// points — same formula as the SQL version used by
// NearbyTravellers, just applied in Go for callers whose lat/lng live
// inside a JSONB data blob rather than dedicated columns.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	x := math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Cos(toRad(lng2)-toRad(lng1)) +
		math.Sin(toRad(lat1))*math.Sin(toRad(lat2))
	return 6371 * math.Acos(math.Min(1, math.Max(-1, x)))
}
