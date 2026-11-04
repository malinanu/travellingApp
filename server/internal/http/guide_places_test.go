package httpapi

import (
	"strings"
	"testing"
	"time"

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
	known := guideSystemPrompt(time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC), guideLocation{Known: true, Lat: 6.03, Lng: 80.21, Area: "Galle"}, "luxury", "sunny")
	if !strings.Contains(known, "near Galle") || !strings.Contains(known, "never ask which city or town") {
		t.Fatalf("known-location prompt missing area or the don't-ask rule:\n%s", known)
	}
	unknown := guideSystemPrompt(time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC), guideLocation{}, "none stated", "unknown")
	if !strings.Contains(known, "Today's date: Friday 23 October 2026") {
		t.Fatalf("prompt missing today's date:\n%s", known)
	}
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
	scVibe := recommend.Scored{
		Place:       p,
		Reasons:     []string{"matches your vibe"},
		VibeSummary: "92% match for digital nomad: productive (4.6/5)",
	}
	outVibe := guideSearchResults([]recommend.Scored{scVibe}, guideLocation{Known: true, Lat: 6.03, Lng: 80.21})
	if outVibe[0].VibeSummary != "92% match for digital nomad: productive (4.6/5)" {
		t.Fatalf("VibeSummary missing or incorrect: %s", outVibe[0].VibeSummary)
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

func TestExtractPlaceDescription(t *testing.T) {
	reply := `Here are a few great spots to relax by the ocean:

- **Doctor's House**: A historic Dutch colonial property converted into a vibrant beach bar and restaurant with woodfired pizzas.
- **Dots Bay House** (0.8 km · $$): Surf lodge right on the sand with live music sessions.
1. Sea Breeze Cafe — Cozy beachfront terrace famous for iced coconut coffee and sunset views.
* **Secret Beach**:
Tucked away cove with crystal clear waters and a rustic beach shack.

Which one sounds best to you?`

	cases := []struct {
		name      string
		placeName string
		wantSub   string
	}{
		{
			name:      "bullet with colon and bold",
			placeName: "Doctor's House",
			wantSub:   "A historic Dutch colonial property",
		},
		{
			name:      "bullet with parentheses meta",
			placeName: "Dots Bay House",
			wantSub:   "Surf lodge right on the sand",
		},
		{
			name:      "numbered item with em-dash",
			placeName: "Sea Breeze Cafe",
			wantSub:   "Cozy beachfront terrace famous for iced coconut coffee",
		},
		{
			name:      "item with description on continuation line",
			placeName: "Secret Beach",
			wantSub:   "Tucked away cove with crystal clear waters",
		},
		{
			name:      "unmentioned place returns empty",
			placeName: "Nonexistent Spot",
			wantSub:   "",
		},
		{
			name:      "curly apostrophe matching",
			placeName: "Doctor's House",
			wantSub:   "A historic Dutch colonial property",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractPlaceDescription(reply, c.placeName)
			if c.wantSub == "" {
				if got != "" {
					t.Fatalf("expected empty description for %q, got %q", c.placeName, got)
				}
			} else {
				if !strings.Contains(got, c.wantSub) {
					t.Fatalf("extractPlaceDescription(%q) = %q, want substring %q", c.placeName, got, c.wantSub)
				}
			}
		})
	}

	// Test curly apostrophe in reply when place has straight apostrophe
	curlyReply := `Here are some recommendations:
- **Doctor’s House**: Vibrant oceanfront venue famous for lively music and seafood.
Hope you enjoy it!`
	gotCurly := extractPlaceDescription(curlyReply, "Doctor's House")
	if !strings.Contains(gotCurly, "Vibrant oceanfront venue") {
		t.Fatalf("expected curly quote to match straight quote, got %q", gotCurly)
	}
	if strings.Contains(gotCurly, "Hope you enjoy") {
		t.Fatalf("outro text was incorrectly appended to place description, got %q", gotCurly)
	}

	// Test markdown link in bullet
	linkReply := `Here are recommendations:
- [Doctor's House](https://spotto.app/p/123): Rustic beach garden bar.
Have fun!`
	gotLink := extractPlaceDescription(linkReply, "Doctor's House")
	if !strings.Contains(gotLink, "Rustic beach garden bar") {
		t.Fatalf("expected markdown link to be stripped, got %q", gotLink)
	}
	if strings.Contains(gotLink, "Have fun") {
		t.Fatalf("outro text was incorrectly appended to place description, got %q", gotLink)
	}
}

func TestGuidePlaceCardVibeBadge(t *testing.T) {
	p := domain.Place{
		ID:   "p_nomad",
		Name: "Nomad Cafe",
		Data: []byte(`{"categoryId":"coffee","type":"cafe","rating":4.8,"vibe":{"productive":4.8,"loud":1.5,"crowded":2.0,"sampleSize":10}}`),
	}
	card := guidePlaceCardFor(p, map[string]any{"coverThumb": "thumb.jpg"}, guideLocation{}, guideUserVibePrefs{
		TravelStyles: []string{"nomad"},
		Interests:    []string{"quiet"},
	})
	if card.VibeMatchPercent < 80 {
		t.Errorf("expected high vibe match percent, got %d", card.VibeMatchPercent)
	}
	if !strings.Contains(card.VibeBadge, "Digital Nomad") {
		t.Errorf("expected badge to mention Digital Nomad, got %s", card.VibeBadge)
	}

	// User without travel styles or interests configured must not receive a vibe badge
	cardNoPrefs := guidePlaceCardFor(p, map[string]any{"coverThumb": "thumb.jpg"}, guideLocation{}, guideUserVibePrefs{})
	if cardNoPrefs.VibeMatchPercent != 0 || cardNoPrefs.VibeBadge != "" {
		t.Errorf("expected no vibe badge when user has no travel styles set, got %d %q", cardNoPrefs.VibeMatchPercent, cardNoPrefs.VibeBadge)
	}
}
