package httpapi

// Location and place shaping for the Local Guide: what the model is told
// about where the traveller is, what search_places hands back to it, and
// the place cards the app shows under a reply. Kept as small pure functions
// so they can be tested without a model or a database.

import (
	"encoding/json"
	"math"
	"sort"
	"strings"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/recommend"
)

// guideMaxPlaceCards bounds the cards shown under one reply.
const guideMaxPlaceCards = 4

// guideLocation is where the traveller is, as far as the guide knows.
// Known is false when the app couldn't get a real fix (permission denied,
// location off) — lat/lng are then zero so every tool treats the location
// as unknown rather than ranking around a made-up point.
type guideLocation struct {
	Known    bool
	Lat, Lng float64
	Area     string // nearest region name, e.g. "Galle"
}

// newGuideLocation interprets a chat request's location. locationKnown is
// nil from app builds older than the flag; they only ever sent a real fix
// or the app's old fallback point, so a non-zero point is trusted as before.
func newGuideLocation(lat, lng float64, locationKnown *bool) guideLocation {
	if (lat == 0 && lng == 0) || (locationKnown != nil && !*locationKnown) {
		return guideLocation{}
	}
	return guideLocation{Known: true, Lat: lat, Lng: lng, Area: geo.NearestRegion(lat, lng).Name}
}

// promptLine is the "Traveller's location" line of the system prompt.
func (l guideLocation) promptLine() string {
	if !l.Known {
		return "unknown — the app couldn't get their location. If they ask for something nearby, ask which town or area they're in."
	}
	return "known, near " + l.Area + ". search_places already searches and sorts around their exact position, so never ask which city or town they're in."
}

// guidePlaceFacts is the slice of a place's Data blob the guide talks about.
type guidePlaceFacts struct {
	CategoryID string  `json:"categoryId"`
	Type       string  `json:"type"`
	Region     string  `json:"region"`
	PriceLevel string  `json:"priceLevel"`
	Rating     float64 `json:"rating"`
	OpenNow    *bool   `json:"openNow"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

func parseGuideFacts(p domain.Place) guidePlaceFacts {
	var f guidePlaceFacts
	_ = json.Unmarshal(p.Data, &f)
	return f
}

// distanceFrom is the km from loc to the place, rounded to 0.1, or nil when
// either point is unknown.
func (f guidePlaceFacts) distanceFrom(loc guideLocation) *float64 {
	if !loc.Known || (f.Lat == 0 && f.Lng == 0) {
		return nil
	}
	km := math.Round(geo.HaversineKm(loc.Lat, loc.Lng, f.Lat, f.Lng)*10) / 10
	return &km
}

func (f guidePlaceFacts) category() string {
	if f.Type != "" {
		return f.Type
	}
	return f.CategoryID
}

// guideSearchPlace is one search_places result as the model sees it: real
// facts to talk about, so it never has to guess distance, price or rating.
type guideSearchPlace struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Category   string   `json:"category,omitempty"`
	Region     string   `json:"region,omitempty"`
	PriceLevel string   `json:"priceLevel,omitempty"`
	Rating     float64  `json:"rating,omitempty"`
	DistanceKm *float64 `json:"distanceKm,omitempty"`
	OpenNow    *bool    `json:"openNow,omitempty"`
	Reasons    []string `json:"reasons"`
}

func guideSearchResults(ranked []recommend.Scored, loc guideLocation) []guideSearchPlace {
	out := make([]guideSearchPlace, 0, len(ranked))
	for _, sc := range ranked {
		f := parseGuideFacts(sc.Place)
		out = append(out, guideSearchPlace{
			ID: sc.Place.ID, Name: sc.Place.Name, Category: f.category(), Region: f.Region,
			PriceLevel: f.PriceLevel, Rating: f.Rating, DistanceKm: f.distanceFrom(loc),
			OpenNow: f.OpenNow, Reasons: sc.Reasons,
		})
	}
	return out
}

// guidePlaceCard is a place card under a guide reply: enough to show a
// photo, "0.8 km · $$ · ★ 4.6", and a directions link without another
// request. Media URLs are signed like every other place response.
type guidePlaceCard struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Category   string   `json:"category,omitempty"`
	Region     string   `json:"region,omitempty"`
	PriceLevel string   `json:"priceLevel,omitempty"`
	Rating     float64  `json:"rating,omitempty"`
	DistanceKm *float64 `json:"distanceKm,omitempty"`
	Lat        float64  `json:"lat,omitempty"`
	Lng        float64  `json:"lng,omitempty"`
	CoverThumb string   `json:"coverThumb,omitempty"`
	CoverMicro string   `json:"coverMicro,omitempty"`
}

// guidePlaceCardFor builds a card; placeData is the place's signed
// placeJSON map (for the media URLs).
func guidePlaceCardFor(p domain.Place, placeData map[string]any, loc guideLocation) guidePlaceCard {
	f := parseGuideFacts(p)
	str := func(k string) string { s, _ := placeData[k].(string); return s }
	thumb := str("coverThumb")
	if thumb == "" {
		thumb = str("cover")
	}
	return guidePlaceCard{
		ID: p.ID, Name: p.Name, Category: f.category(), Region: f.Region,
		PriceLevel: f.PriceLevel, Rating: f.Rating, DistanceKm: f.distanceFrom(loc),
		Lat: f.Lat, Lng: f.Lng, CoverThumb: thumb, CoverMicro: str("coverMicro"),
	}
}

// guideCardOrder picks which surfaced places get cards and in what order:
// the ones the reply mentions by name, in the order it mentions them. If it
// names none (e.g. a reply that only asks a follow-up), the first few the
// tools returned. ids are deduped; names maps id to place name.
func guideCardOrder(reply string, ids []string, names map[string]string) []string {
	ids = dedupe(ids)
	lower := strings.ToLower(reply)
	type hit struct {
		id  string
		pos int
	}
	var hits []hit
	for _, id := range ids {
		name := strings.ToLower(strings.TrimSpace(names[id]))
		if name == "" {
			continue
		}
		if pos := strings.Index(lower, name); pos >= 0 {
			hits = append(hits, hit{id, pos})
		}
	}
	if len(hits) == 0 {
		if len(ids) > 3 {
			ids = ids[:3]
		}
		return ids
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].pos < hits[j].pos })
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, h.id)
	}
	if len(out) > guideMaxPlaceCards {
		out = out[:guideMaxPlaceCards]
	}
	return out
}
