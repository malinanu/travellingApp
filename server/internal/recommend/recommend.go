// Package recommend ranks places against a traveller's stated/inferred
// preferences for the Local Guide chat (see internal/http/guide.go). A
// deterministic, tunable hard-filter-then-weighted-score algorithm — no ML
// training required, fully debuggable, and reusable outside chat later
// (e.g. a "recommended for you" feed) since it operates on the same
// domain.Place data every other feature already uses.
package recommend

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
)

// Preferences captures what the traveller is looking for, either stated
// explicitly in chat or inferred from context (weather, saved travel
// styles). Every field is optional — nil/zero means "no opinion" and that
// factor drops out of scoring rather than penalizing anything.
type Preferences struct {
	// DesiredProductive/Loud/Crowded are 1-5, matching VibeAggregate's
	// scale — nil means the traveller expressed no preference on that
	// dimension.
	DesiredProductive *float64
	DesiredLoud       *float64
	DesiredCrowded    *float64
	// Categories is matched against a place's type/tags — a hard filter
	// when non-empty (e.g. ["cafe"], ["surf"]).
	Categories []string
	// Tags are free-form interests soft-scored against a place's
	// tags/styles — never a hard filter, since a traveller's phrasing
	// rarely matches a place's tags exactly.
	Tags           []string
	MaxDistanceKm  *float64
	MaxPriceLevel  string // "$" / "$$" / "$$$", "" = no ceiling
	RequireIndoor  *bool
	RequireOpenNow bool
	RequireSeaView *bool
	WantsDeals     bool
	Lat, Lng       float64
	// QueryEmbedding is the traveller's free-text search embedded via
	// openrouter.Embed — nil means no free-text query was given, same "no
	// opinion" convention as every other optional field here.
	QueryEmbedding []float64
	// Personalization: traveller's bookmarks and ratings
	BookmarkedIDs []string
	FavoritedIDs  []string
	AvoidIDs      []string
	TargetRegion  string
	QueryText     string
	TravelStyles  []string
	Interests     []string
}

// VibeMatch contains computed vibe compatibility metrics for a place.
type VibeMatch struct {
	Percentage int    `json:"percentage"`
	BadgeLabel string `json:"badgeLabel"`
	Headline   string `json:"headline"`
	Summary    string `json:"summary"`
}

// Scored is one ranked result — Reasons is handed back to the LLM so it
// can explain *why* a place was suggested instead of just naming it.
type Scored struct {
	Place   domain.Place
	Score   float64
	Reasons     []string
	VibeMatch   VibeMatch
	VibeSummary string
}

// facts is what this package actually needs out of a Place's opaque JSONB
// Data blob (see domain.Place's own doc comment: Data mirrors the Dart
// Place model) — deliberately narrow, not a full mirror of every field.
type facts struct {
	Type string `json:"type"`
	// CategoryID is the canonical domain.VendorCategories id (set by
	// syncVendorPlace, see internal/http/vendor.go), distinct from Type
	// which is the display label. Lets matchesCategory do an exact id match
	// instead of relying solely on fragile label substring matching.
	CategoryID string   `json:"categoryId"`
	Tags       []string `json:"tags"`
	Styles     []string `json:"styles"`
	Region     string   `json:"region"`
	Blurb      string   `json:"blurb"`
	PriceLevel string   `json:"priceLevel"`
	OpenNow    bool     `json:"openNow"`
	SeaView    bool     `json:"seaView"`
	Indoor     bool     `json:"indoor"`
	Lat        float64  `json:"lat"`
	Lng        float64  `json:"lng"`
	Rating     float64  `json:"rating"`
	Reviews    int      `json:"reviews"`
	Deal       struct {
		Active bool `json:"active"`
	} `json:"deal"`
	Vibe domain.VibeAggregate `json:"vibe"`
	// Embedding comes from p.Embedding directly, not from Data — it lives
	// in its own DB column (see domain.Place's doc comment).
	Embedding []float64
}

func parseFacts(p domain.Place) facts {
	var f facts
	// Best-effort: a place with malformed/missing Data still gets scored
	// (against zero-value facts) rather than dropped from the app entirely.
	_ = json.Unmarshal(p.Data, &f)
	if f.CategoryID == "" && len(p.Data) > 0 {
		var m map[string]any
		if json.Unmarshal(p.Data, &m) == nil {
			f.CategoryID = domain.CategoryForPlaceData(m)
		}
	}
	if f.Region == "" && (f.Lat != 0 || f.Lng != 0) {
		f.Region = geo.NearestRegion(f.Lat, f.Lng).Name
	}
	f.Embedding = p.Embedding
	return f
}

// SummarizeForEmbedding builds a short descriptive string for a place,
// suitable for embedding (see openrouter.Embed) — reuses parseFacts so this
// doesn't duplicate the Data-blob parsing.
func SummarizeForEmbedding(p domain.Place) string {
	f := parseFacts(p)
	parts := []string{p.Name}
	if f.Type != "" {
		parts = append(parts, f.Type)
	}
	if len(f.Tags) > 0 {
		parts = append(parts, strings.Join(f.Tags, ", "))
	}
	if len(f.Styles) > 0 {
		parts = append(parts, strings.Join(f.Styles, ", "))
	}
	if f.Region != "" {
		parts = append(parts, "in "+f.Region)
	}
	// A trimmed description lets searches like "rooftop seafood" match on
	// what the place actually offers, not just its name and tags. Changing
	// this text doesn't re-embed anything by itself (the worker only redoes
	// places with no embedding for the current model) — use the admin
	// "Re-embed all" when the quota allows.
	if b := strings.TrimSpace(f.Blurb); b != "" {
		if r := []rune(b); len(r) > 240 {
			b = string(r[:240])
		}
		parts = append(parts, b)
	}
	return strings.Join(parts, ". ")
}

// priceRank maps "$"/"$$"/"$$$" to a comparable int; unknown/empty sorts
// as cheapest so a place with no price data set is never wrongly excluded
// by a budget ceiling.
func priceRank(level string) int {
	return len(strings.TrimSpace(level))
}

// Rank filters places to real candidates, scores the rest, and returns
// them sorted best-first with a light per-category diversity cap.
func Rank(places []domain.Place, prefs Preferences, weather string) []Scored {
	type candidate struct {
		place domain.Place
		facts facts
		dist  float64
	}

	candidates := make([]candidate, 0, len(places))
	avoidMap := make(map[string]bool, len(prefs.AvoidIDs))
	for _, id := range prefs.AvoidIDs {
		avoidMap[id] = true
	}

	for _, p := range places {
		if avoidMap[p.ID] {
			continue
		}
		f := parseFacts(p)

		if len(prefs.Categories) > 0 && !matchesCategory(f, prefs.Categories) {
			continue
		}
		if prefs.RequireOpenNow && !f.OpenNow {
			continue
		}
		if prefs.RequireIndoor != nil && f.Indoor != *prefs.RequireIndoor {
			continue
		}
		if prefs.RequireSeaView != nil && f.SeaView != *prefs.RequireSeaView {
			continue
		}
		if prefs.MaxPriceLevel != "" && f.PriceLevel != "" && priceRank(f.PriceLevel) > priceRank(prefs.MaxPriceLevel) {
			continue
		}
		// (0,0) means the traveller's location hasn't resolved yet, not that
		// they're actually at Null Island — treat distance as unknown rather
		// than computing (and filtering on) a meaningless ~8,700km gap.
		dist := unknownDistance
		if prefs.Lat != 0 || prefs.Lng != 0 {
			dist = geo.HaversineKm(prefs.Lat, prefs.Lng, f.Lat, f.Lng)
			if prefs.MaxDistanceKm != nil && dist > *prefs.MaxDistanceKm {
				continue
			}
		}

		candidates = append(candidates, candidate{place: p, facts: f, dist: dist})
	}

	scored := make([]Scored, 0, len(candidates))
	for _, c := range candidates {
		s, reasons := score(c.place.ID, c.facts, c.dist, prefs, weather)
		vm := CalculateVibeMatchFromFacts(c.facts, prefs.TravelStyles, prefs.Interests)
		scored = append(scored, Scored{
			Place:       c.place,
			Score:       s,
			Reasons:     reasons,
			VibeMatch:   vm,
			VibeSummary: vm.Summary,
		})
	}

	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	return capPerCategory(scored, prefs.Categories)
}

func matchesCategory(f facts, categories []string) bool {
	for _, want := range categories {
		want = strings.ToLower(strings.TrimSpace(want))
		if want == "" {
			continue
		}
		// Generic or catch-all categories should never exclude places
		if want == "places" || want == "spots" || want == "sights" || want == "all" || want == "any" ||
			want == "things to do" || want == "things to see" || want == "recommendations" ||
			want == "destinations" || want == "explore" || want == "visit" || want == "must visit" ||
			want == "highlights" || want == "top sights" || want == "places to visit" ||
			want == "places to see" || want == "activities" || want == "attractions" ||
			strings.Contains(want, "place") || strings.Contains(want, "spot") ||
			strings.Contains(want, "sight") || strings.Contains(want, "visit") ||
			strings.Contains(want, "thing") || strings.Contains(want, "destinat") {
			return true
		}

		cat := strings.ToLower(f.CategoryID)
		typ := strings.ToLower(f.Type)

		// Coffee / Cafes
		if strings.Contains(want, "cafe") || strings.Contains(want, "coffee") || strings.Contains(want, "tea") || strings.Contains(want, "bakery") {
			if cat == "coffee" || strings.Contains(typ, "cafe") || strings.Contains(typ, "coffee") {
				return true
			}
			for _, tag := range f.Tags {
				tLower := strings.ToLower(tag)
				if strings.Contains(tLower, "cafe") || strings.Contains(tLower, "coffee") || strings.Contains(tLower, "bakery") {
					return true
				}
			}
		}

		// Food / Restaurants
		if strings.Contains(want, "restaurant") || strings.Contains(want, "food") || strings.Contains(want, "eat") || strings.Contains(want, "dining") {
			if cat == "restaurants" || cat == "takeout" || cat == "delivery" || strings.Contains(typ, "food") || strings.Contains(typ, "restaurant") {
				return true
			}
			for _, tag := range f.Tags {
				tLower := strings.ToLower(tag)
				if strings.Contains(tLower, "restaurant") || strings.Contains(tLower, "food") {
					return true
				}
			}
		}

		// Bars / Nightlife
		if strings.Contains(want, "bar") || strings.Contains(want, "pub") || strings.Contains(want, "nightlife") || strings.Contains(want, "drinks") {
			if cat == "bars" || cat == "nightlife" || strings.Contains(typ, "bar") {
				return true
			}
		}

		// Hotels / Stays
		if strings.Contains(want, "hotel") || strings.Contains(want, "stay") || strings.Contains(want, "resort") || strings.Contains(want, "hostel") || strings.Contains(want, "lodging") {
			if cat == "hotels" || strings.Contains(typ, "lodging") || strings.Contains(typ, "hotel") {
				return true
			}
		}

		// Beach / Coastal / Surf
		if strings.Contains(want, "beach") || strings.Contains(want, "surf") || strings.Contains(want, "coast") {
			if strings.Contains(typ, "beach") || strings.Contains(typ, "surf") {
				return true
			}
			for _, tag := range f.Tags {
				tLower := strings.ToLower(tag)
				if strings.Contains(tLower, "beach") || strings.Contains(tLower, "surf") {
					return true
				}
			}
		}

		// Direct matches & singular/plural variations
		wantTrimmed := strings.TrimSuffix(strings.TrimSuffix(want, "es"), "s")
		catTrimmed := strings.TrimSuffix(strings.TrimSuffix(cat, "es"), "s")
		if cat != "" && (cat == want || catTrimmed == wantTrimmed || strings.Contains(cat, wantTrimmed) || strings.Contains(want, catTrimmed)) {
			return true
		}
		if typ != "" && (typ == want || strings.Contains(typ, wantTrimmed) || strings.Contains(want, typ)) {
			return true
		}
		for _, tag := range f.Tags {
			tLower := strings.ToLower(tag)
			if tLower == want || strings.Contains(tLower, wantTrimmed) || strings.Contains(want, tLower) {
				return true
			}
		}
	}
	return false
}

const (
	wVibe     = 0.30
	wDistance = 0.15
	wRating   = 0.15
	wTags     = 0.10
	wWeather  = 0.10
	wDeal     = 0.05
	wSemantic = 0.15
)

// unknownDistance marks a candidate whose distance couldn't be computed
// because the traveller's location hasn't resolved yet.
const unknownDistance float64 = -1

func score(placeID string, f facts, distKm float64, prefs Preferences, weather string) (float64, []string) {
	var reasons []string

	vibe, hasVibePref := vibeScore(f, prefs)
	if hasVibePref {
		if vibe >= 0.75 {
			addedStyleReason := false
			for _, s := range prefs.TravelStyles {
				switch strings.ToLower(strings.TrimSpace(s)) {
				case "nomad", "digital nomad":
					reasons = append(reasons, "great vibe for nomads (productive & quiet)")
					addedStyleReason = true
				case "surf", "surf traveler":
					reasons = append(reasons, "relaxed surf vibe")
					addedStyleReason = true
				case "luxury":
					reasons = append(reasons, "exclusive & quiet atmosphere")
					addedStyleReason = true
				case "backpacker":
					reasons = append(reasons, "social backpacker atmosphere")
					addedStyleReason = true
				}
				if addedStyleReason {
					break
				}
			}
			if !addedStyleReason {
				for _, in := range prefs.Interests {
					switch strings.ToLower(strings.TrimSpace(in)) {
					case "coworking", "wifi", "fast wi-fi":
						reasons = append(reasons, "productive coworking vibe")
						addedStyleReason = true
					case "quiet", "quiet zone":
						reasons = append(reasons, "calm and quiet vibe")
						addedStyleReason = true
					case "nightlife", "live music":
						reasons = append(reasons, "vibrant social atmosphere")
						addedStyleReason = true
					}
					if addedStyleReason {
						break
					}
				}
			}
			if !addedStyleReason {
				reasons = append(reasons, "great vibe match for your style")
			}
		} else {
			reasons = append(reasons, "matches the vibe you're after")
		}
	}

	dist := 0.5
	if distKm != unknownDistance {
		dist = 1 / (1 + distKm/3)
		if distKm < 1 {
			reasons = append(reasons, "very close by")
		} else if distKm <= 5 {
			reasons = append(reasons, "nearby in the area")
		}
	}

	rating := (f.Rating / 5) * min1(float64(f.Reviews)/15)
	if f.Rating >= 4.3 && f.Reviews >= 10 {
		reasons = append(reasons, "highly rated")
	}

	tags := tagScore(f, prefs.Tags)
	if tags > 0.75 {
		reasons = append(reasons, "matches your interests")
	}

	weatherFit := weatherScore(f, weather)
	if weatherFit >= 1.0 && weather == "rainy" && f.Indoor {
		reasons = append(reasons, "indoors, good for the rain")
	}

	deal := 0.0
	if prefs.WantsDeals && f.Deal.Active {
		deal = 1.0
		reasons = append(reasons, "has an active deal")
	}

	semantic, hasSemantic := semanticScore(f, prefs)
	if hasSemantic && semantic > 0.8 {
		reasons = append(reasons, "matches what you described")
	}

	total := wVibe*vibe + wDistance*dist + wRating*rating + wTags*tags +
		wWeather*weatherFit + wDeal*deal + wSemantic*semantic

	// Personalization bonuses:
	for _, id := range prefs.BookmarkedIDs {
		if id == placeID {
			total += 0.35
			reasons = append(reasons, "saved in your bookmarks")
			break
		}
	}
	for _, id := range prefs.FavoritedIDs {
		if id == placeID {
			total += 0.30
			reasons = append(reasons, "one of your top rated places")
			break
		}
	}

	// Target region boost:
	if prefs.TargetRegion != "" && f.Region != "" {
		if strings.EqualFold(f.Region, prefs.TargetRegion) || strings.Contains(strings.ToLower(f.Region), strings.ToLower(prefs.TargetRegion)) {
			total += 0.25
			reasons = append(reasons, "in "+f.Region)
		}
	}

	// Keyword match boost from free text query:
	if prefs.QueryText != "" {
		q := strings.ToLower(prefs.QueryText)
		if strings.Contains(strings.ToLower(f.Blurb), q) || strings.Contains(strings.ToLower(f.Type), q) {
			total += 0.15
		}
	}

	return total, reasons
}

// semanticScore compares the traveller's free-text query embedding against
// a place's embedding via cosine similarity, remapped from [-1,1] to [0,1]
// so it composes with every other [0,1] scoring factor here. Neutral (0.5,
// false) when either side is missing — same "no opinion" convention as
// vibeScore/tagScore, e.g. before the background worker (see
// internal/http/embedding_worker.go) has backfilled a given place yet.
func semanticScore(f facts, prefs Preferences) (float64, bool) {
	if len(prefs.QueryEmbedding) == 0 || len(f.Embedding) == 0 {
		return 0.5, false
	}
	return (cosineSimilarity(prefs.QueryEmbedding, f.Embedding) + 1) / 2, true
}

func cosineSimilarity(a, b []float64) float64 {
	n := min(len(a), len(b))
	var dot, magA, magB float64
	for i := range n {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// vibeScore averages, per stated dimension, 1 - |desired-actual|/4 (4 =
// max possible gap on the 1-5 scale). A direct per-dimension distance
// rather than cosine similarity: cosine suits sparse/unbounded vectors
// (text embeddings); here there are exactly 3 well-understood, bounded
// dimensions, so a direct distance is more intuitive to tune/explain and
// avoids cosine's instability near zero-magnitude vectors (e.g. "I want it
// quiet" targets loud≈1, a small-magnitude target — exactly where cosine
// similarity gets unreliable).
//
// Each dimension is also gated on domain.RelevantVibeDimensions(f.CategoryID)
// — e.g. "productive" doesn't mean anything for a beach bar, so a
// traveller's productivity preference is excluded from that candidate's
// score entirely rather than compared against a meaningless average.
// InferDesiredVibes maps traveller styles and interests to desired 1-5 vibe ratings.
func InferDesiredVibes(travelStyles, interests []string) (desiredProd, desiredLoud, desiredCrowd *float64) {
	if len(travelStyles) == 0 && len(interests) == 0 {
		return nil, nil, nil
	}
	var prodSum, loudSum, crowdSum float64
	var prodWeight, loudWeight, crowdWeight float64

	add := func(p, l, c, w float64) {
		prodSum += p * w
		loudSum += l * w
		crowdSum += c * w
		prodWeight += w
		loudWeight += w
		crowdWeight += w
	}

	for _, s := range travelStyles {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "nomad", "digital nomad":
			add(4.5, 1.8, 2.5, 1.5)
		case "surf", "surf traveler":
			add(2.0, 3.0, 3.0, 1.0)
		case "luxury":
			add(3.0, 1.5, 1.8, 1.2)
		case "backpacker":
			add(2.5, 3.8, 4.0, 1.2)
		}
	}

	for _, in := range interests {
		switch strings.ToLower(strings.TrimSpace(in)) {
		case "quiet", "quiet zone":
			loudSum += 1.0 * 1.5
			loudWeight += 1.5
		case "coworking":
			prodSum += 5.0 * 1.5
			prodWeight += 1.5
		case "wifi", "fast wi-fi":
			prodSum += 4.5 * 1.2
			prodWeight += 1.2
		case "nightlife", "live music":
			loudSum += 4.5 * 1.5
			crowdSum += 4.5 * 1.5
			loudWeight += 1.5
			crowdWeight += 1.5
		}
	}

	if prodWeight > 0 {
		p := math.Round((prodSum/prodWeight)*10) / 10
		desiredProd = &p
	}
	if loudWeight > 0 {
		l := math.Round((loudSum/loudWeight)*10) / 10
		desiredLoud = &l
	}
	if crowdWeight > 0 {
		c := math.Round((crowdSum/crowdWeight)*10) / 10
		desiredCrowd = &c
	}
	return
}

func categoryVibePriors(categoryID string) (p, l, c float64) {
	switch strings.ToLower(categoryID) {
	case "coffee", "coworking":
		return 4.2, 2.0, 2.5
	case "restaurants":
		return 2.5, 3.0, 3.2
	case "nightlife", "bars":
		return 1.5, 4.2, 4.2
	case "surf", "beaches":
		return 2.0, 2.8, 2.8
	case "hotels", "stays":
		return 3.2, 1.8, 2.0
	default:
		return 2.5, 2.5, 2.5
	}
}

func vibeScore(f facts, prefs Preferences) (float64, bool) {
	relevant := map[string]bool{}
	for _, d := range domain.RelevantVibeDimensions(f.CategoryID) {
		relevant[d] = true
	}
	v := f.Vibe
	unrated := v.Productive == 0 && v.Loud == 0 && v.Crowded == 0 && v.SampleSize <= 0
	if unrated {
		p, l, c := categoryVibePriors(f.CategoryID)
		v.Productive, v.Loud, v.Crowded = p, l, c
	}

	desiredProd := prefs.DesiredProductive
	desiredLoud := prefs.DesiredLoud
	desiredCrowd := prefs.DesiredCrowded

	if desiredProd == nil && desiredLoud == nil && desiredCrowd == nil {
		desiredProd, desiredLoud, desiredCrowd = InferDesiredVibes(prefs.TravelStyles, prefs.Interests)
	}
	total, dims := 0.0, 0
	if desiredProd != nil && relevant["productive"] {
		total += 1 - abs(*desiredProd-v.Productive)/4
		dims++
	}
	if desiredLoud != nil && relevant["loud"] {
		total += 1 - abs(*desiredLoud-v.Loud)/4
		dims++
	}
	if desiredCrowd != nil && relevant["crowded"] {
		total += 1 - abs(*desiredCrowd-v.Crowded)/4
		dims++
	}
	if dims == 0 {
		return 0.5, false
	}
	score := total / float64(dims)
	if unrated {
		score = 0.65 + (score-0.5)*0.5
	}
	return score, true
}

// tagScore is |userTags ∩ (place.tags ∪ place.styles)| / |userTags| — the
// denominator is the traveller's own tag count, so a place matching every
// stated interest scores 1.0 even if it carries other, unrelated tags too.
func tagScore(f facts, userTags []string) float64 {
	if len(userTags) == 0 {
		return 0.5
	}
	placeTags := map[string]bool{}
	for _, t := range f.Tags {
		placeTags[strings.ToLower(t)] = true
	}
	for _, t := range f.Styles {
		placeTags[strings.ToLower(t)] = true
	}
	matched := 0
	for _, t := range userTags {
		if placeTags[strings.ToLower(strings.TrimSpace(t))] {
			matched++
		}
	}
	return float64(matched) / float64(len(userTags))
}

func weatherScore(f facts, weather string) float64 {
	switch weather {
	case "rainy":
		if f.Indoor {
			return 1.0
		}
		return 0.3
	case "sunny":
		if f.SeaView || !f.Indoor {
			return 1.0
		}
		return 0.6
	default:
		return 0.5
	}
}

// capPerCategory keeps at most 2 results per category in the final
// ranking unless the traveller asked for a specific category — a generic
// "what's good nearby" shouldn't come back as 5 near-identical cafés.
func capPerCategory(scored []Scored, categories []string) []Scored {
	if len(categories) > 0 {
		return scored
	}
	const maxPerCategory = 2
	counts := map[string]int{}
	out := make([]Scored, 0, len(scored))
	for _, s := range scored {
		cat := strings.ToLower(parseFacts(s.Place).Type)
		if counts[cat] >= maxPerCategory {
			continue
		}
		counts[cat]++
		out = append(out, s)
	}
	return out
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min1(x float64) float64 {
	if x > 1 {
		return 1
	}
	return x
}

// CalculateVibeMatch computes the vibe compatibility metrics and summary for a place.
func CalculateVibeMatch(p domain.Place, travelStyles, interests []string) VibeMatch {
	return CalculateVibeMatchFromFacts(parseFacts(p), travelStyles, interests)
}

// CalculateVibeMatchFromFacts computes vibe match metrics given parsed facts.
func CalculateVibeMatchFromFacts(f facts, travelStyles, interests []string) VibeMatch {
	primaryStyleLabel := "your vibe"
	for _, s := range travelStyles {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "nomad", "digital nomad":
			primaryStyleLabel = "Digital Nomad"
		case "surf", "surf traveler":
			if primaryStyleLabel == "your vibe" {
				primaryStyleLabel = "Surf Traveler"
			}
		case "luxury":
			if primaryStyleLabel == "your vibe" {
				primaryStyleLabel = "Luxury Traveler"
			}
		case "backpacker":
			if primaryStyleLabel == "your vibe" {
				primaryStyleLabel = "Backpacker"
			}
		}
	}

	desiredProd, desiredLoud, desiredCrowd := InferDesiredVibes(travelStyles, interests)
	v := f.Vibe
	isEstimated := v.SampleSize <= 0
	if isEstimated {
		v.Productive, v.Loud, v.Crowded = categoryVibePriors(f.CategoryID)
	}

	relevant := map[string]bool{}
	for _, d := range domain.RelevantVibeDimensions(f.CategoryID) {
		relevant[d] = true
	}

	totalScore := 0.0
	dims := 0
	if relevant["productive"] {
		dp := 3.0
		if desiredProd != nil {
			dp = *desiredProd
		}
		totalScore += 1.0 - math.Abs(dp-v.Productive)/4.0
		dims++
	}
	if relevant["loud"] {
		dl := 2.5
		if desiredLoud != nil {
			dl = *desiredLoud
		}
		totalScore += 1.0 - math.Abs(dl-v.Loud)/4.0
		dims++
	}
	if relevant["crowded"] {
		dc := 2.5
		if desiredCrowd != nil {
			dc = *desiredCrowd
		}
		totalScore += 1.0 - math.Abs(dc-v.Crowded)/4.0
		dims++
	}

	score := 0.75
	if dims > 0 {
		score = totalScore / float64(dims)
	}
	if isEstimated {
		score = 0.70 + (score-0.5)*0.4
	}

	percent := int(math.Round(score * 100))
	if percent < 10 {
		percent = 10
	}
	if percent > 99 {
		percent = 99
	}

	badgeLabel := "Ideal for " + primaryStyleLabel
	headline := fmt.Sprintf("%d%% Vibe Match", percent)
	if percent < 82 {
		badgeLabel = "Good fit for " + primaryStyleLabel
		headline = fmt.Sprintf("%d%% Good Fit", percent)
	}

	var dimFacts []string
	if relevant["productive"] {
		dimFacts = append(dimFacts, fmt.Sprintf("productive (%.1f/5)", v.Productive))
	}
	if relevant["loud"] {
		if v.Loud <= 2.2 {
			dimFacts = append(dimFacts, fmt.Sprintf("quiet (%.1f/5)", v.Loud))
		} else if v.Loud >= 3.8 {
			dimFacts = append(dimFacts, fmt.Sprintf("lively (%.1f/5)", v.Loud))
		} else {
			dimFacts = append(dimFacts, fmt.Sprintf("noise %.1f/5", v.Loud))
		}
	}
	if relevant["crowded"] {
		if v.Crowded <= 2.2 {
			dimFacts = append(dimFacts, fmt.Sprintf("uncrowded (%.1f/5)", v.Crowded))
		} else if v.Crowded >= 3.8 {
			dimFacts = append(dimFacts, fmt.Sprintf("buzzing (%.1f/5)", v.Crowded))
		} else {
			dimFacts = append(dimFacts, fmt.Sprintf("crowd %.1f/5", v.Crowded))
		}
	}

	summary := fmt.Sprintf("%d%% match for %s", percent, strings.ToLower(primaryStyleLabel))
	if len(dimFacts) > 0 {
		summary += ": " + strings.Join(dimFacts, ", ")
	}

	return VibeMatch{
		Percentage: percent,
		BadgeLabel: badgeLabel,
		Headline:   headline,
		Summary:    summary,
	}
}
