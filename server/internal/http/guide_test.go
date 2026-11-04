package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
)

func TestClampVibe(t *testing.T) {
	ptr := func(f float64) *float64 { return &f }
	cases := []struct {
		name string
		in   *float64
		want *float64
	}{
		{"nil stays nil", nil, nil},
		{"in range unchanged", ptr(3), ptr(3)},
		{"above range clamps to 5", ptr(10), ptr(5)},
		{"below range clamps to 1", ptr(-3), ptr(1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := clampVibe(c.in)
			if (got == nil) != (c.want == nil) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			if got != nil && *got != *c.want {
				t.Fatalf("got %v, want %v", *got, *c.want)
			}
		})
	}
}

func TestGuideRateLimiting(t *testing.T) {
	limiter := newRateLimiter(3, 1*time.Minute)
	key := "user_test_123"

	for i := 1; i <= 3; i++ {
		if !limiter.allowInc(key) {
			t.Errorf("request %d should be allowed", i)
		}
	}

	if limiter.allowInc(key) {
		t.Errorf("request 4 should be rate limited")
	}
}

func TestGuideSystemPromptPersonalization(t *testing.T) {
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	loc := guideLocation{Known: true, Lat: 5.9485, Lng: 80.4716, Area: "Mirissa"}
	pData := guidePromptProfile{
		Interests: "Surfing, Street Food, Beaches",
		Bookmarks: "- Coconut Tree Hill (Mirissa, beach)\n- Mirissa Whale Watching (Mirissa, attraction)",
		Reviews:   "- Zephyr Restaurant & Bar (Mirissa) - rated 5/5",
	}

	prompt := guideSystemPrompt(now, loc, "Budget Backpacker, Foodie", "clear", pData)

	if !strings.Contains(prompt, "Budget Backpacker") || !strings.Contains(prompt, "Foodie") {
		t.Errorf("prompt missing travel styles: %s", prompt)
	}
	if !strings.Contains(prompt, "Surfing") || !strings.Contains(prompt, "Street Food") {
		t.Errorf("prompt missing interests: %s", prompt)
	}
	if !strings.Contains(prompt, "Coconut Tree Hill") {
		t.Errorf("prompt missing bookmarked places: %s", prompt)
	}
	if !strings.Contains(prompt, "Zephyr Restaurant & Bar") {
		t.Errorf("prompt missing top rated places: %s", prompt)
	}
	if !strings.Contains(prompt, "NEVER claim that your search tool has a temporary roadblock or glitch") {
		t.Errorf("prompt missing anti-glitch hallucination instruction: %s", prompt)
	}
}

func TestSearchCatalogPlaces_MirissaAndArugamBay(t *testing.T) {
	// Seed global catalog cache
	globalCatalogCache.mu.Lock()
	globalCatalogCache.places = []domain.Place{
		{
			ID:   "p_mirissa_1",
			Name: "Coconut Tree Hill",
			Data: []byte(`{"region":"Mirissa","type":"Attraction","rating":4.8,"lat":5.9431,"lng":80.4608}`),
		},
		{
			ID:   "p_mirissa_2",
			Name: "Secret Beach Mirissa",
			Data: []byte(`{"region":"Mirissa","type":"Beach","rating":4.7,"lat":5.9480,"lng":80.4500}`),
		},
		{
			ID:   "p_arugam_1",
			Name: "Whiskey Point",
			Data: []byte(`{"region":"Arugam Bay","type":"Beach","rating":4.9,"lat":6.8920,"lng":81.8320}`),
		},
		{
			ID:   "p_colombo_1",
			Name: "Galle Face Green",
			Data: []byte(`{"region":"Colombo","type":"Park","rating":4.3,"lat":6.9271,"lng":79.8428}`),
		},
	}
	globalCatalogCache.fetchedAt = time.Now()
	globalCatalogCache.mu.Unlock()

	s := &Server{}
	ctx := context.Background()

	// User is located in Colombo, but queries "I'm planning to go mirissa next week, give me places"
	userLoc := guideLocation{Known: true, Lat: 6.9271, Lng: 79.8428, Area: "Colombo"}
	res := s.searchCatalogPlaces(ctx, "I'm planning to go mirissa next week, give me places", 6, userLoc)
	if len(res) < 2 {
		t.Fatalf("expected at least 2 places for Mirissa query, got %d: %+v", len(res), res)
	}
	if !strings.Contains(res[0].Name, "Coconut Tree Hill") && !strings.Contains(res[0].Name, "Secret Beach") {
		t.Errorf("expected Mirissa spots first, got: %v", res[0].Name)
	}

	// User queries "next month I'm planning to visit arugambay. give me the places to visit"
	resArugam := s.searchCatalogPlaces(ctx, "next month I'm planning to visit arugambay. give me the places to visit", 6, userLoc)
	if len(resArugam) < 1 {
		t.Fatalf("expected at least 1 place for Arugam Bay query, got %d", len(resArugam))
	}
	if resArugam[0].Name != "Whiskey Point" {
		t.Errorf("expected Whiskey Point for arugambay query, got: %v", resArugam[0].Name)
	}

	// Personalization test: Secret Beach is bookmarked by the user -> should rank first with [Saved in your bookmarks]
	turnWithBookmark := &guideTurn{
		bookmarkedIDs: []string{"p_mirissa_2"},
	}
	resPersonalized := s.searchCatalogPlaces(ctx, "places in Mirissa", 6, userLoc, turnWithBookmark)
	if len(resPersonalized) == 0 {
		t.Fatal("expected personalized results")
	}
	if resPersonalized[0].Name != "Secret Beach Mirissa" {
		t.Errorf("expected bookmarked Secret Beach Mirissa to rank first, got: %v", resPersonalized[0].Name)
	}
	if !strings.Contains(resPersonalized[0].Summary, "[Saved in your bookmarks]") {
		t.Errorf("expected summary to indicate bookmark, got: %s", resPersonalized[0].Summary)
	}

	// AvoidIDs exclusion test: when p_mirissa_1 is avoided, search must not return it
	turnWithAvoid := &guideTurn{
		avoidIDs: []string{"p_mirissa_1"},
	}
	resAvoided := s.searchCatalogPlaces(ctx, "places in Mirissa", 6, userLoc, turnWithAvoid)
	for _, p := range resAvoided {
		if p.Ref == "p:p_mirissa_1" || p.Name == "Coconut Tree Hill" {
			t.Errorf("expected p_mirissa_1 to be excluded, but got: %+v", p)
		}
	}
}

func TestIsRequestingAlternativesOrMore(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"give me more place", true},
		{"I don't like these places, give me more place", true},
		{"dont like these spots", true},
		{"give me more options", true},
		{"other places to visit", true},
		{"different spots in Mirissa", true},
		{"can you swap places", true},
		{"show me something else", true},
		{"🔄 Give me more options", true},
		{"🔄 More recommendations", true},
		{"give me other suggestions", true},
		{"any alternatives?", true},
		{"can you replace this stop", true},
		{"dont want these places", true},
		{"show me something different", true},
		{"Where can I get coffee in Galle?", false},
		{"Plan a trip to Mirissa", false},
		{"What is the weather today?", false},
		{"Best cafes nearby", false},
	}

	for _, c := range cases {
		got := isRequestingAlternativesOrMore(c.input)
		if got != c.want {
			t.Errorf("isRequestingAlternativesOrMore(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestGuideSuggestionsForReply(t *testing.T) {
	// 1. Trip plan reply
	planRes := guideChatResult{
		Reply:   "Here is your trip",
		Payload: &guidePayload{Type: guidePayloadPlan, Plan: &guidePlan{ID: "gp_1", Title: "Trip"}},
	}
	planSuggestions := guideSuggestionsForReply(planRes, nil)
	if len(planSuggestions) == 0 || !strings.Contains(planSuggestions[0], "more options") {
		t.Errorf("expected plan suggestions, got: %v", planSuggestions)
	}

	// 2. Place cards reply
	placeRes := guideChatResult{
		Reply:    "Check these out",
		PlaceIDs: []string{"p1", "p2"},
	}
	cards := []guidePlaceCard{{ID: "p1", Name: "Spot 1"}}
	placeSuggestions := guideSuggestionsForReply(placeRes, cards)
	if len(placeSuggestions) == 0 || !strings.Contains(placeSuggestions[1], "Plan a trip") {
		t.Errorf("expected place suggestions, got: %v", placeSuggestions)
	}

	// 3. Advice / Q&A reply
	adviceRes := guideChatResult{
		Reply: "Mirissa is great for surfing from November to April.",
	}
	adviceSuggestions := guideSuggestionsForReply(adviceRes, nil)
	if len(adviceSuggestions) == 0 || !strings.Contains(adviceSuggestions[0], "Plan a trip") {
		t.Errorf("expected advice suggestions, got: %v", adviceSuggestions)
	}
}

