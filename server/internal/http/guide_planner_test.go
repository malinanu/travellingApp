package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spotto/server/internal/domain"
)

func TestCleanQuestions(t *testing.T) {
	got := cleanQuestions([]guideQuestion{
		{ID: "dates", Text: "When are you going?", Kind: "dates"},
		{Text: "  "}, // dropped
		{ID: "pace", Text: "Pace?", Kind: "single"}, // no options: becomes a text question
		{Text: "What do you enjoy?", Kind: "weird", Options: []string{"Hiking", " ", "Food"}},
	})
	if len(got) != 3 {
		t.Fatalf("got %d questions: %+v", len(got), got)
	}
	if got[1].Kind != "text" {
		t.Errorf("single without options kind = %q, want text", got[1].Kind)
	}
	if got[2].Kind != "single" || got[2].ID != "q4" || strings.Join(got[2].Options, ",") != "Hiking,Food" {
		t.Errorf("unknown kind not normalised: %+v", got[2])
	}
}

func TestValidClockAndDates(t *testing.T) {
	for in, want := range map[string]string{"7:30": "07:30", "16:05": "16:05", "24:00": "", "noon": "", " 06:00 ": "06:00"} {
		if got := validClock(in); got != want {
			t.Errorf("validClock(%q) = %q, want %q", in, got, want)
		}
	}
	if validDate("2026-11-12") != "2026-11-12" || validDate("12/11/2026") != "" {
		t.Error("validDate")
	}
	if got := addDays("2026-12-31", 1); got != "2027-01-01" {
		t.Errorf("addDays = %q", got)
	}
}

func samplePlan() guidePlan {
	rock := guidePlanPlace{Ref: "g:gp_1", GeoPlaceID: "gp_1", Name: "Sigiriya Rock"}
	cafe := guidePlanPlace{Ref: "p:cafe1", PlaceID: "cafe1", Name: "Rastha Cafe"}
	pid := guidePlanPlace{Ref: "g:gp_2", GeoPlaceID: "gp_2", Name: "Pidurangala Rock"}
	return guidePlan{
		ID: "gplan_x", Title: "2 nights in Sigiriya", StartDate: "2026-11-12", Nights: 2,
		Days: []guidePlanDay{
			{Title: "Fortress", Stops: []guidePlanStop{{Time: "07:00", Place: rock}, {Time: "12:30", Place: cafe}}},
			{Title: "Sunrise", Stops: []guidePlanStop{{Time: "05:30", Place: pid}}},
		},
	}
}

func TestPlanCalendarStops(t *testing.T) {
	stops := planCalendarStops(samplePlan())
	if len(stops) != 3 || stops[2].DayIndex != 1 || stops[2].Time != "05:30" || stops[0].Place.Name != "Sigiriya Rock" {
		t.Errorf("stops = %+v", stops)
	}
	if day, at, ok := findInPlan(samplePlan(), "g:gp_2"); !ok || day != 1 || at != "05:30" {
		t.Errorf("findInPlan = %d %q %v", day, at, ok)
	}
}

func TestGuidePayloadSummaryAndHistory(t *testing.T) {
	plan := samplePlan()
	raw, _ := json.Marshal(guidePayload{Type: guidePayloadPlan, Plan: &plan})
	sum := guidePayloadSummary(raw)
	for _, want := range []string{"planId=gplan_x", "Day 2:", "05:30 Pidurangala Rock (g:gp_2)"} {
		if !strings.Contains(sum, want) {
			t.Errorf("summary missing %q: %s", want, sum)
		}
	}
	q, _ := json.Marshal(guidePayload{Type: guidePayloadQuestions, Questions: &guideQuestions{
		Questions: []guideQuestion{{ID: "n", Text: "How many nights?", Kind: "single", Options: []string{"1", "2"}}}}})
	if s := guidePayloadSummary(q); !strings.Contains(s, "How many nights?") {
		t.Errorf("questions summary = %q", s)
	}
	if guidePayloadSummary(nil) != "" {
		t.Error("empty payload should have no summary")
	}

	history := []domain.GuideMessage{
		{Role: "user", Content: "plan Sigiriya"},
		{Role: "assistant", Content: "Here you go", Payload: raw},
		{Role: "user", Content: "thanks"},
		{Role: "assistant", Content: "Anything else?"},
	}
	if id := latestPlanID(history); id != "gplan_x" {
		t.Errorf("latestPlanID = %q", id)
	}
	if turn := newGuideTurn(history); turn.plans["gplan_x"].Title != plan.Title {
		t.Errorf("newGuideTurn plans = %+v", turn.plans)
	}
}

func TestPlannerToolsRegistered(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range guideTools() {
		names[tool.Function.Name] = true
	}
	for _, n := range []string{"search_places", "find_places", "ask_trip_questions", "propose_trip_plan", "add_to_calendar"} {
		if !names[n] {
			t.Errorf("tool %s missing", n)
		}
	}
	if names["save_trip"] {
		t.Error("save_trip should be gone")
	}
}
