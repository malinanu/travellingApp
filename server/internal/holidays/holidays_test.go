package holidays

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUpcomingStaticFallback(t *testing.T) {
	got, err := Upcoming(context.Background(), "", 3650)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one static holiday within a 10-year window")
	}
	for _, h := range got {
		if h.Name == "" {
			t.Errorf("holiday %v has empty name", h.Date)
		}
	}
}

func TestUpcomingStaticFallbackExcludesPast(t *testing.T) {
	got, err := Upcoming(context.Background(), "", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := truncateToDay(time.Now())
	for _, h := range got {
		if h.Date.Before(now) {
			t.Errorf("holiday %v is in the past, should have been filtered", h.Date)
		}
		if h.Date.After(now.AddDate(0, 0, 7)) {
			t.Errorf("holiday %v is outside the 7-day window", h.Date)
		}
	}
}

// TestParseCalendarEvents verifies the Google Calendar response-parsing
// logic directly against a sample response shape — this can't be tested
// against the real endpoint without a live API key, but the parsing logic
// itself (date parsing, skipping malformed entries) is fully exercised
// this way, independent of the network call in fetchFromGoogle.
func TestParseCalendarEvents(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"items": []map[string]any{
				{"summary": "Mock Holiday", "start": map[string]string{"date": "2026-12-25"}},
				{"summary": "Skip me (no date)", "start": map[string]string{}},
				{"summary": "Skip me (bad date)", "start": map[string]string{"date": "not-a-date"}},
			},
		})
	}))
	defer mock.Close()

	res, err := http.Get(mock.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var parsed calendarEventsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		t.Fatal(err)
	}

	got := parseCalendarEvents(parsed)
	if len(got) != 1 {
		t.Fatalf("expected 1 valid holiday after filtering malformed entries, got %d: %+v", len(got), got)
	}
	if got[0].Name != "Mock Holiday" {
		t.Errorf("expected name 'Mock Holiday', got %q", got[0].Name)
	}
	want := mustDate("2026-12-25")
	if !got[0].Date.Equal(want) {
		t.Errorf("expected date %v, got %v", want, got[0].Date)
	}
}
