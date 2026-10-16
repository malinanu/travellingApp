package httpapi

import (
	"strings"
	"testing"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/recommend"
)

func TestNewGuideLocation(t *testing.T) {
	yes, no := true, false
	weligama := newGuideLocation(5.9724, 80.4292, &yes)
	if !weligama.Known || weligama.Area == "" {
		t.Fatalf("real fix should be known with an area, got %+v", weligama)
	}
	if l := newGuideLocation(5.9724, 80.4292, &no); l.Known || l.Lat != 0 {
		t.Fatalf("locationKnown=false must drop the point, got %+v", l)
	}
	if l := newGuideLocation(0, 0, nil); l.Known {
		t.Fatalf("0,0 must be unknown, got %+v", l)
	}
	if l := newGuideLocation(6.9, 79.85, nil); !l.Known {
		t.Fatalf("older apps without the flag keep their point, got %+v", l)
	}
}

func TestGuideSystemPromptLocation(t *testing.T) {
	known := guideSystemPrompt(guideLocation{Known: true, Lat: 6.03, Lng: 80.21, Area: "Galle"}, "luxury", "sunny")
	if !strings.Contains(known, "near Galle") || !strings.Contains(known, "never ask which city or town") {
		t.Fatalf("known-location prompt missing area or the don't-ask rule:\n%s", known)
	}
	unknown := guideSystemPrompt(guideLocation{}, "none stated", "unknown")
	if !strings.Contains(unknown, "ask which town or area") {
		t.Fatalf("unknown-location prompt should allow asking for a town:\n%s", unknown)
	}
}

func TestGuideSearchResultsCarryFacts(t *testing.T) {
	p := domain.Place{ID: "plc_1", Name: "Sea Breeze", Data: []byte(`{"type":"Restaurant","region":"Galle","priceLevel":"$$","rating":4.6,"lat":6.031,"lng":80.217}`)}
	out := guideSearchResults([]recommend.Scored{{Place: p, Reasons: []string{"very close by"}}},
		guideLocation{Known: true, Lat: 6.03, Lng: 80.21})
	if len(out) != 1 {
		t.Fatal("want one result")
	}
	r := out[0]
	if r.Category != "Restaurant" || r.PriceLevel != "$$" || r.Rating != 4.6 || r.Region != "Galle" {
		t.Fatalf("facts missing: %+v", r)
	}
	if r.DistanceKm == nil || *r.DistanceKm <= 0 || *r.DistanceKm > 2 {
		t.Fatalf("distance = %v, want under 2 km", r.DistanceKm)
	}
	if noLoc := guideSearchResults([]recommend.Scored{{Place: p}}, guideLocation{}); noLoc[0].DistanceKm != nil {
		t.Fatal("no distance without a known location")
	}
}

func TestGuideCardOrder(t *testing.T) {
	names := map[string]string{"a": "Sea Breeze", "b": "Hill Cafe", "c": "Rice Bowl", "d": "Fourth"}
	got := guideCardOrder("Try Hill Cafe first, then Sea Breeze for dinner.", []string{"a", "b", "c", "a"}, names)
	if strings.Join(got, ",") != "b,a" {
		t.Fatalf("mentioned order = %v, want b,a", got)
	}
	got = guideCardOrder("What budget are you thinking?", []string{"a", "b", "c", "d"}, names)
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("no mentions should fall back to the first three, got %v", got)
	}
}
