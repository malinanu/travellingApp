package recommend

import (
	"encoding/json"
	"testing"

	"github.com/spotto/server/internal/domain"
)

func place(id string, data map[string]any) domain.Place {
	b, _ := json.Marshal(data)
	return domain.Place{ID: id, Name: id, Data: b}
}

func ptr(f float64) *float64 { return &f }

func TestRankPrefersMatchingVibe(t *testing.T) {
	quiet := place("quiet-cafe", map[string]any{
		"type": "cafe", "lat": 5.9724, "lng": 80.4292, "openNow": true,
		"vibe": map[string]any{"productive": 5, "loud": 1, "crowded": 1},
	})
	loud := place("loud-bar", map[string]any{
		"type": "cafe", "lat": 5.9724, "lng": 80.4292, "openNow": true,
		"vibe": map[string]any{"productive": 1, "loud": 5, "crowded": 5},
	})

	prefs := Preferences{
		DesiredProductive: ptr(5), DesiredLoud: ptr(1), DesiredCrowded: ptr(1),
		Lat: 5.9724, Lng: 80.4292,
	}
	ranked := Rank([]domain.Place{loud, quiet}, prefs, "sunny")
	if len(ranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ranked))
	}
	if ranked[0].Place.ID != "quiet-cafe" {
		t.Fatalf("expected quiet-cafe to rank first, got %s (scores: %v)", ranked[0].Place.ID, ranked)
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Fatalf("expected quiet-cafe's score to beat loud-bar's: %v", ranked)
	}
}

func TestRankHardFiltersExclude(t *testing.T) {
	closedCafe := place("closed", map[string]any{"type": "cafe", "openNow": false})
	openCafe := place("open", map[string]any{"type": "cafe", "openNow": true})

	ranked := Rank([]domain.Place{closedCafe, openCafe}, Preferences{RequireOpenNow: true}, "sunny")
	if len(ranked) != 1 || ranked[0].Place.ID != "open" {
		t.Fatalf("expected only the open cafe to survive the filter, got %v", ranked)
	}
}

func TestRankCategoryFilter(t *testing.T) {
	cafe := place("cafe1", map[string]any{"type": "cafe"})
	surf := place("surf1", map[string]any{"type": "surf school"})

	ranked := Rank([]domain.Place{cafe, surf}, Preferences{Categories: []string{"surf"}}, "sunny")
	if len(ranked) != 1 || ranked[0].Place.ID != "surf1" {
		t.Fatalf("expected only the surf school to match category filter, got %v", ranked)
	}
}

func TestRankDistanceDecay(t *testing.T) {
	near := place("near", map[string]any{"type": "cafe", "lat": 5.9724, "lng": 80.4292})
	far := place("far", map[string]any{"type": "cafe", "lat": 6.9271, "lng": 79.8612}) // Colombo, ~140km away

	ranked := Rank([]domain.Place{far, near}, Preferences{Lat: 5.9724, Lng: 80.4292}, "sunny")
	if ranked[0].Place.ID != "near" {
		t.Fatalf("expected the nearer place to rank first, got %v", ranked)
	}
}

func TestRankUnknownLocationScoresDistanceNeutral(t *testing.T) {
	near := place("near", map[string]any{"type": "cafe", "lat": 5.9724, "lng": 80.4292})
	far := place("far", map[string]any{"type": "cafe", "lat": 6.9271, "lng": 79.8612}) // Colombo, ~140km away

	// Lat/Lng both zero (unresolved location) — without the fix this scores
	// via HaversineKm from (0,0), a real ~8,700km "Null Island" distance.
	ranked := Rank([]domain.Place{far, near}, Preferences{}, "sunny")
	if len(ranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ranked))
	}
	if ranked[0].Score != ranked[1].Score {
		t.Fatalf("expected equal scores when location is unknown (distance neutralized), got %v", ranked)
	}
}

func TestSemanticScoreNeutralWithoutEmbeddings(t *testing.T) {
	f := facts{}
	score, has := semanticScore(f, Preferences{})
	if has || score != 0.5 {
		t.Fatalf("expected neutral (0.5, false) with no embeddings, got (%v, %v)", score, has)
	}

	f.Embedding = []float64{1, 0, 0}
	score, has = semanticScore(f, Preferences{})
	if has || score != 0.5 {
		t.Fatalf("expected neutral when only the place has an embedding, got (%v, %v)", score, has)
	}
}

func TestRankPrefersSemanticMatch(t *testing.T) {
	// Place embeddings are 3-D unit-ish vectors; "close" points the same
	// direction as the query, "far" points away from it.
	near := place("near", map[string]any{"type": "cafe"})
	near.Embedding = []float64{1, 0, 0}
	far := place("far", map[string]any{"type": "cafe"})
	far.Embedding = []float64{0, 1, 0}

	prefs := Preferences{QueryEmbedding: []float64{1, 0, 0}}
	ranked := Rank([]domain.Place{far, near}, prefs, "sunny")
	if len(ranked) != 2 {
		t.Fatalf("expected 2 results, got %d", len(ranked))
	}
	if ranked[0].Place.ID != "near" {
		t.Fatalf("expected the semantically closer place to rank first, got %v", ranked)
	}
	if ranked[0].Score <= ranked[1].Score {
		t.Fatalf("expected near's score to beat far's: %v", ranked)
	}
}

func TestRankWeatherFitIndoorWhenRainy(t *testing.T) {
	indoor := place("indoor", map[string]any{"type": "cafe", "indoor": true})
	outdoor := place("outdoor", map[string]any{"type": "cafe", "indoor": false})

	ranked := Rank([]domain.Place{outdoor, indoor}, Preferences{}, "rainy")
	if ranked[0].Place.ID != "indoor" {
		t.Fatalf("expected the indoor place to rank first when rainy, got %v", ranked)
	}
}
