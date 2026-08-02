// Package recommend ranks places against a traveller's stated/inferred
// preferences for the Local Guide chat (see internal/http/guide.go). A
// deterministic, tunable hard-filter-then-weighted-score algorithm — no ML
// training required, fully debuggable, and reusable outside chat later
// (e.g. a "recommended for you" feed) since it operates on the same
// domain.Place data every other feature already uses.
package recommend

import (
	"encoding/json"
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
}

// Scored is one ranked result — Reasons is handed back to the LLM so it
// can explain *why* a place was suggested instead of just naming it.
type Scored struct {
	Place   domain.Place
	Score   float64
	Reasons []string
}

// facts is what this package actually needs out of a Place's opaque JSONB
// Data blob (see domain.Place's own doc comment: Data mirrors the Dart
// Place model) — deliberately narrow, not a full mirror of every field.
type facts struct {
	Type       string   `json:"type"`
	Tags       []string `json:"tags"`
	Styles     []string `json:"styles"`
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
	for _, p := range places {
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
		s, reasons := score(c.facts, c.dist, prefs, weather)
		scored = append(scored, Scored{Place: c.place, Score: s, Reasons: reasons})
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
		if strings.Contains(strings.ToLower(f.Type), want) {
			return true
		}
		for _, tag := range f.Tags {
			if strings.Contains(strings.ToLower(tag), want) {
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

func score(f facts, distKm float64, prefs Preferences, weather string) (float64, []string) {
	var reasons []string

	vibe, hasVibePref := vibeScore(f.Vibe, prefs)
	if hasVibePref {
		reasons = append(reasons, "matches the vibe you're after")
	}

	dist := 0.5
	if distKm != unknownDistance {
		dist = 1 / (1 + distKm/3)
		if distKm < 1 {
			reasons = append(reasons, "very close by")
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
func vibeScore(v domain.VibeAggregate, prefs Preferences) (float64, bool) {
	total, dims := 0.0, 0
	if prefs.DesiredProductive != nil {
		total += 1 - abs(*prefs.DesiredProductive-v.Productive)/4
		dims++
	}
	if prefs.DesiredLoud != nil {
		total += 1 - abs(*prefs.DesiredLoud-v.Loud)/4
		dims++
	}
	if prefs.DesiredCrowded != nil {
		total += 1 - abs(*prefs.DesiredCrowded-v.Crowded)/4
		dims++
	}
	if dims == 0 {
		return 0.5, false
	}
	return total / float64(dims), true
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
