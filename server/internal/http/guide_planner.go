package httpapi

// Local Guide trip planner. A trip request runs in three steps, each a tool
// the model calls instead of writing a long text answer:
//
//  1. ask_trip_questions — the app shows a card with tappable answers
//     (dates/nights, hotel, area, interests…) before anything is planned.
//  2. find_places + propose_trip_plan — real places from Google Places
//     anywhere in the world, arranged into days; the app shows a visual
//     plan (photos, ratings, map, directions).
//  3. add_to_calendar — "please add this to calendar": the app adds the
//     whole plan, or only the places named, to its Trip calendar.
//
// The three "card" tools end the turn: their arguments become the reply's
// payload (stored in guide_messages.payload) and their short message is the
// reply text. Places in a plan are referenced by "g:<geo place id>" or
// "p:<catalog place id>" and must exist, so the model can't invent one.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/googleplaces"
)

// guideMaxFindPlaces caps Google Places searches per reply (each is billed).
const guideMaxFindPlaces = 6

// Payload types.
const (
	guidePayloadQuestions = "questions"
	guidePayloadPlan      = "plan"
	guidePayloadCalendar  = "calendar"
)

// guidePayload is the structured card shown with an assistant turn.
type guidePayload struct {
	Type      string          `json:"type"`
	Questions *guideQuestions `json:"questions,omitempty"`
	Plan      *guidePlan      `json:"plan,omitempty"`
	Calendar  *guideCalendar  `json:"calendar,omitempty"`
}

type guideQuestions struct {
	Questions []guideQuestion `json:"questions"`
}

// guideQuestion kinds: single (pick one chip), multi (pick several), text
// (type an answer; options are quick picks), dates (date-range picker).
type guideQuestion struct {
	ID      string   `json:"id"`
	Text    string   `json:"text"`
	Kind    string   `json:"kind"`
	Options []string `json:"options,omitempty"`
}

// guidePlanPlace is everything the app needs to draw a stop without another
// request. Exactly one of PlaceID (Spotto catalog) or GeoPlaceID is set.
type guidePlanPlace struct {
	Ref         string   `json:"ref"`
	PlaceID     string   `json:"placeId,omitempty"`
	GeoPlaceID  string   `json:"geoPlaceId,omitempty"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind,omitempty"`
	Address     string   `json:"address,omitempty"`
	Rating      *float64 `json:"rating,omitempty"`
	RatingCount int      `json:"ratingCount,omitempty"`
	Lat         float64  `json:"lat"`
	Lng         float64  `json:"lng"`
	PhotoURL    string   `json:"photoUrl,omitempty"`
	MapsURL     string   `json:"mapsUrl,omitempty"`
}

type guidePlanStop struct {
	Time        string         `json:"time,omitempty"` // "HH:mm"
	DurationMin int            `json:"durationMin,omitempty"`
	Note        string         `json:"note,omitempty"`
	Place       guidePlanPlace `json:"place"`
}

type guidePlanDay struct {
	Title string          `json:"title,omitempty"`
	Date  string          `json:"date,omitempty"` // YYYY-MM-DD
	Stops []guidePlanStop `json:"stops"`
}

type guidePlan struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Area      string         `json:"area,omitempty"`
	StartDate string         `json:"startDate,omitempty"` // YYYY-MM-DD
	Nights    int            `json:"nights,omitempty"`
	Stay      string         `json:"stay,omitempty"`
	Days      []guidePlanDay `json:"days"`
	Tips      []string       `json:"tips,omitempty"`
}

// guideCalendar asks the app to add stops to its Trip calendar. Mode is
// "new" (a new trip), "current" (the active trip) or "ask" (the app asks).
type guideCalendar struct {
	PlanID    string              `json:"planId,omitempty"`
	Mode      string              `json:"mode"`
	Title     string              `json:"title"`
	StartDate string              `json:"startDate,omitempty"`
	DayCount  int                 `json:"dayCount"`
	Stops     []guideCalendarStop `json:"stops"`
}

type guideCalendarStop struct {
	DayIndex int            `json:"dayIndex"`
	Time     string         `json:"time,omitempty"`
	Place    guidePlanPlace `json:"place"`
}

// guideTurn is the state of one reply: plans from earlier turns (for
// add_to_calendar), the Places search budget, and the card, once a card
// tool has been called.
type guideTurn struct {
	plans     map[string]guidePlan
	findCalls int
	payload   *guidePayload
	message   string
}

func newGuideTurn(history []domain.GuideMessage) *guideTurn {
	t := &guideTurn{plans: map[string]guidePlan{}}
	for _, m := range history {
		var p guidePayload
		if len(m.Payload) > 0 && json.Unmarshal(m.Payload, &p) == nil && p.Plan != nil {
			t.plans[p.Plan.ID] = *p.Plan
		}
	}
	return t
}

// latestPlanID is the most recent plan, for "add this to calendar" when the
// model doesn't pass an id.
func latestPlanID(history []domain.GuideMessage) string {
	for i := len(history) - 1; i >= 0; i-- {
		var p guidePayload
		if len(history[i].Payload) > 0 && json.Unmarshal(history[i].Payload, &p) == nil && p.Plan != nil {
			return p.Plan.ID
		}
	}
	return ""
}

func plannerTools() []gemini.Tool {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	return []gemini.Tool{
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "ask_trip_questions",
				Description: "Show the traveller a card of short questions with tappable answers before planning a trip. " +
					"Use it first whenever they ask you to plan or create a trip and haven't given these details yet. " +
					"Skip anything they already told you.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": str("One short friendly sentence shown above the questions."),
						"questions": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"id":   str("Short id, e.g. dates, hotel, area, interests, pace, budget."),
									"text": str("The question, short."),
									"kind": map[string]any{"type": "string", "enum": []string{"single", "multi", "text", "dates"},
										"description": "single = pick one option, multi = pick several, text = typed answer (options are quick picks), dates = travel dates."},
									"options": map[string]any{"type": "array", "items": map[string]any{"type": "string"},
										"description": "2-6 short answer chips."},
								},
								"required": []string{"id", "text", "kind"},
							},
						},
					},
					"required": []string{"message", "questions"},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "find_places",
				Description: "Search Google Places for real places anywhere in the world, e.g. \"top sights in Sigiriya, Sri Lanka\", " +
					"\"breakfast cafes near Hotel Sigiriya\", \"sunset viewpoints near Dambulla\". Returns refs to use in propose_trip_plan. " +
					"Use specific queries that include the town and country. At most " + fmt.Sprint(guideMaxFindPlaces) + " searches per reply.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"query": str("Free-text search including the place/area name."),
						"limit": map[string]any{"type": "number", "description": "How many results, 3-8 (default 6)."},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "propose_trip_plan",
				Description: "Show the traveller a visual day-by-day trip plan. Every stop must use a ref returned by find_places " +
					"or search_places. Order each day sensibly by location and opening times.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message":   str("One or two short sentences shown above the plan. No list of places — the card shows them."),
						"title":     str("e.g. \"2 nights in Sigiriya\""),
						"area":      str("Main town/area."),
						"startDate": str("YYYY-MM-DD if the traveller gave dates, else omit."),
						"nights":    map[string]any{"type": "number"},
						"stay":      str("Their hotel, if they said."),
						"days": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"title": str("Theme of the day, e.g. \"Rock fortress & village\""),
									"stops": map[string]any{
										"type": "array",
										"items": map[string]any{
											"type": "object",
											"properties": map[string]any{
												"ref":         str("A ref from find_places/search_places."),
												"time":        str("Suggested start, HH:mm (24h)."),
												"durationMin": map[string]any{"type": "number"},
												"note":        str("One or two sentences: why go and a practical tip."),
											},
											"required": []string{"ref", "time", "note"},
										},
									},
								},
								"required": []string{"stops"},
							},
						},
						"tips": map[string]any{"type": "array", "items": map[string]any{"type": "string"},
							"description": "Up to 3 short practical tips (tickets, dress code, best time)."},
					},
					"required": []string{"message", "title", "days"},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "add_to_calendar",
				Description: "Add a trip plan, or some of its places, to the traveller's Spotto Trip calendar when they ask " +
					"(\"add this to my calendar\", \"put Pidurangala on day 2\"). Use mode \"ask\" unless they said whether it's a new trip " +
					"or their current trip.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": str("One short sentence, e.g. \"Here you go — where should I put it?\""),
						"planId":  str("The plan's id (shown as planId in the conversation). Omit for the latest plan."),
						"mode":    map[string]any{"type": "string", "enum": []string{"ask", "new", "current"}},
						"stops": map[string]any{
							"type":        "array",
							"description": "Only when adding some places, not the whole plan.",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"ref":      str("A place ref from the plan or from find_places."),
									"dayIndex": map[string]any{"type": "number", "description": "0-based day of the trip."},
									"time":     str("HH:mm"),
								},
								"required": []string{"ref"},
							},
						},
					},
					"required": []string{"message", "mode"},
				},
			},
		},
	}
}

// runPlannerTool handles the planner's tools. handled is false for any
// other tool. The card tools set turn.payload, which ends the reply.
func (s *Server) runPlannerTool(ctx context.Context, tc gemini.ToolCall, turn *guideTurn, history []domain.GuideMessage) (result string, handled bool) {
	switch tc.Function.Name {
	case "ask_trip_questions":
		var args struct {
			Message   string          `json:"message"`
			Questions []guideQuestion `json:"questions"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		qs := cleanQuestions(args.Questions)
		if len(qs) == 0 {
			return `{"error":"no usable questions"}`, true
		}
		turn.payload = &guidePayload{Type: guidePayloadQuestions, Questions: &guideQuestions{Questions: qs}}
		turn.message = cardMessage(args.Message, "A few quick questions so I can plan this properly:")
		return `{"status":"shown"}`, true

	case "find_places":
		var args struct {
			Query string  `json:"query"`
			Limit float64 `json:"limit"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		return s.findPlacesTool(ctx, turn, strings.TrimSpace(args.Query), int(args.Limit)), true

	case "propose_trip_plan":
		var args struct {
			Message   string   `json:"message"`
			Title     string   `json:"title"`
			Area      string   `json:"area"`
			StartDate string   `json:"startDate"`
			Nights    float64  `json:"nights"`
			Stay      string   `json:"stay"`
			Tips      []string `json:"tips"`
			Days      []struct {
				Title string `json:"title"`
				Stops []struct {
					Ref         string  `json:"ref"`
					Time        string  `json:"time"`
					DurationMin float64 `json:"durationMin"`
					Note        string  `json:"note"`
				} `json:"stops"`
			} `json:"days"`
		}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			return `{"error":"arguments were not valid JSON"}`, true
		}
		plan := guidePlan{
			ID: newID("gplan"), Title: strings.TrimSpace(args.Title), Area: strings.TrimSpace(args.Area),
			StartDate: validDate(args.StartDate), Nights: int(args.Nights), Stay: strings.TrimSpace(args.Stay),
			Tips: limitStrings(args.Tips, 3),
		}
		var dropped []string
		resolved := map[string]guidePlanPlace{}
		for i, d := range args.Days {
			day := guidePlanDay{Title: strings.TrimSpace(d.Title)}
			if plan.StartDate != "" {
				day.Date = addDays(plan.StartDate, i)
			}
			for _, st := range d.Stops {
				p, ok := resolved[st.Ref]
				if !ok {
					var err error
					if p, err = s.resolvePlanRef(ctx, st.Ref); err != nil {
						dropped = append(dropped, st.Ref)
						continue
					}
					resolved[st.Ref] = p
				}
				day.Stops = append(day.Stops, guidePlanStop{
					Time: validClock(st.Time), DurationMin: int(st.DurationMin), Note: strings.TrimSpace(st.Note), Place: p,
				})
			}
			if len(day.Stops) > 0 {
				plan.Days = append(plan.Days, day)
			}
		}
		if len(plan.Days) == 0 {
			return `{"error":"none of the refs were real places — call find_places and use the refs it returns"}`, true
		}
		if plan.Title == "" {
			plan.Title = "Your trip"
		}
		if len(dropped) > 0 {
			log.Printf("guide: plan dropped unknown refs %v", dropped)
		}
		turn.payload = &guidePayload{Type: guidePayloadPlan, Plan: &plan}
		turn.message = cardMessage(args.Message, "Here's a plan — tap any place for details.")
		return `{"status":"shown","planId":"` + plan.ID + `"}`, true

	case "add_to_calendar":
		var args struct {
			Message string `json:"message"`
			PlanID  string `json:"planId"`
			Mode    string `json:"mode"`
			Stops   []struct {
				Ref      string   `json:"ref"`
				DayIndex *float64 `json:"dayIndex"`
				Time     string   `json:"time"`
			} `json:"stops"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		planID := strings.TrimSpace(args.PlanID)
		if _, ok := turn.plans[planID]; !ok {
			planID = latestPlanID(history)
		}
		plan, hasPlan := turn.plans[planID]
		mode := args.Mode
		if mode != "new" && mode != "current" {
			mode = "ask"
		}
		cal := guideCalendar{Mode: mode}
		if hasPlan {
			cal.PlanID, cal.Title, cal.StartDate = plan.ID, plan.Title, plan.StartDate
			cal.DayCount = max(len(plan.Days), plan.Nights+1)
		}
		if len(args.Stops) == 0 {
			if !hasPlan {
				return `{"error":"there is no trip plan yet — plan one first, or name the places to add"}`, true
			}
			cal.Stops = planCalendarStops(plan)
		} else {
			for _, st := range args.Stops {
				p, err := s.resolvePlanRef(ctx, st.Ref)
				if err != nil {
					continue
				}
				day, at := 0, validClock(st.Time)
				if st.DayIndex != nil {
					day = max(0, int(*st.DayIndex))
				} else if i, t, ok := findInPlan(plan, p.Ref); hasPlan && ok {
					day = i
					if at == "" {
						at = t
					}
				}
				cal.Stops = append(cal.Stops, guideCalendarStop{DayIndex: day, Time: at, Place: p})
				cal.DayCount = max(cal.DayCount, day+1)
			}
			if len(cal.Stops) == 0 {
				return `{"error":"none of those refs are real places"}`, true
			}
			if cal.Title == "" {
				cal.Title = "My trip"
			}
		}
		turn.payload = &guidePayload{Type: guidePayloadCalendar, Calendar: &cal}
		turn.message = cardMessage(args.Message, "Ready to add to your Trip calendar:")
		return `{"status":"shown"}`, true
	}
	return "", false
}

// findPlacesTool searches Google, caches the results as geo places (linked
// to the Spotto catalog when it's the same place) and lists them for the
// model with refs.
func (s *Server) findPlacesTool(ctx context.Context, turn *guideTurn, query string, limit int) string {
	if s.Places == nil {
		return `{"error":"place search is not configured"}`
	}
	if query == "" {
		return `{"error":"query is required"}`
	}
	if turn.findCalls >= guideMaxFindPlaces {
		return `{"error":"search limit reached for this reply — plan with the places you already found"}`
	}
	turn.findCalls++
	if limit < 3 || limit > 8 {
		limit = 6
	}
	found, err := s.Places.SearchTextMany(ctx, query, limit)
	if errors.Is(err, googleplaces.ErrNoResult) {
		return `{"places":[]}`
	}
	if err != nil {
		log.Printf("guide: find_places %q: %v", query, err)
		return `{"error":"place search failed"}`
	}
	type result struct {
		Ref         string   `json:"ref"`
		Name        string   `json:"name"`
		Kind        string   `json:"kind,omitempty"`
		Address     string   `json:"address,omitempty"`
		Rating      *float64 `json:"rating,omitempty"`
		RatingCount int      `json:"ratingCount,omitempty"`
		Summary     string   `json:"summary,omitempty"`
		Lat         float64  `json:"lat"`
		Lng         float64  `json:"lng"`
	}
	out := make([]result, 0, len(found))
	for _, p := range found {
		geoID, err := s.Store.UpsertGeoPlace(ctx, domain.GeoPlace{
			ID: newID("gp"), GooglePlaceID: p.ID, Name: p.Name, Lat: p.Lat, Lng: p.Lng,
			Address: p.Address, City: p.City, CountryCode: p.CountryCode, Kind: googleplaces.Kind(p.PrimaryType, p.Types),
			Rating: p.Rating, RatingCount: p.RatingCount, PhotoName: p.PhotoName, MapsURL: p.MapsURL, Types: p.Types,
		})
		if err != nil {
			log.Printf("guide: save geo place: %v", err)
			continue
		}
		ref := "g:" + geoID
		if catalogID := s.matchCatalogPlace(ctx, p.Name, p.Lat, p.Lng); catalogID != "" {
			ref = "p:" + catalogID
		}
		out = append(out, result{
			Ref: ref, Name: p.Name, Kind: googleplaces.Kind(p.PrimaryType, p.Types), Address: p.Address,
			Rating: p.Rating, RatingCount: p.RatingCount, Summary: p.Summary, Lat: p.Lat, Lng: p.Lng,
		})
	}
	b, _ := json.Marshal(map[string]any{"places": out})
	return string(b)
}

var errUnknownRef = errors.New("guide: unknown place ref")

// resolvePlanRef turns a "g:"/"p:" ref (a bare catalog id is accepted too,
// since search_places returns plain ids) into a stop's place data.
func (s *Server) resolvePlanRef(ctx context.Context, ref string) (guidePlanPlace, error) {
	ref = strings.TrimSpace(ref)
	switch {
	case strings.HasPrefix(ref, "g:"):
		gp, err := s.Store.GeoPlaceByID(ctx, strings.TrimPrefix(ref, "g:"))
		if err != nil {
			return guidePlanPlace{}, errUnknownRef
		}
		return planPlaceFromGeo(gp), nil
	case ref == "":
		return guidePlanPlace{}, errUnknownRef
	default:
		p, err := s.Store.PlaceByID(ctx, strings.TrimPrefix(ref, "p:"))
		if err != nil {
			return guidePlanPlace{}, errUnknownRef
		}
		card := guidePlaceCardFor(p, s.placeJSON(ctx, p), guideLocation{})
		var rating *float64
		if card.Rating > 0 {
			rating = &card.Rating
		}
		return guidePlanPlace{
			Ref: "p:" + p.ID, PlaceID: p.ID, Name: p.Name, Kind: card.Category, Address: card.Region,
			Rating: rating, Lat: card.Lat, Lng: card.Lng, PhotoURL: card.CoverThumb,
			MapsURL: mapsSearchURL(p.Name, card.Lat, card.Lng),
		}, nil
	}
}

func planPlaceFromGeo(gp domain.GeoPlace) guidePlanPlace {
	p := guidePlanPlace{
		Ref: "g:" + gp.ID, GeoPlaceID: gp.ID, Name: gp.Name, Kind: gp.Kind, Address: gp.Address,
		Rating: gp.Rating, RatingCount: gp.RatingCount, Lat: gp.Lat, Lng: gp.Lng, MapsURL: gp.MapsURL,
	}
	if gp.HasPhoto {
		p.PhotoURL = "/geo-places/" + gp.ID + "/photo"
	}
	if p.MapsURL == "" {
		p.MapsURL = mapsSearchURL(gp.Name, gp.Lat, gp.Lng)
	}
	return p
}

func mapsSearchURL(name string, lat, lng float64) string {
	q := name
	if lat != 0 || lng != 0 {
		q = fmt.Sprintf("%s @%.6f,%.6f", name, lat, lng)
	}
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(q)
}

func planCalendarStops(plan guidePlan) []guideCalendarStop {
	var out []guideCalendarStop
	for i, d := range plan.Days {
		for _, st := range d.Stops {
			out = append(out, guideCalendarStop{DayIndex: i, Time: st.Time, Place: st.Place})
		}
	}
	return out
}

// findInPlan reports which day (and at what time) a place is in a plan.
func findInPlan(plan guidePlan, ref string) (int, string, bool) {
	for i, d := range plan.Days {
		for _, st := range d.Stops {
			if st.Place.Ref == ref {
				return i, st.Time, true
			}
		}
	}
	return 0, "", false
}

// guidePayloadSummary describes a card in one line of text, so the model
// knows what the traveller was shown (including plan ids and refs) when the
// conversation is replayed.
func guidePayloadSummary(raw json.RawMessage) string {
	var p guidePayload
	if len(raw) == 0 || json.Unmarshal(raw, &p) != nil {
		return ""
	}
	switch {
	case p.Questions != nil:
		var qs []string
		for _, q := range p.Questions.Questions {
			qs = append(qs, q.Text)
		}
		return "[Shown a question card: " + strings.Join(qs, " | ") + "]"
	case p.Plan != nil:
		var b strings.Builder
		fmt.Fprintf(&b, "[Shown trip plan planId=%s %q", p.Plan.ID, p.Plan.Title)
		if p.Plan.StartDate != "" {
			fmt.Fprintf(&b, " from %s", p.Plan.StartDate)
		}
		for i, d := range p.Plan.Days {
			fmt.Fprintf(&b, "; Day %d:", i+1)
			for _, st := range d.Stops {
				fmt.Fprintf(&b, " %s %s (%s),", st.Time, st.Place.Name, st.Place.Ref)
			}
		}
		b.WriteString("]")
		return b.String()
	case p.Calendar != nil:
		return fmt.Sprintf("[Offered to add %d place(s) to the Trip calendar (mode %s)]", len(p.Calendar.Stops), p.Calendar.Mode)
	}
	return ""
}

func cleanQuestions(in []guideQuestion) []guideQuestion {
	var out []guideQuestion
	for i, q := range in {
		q.Text = strings.TrimSpace(q.Text)
		if q.Text == "" {
			continue
		}
		switch q.Kind {
		case "single", "multi", "text", "dates":
		default:
			q.Kind = "single"
			if len(q.Options) == 0 {
				q.Kind = "text"
			}
		}
		if q.Kind == "single" || q.Kind == "multi" {
			if len(q.Options) == 0 {
				q.Kind = "text"
			}
		}
		q.Options = limitStrings(q.Options, 6)
		if strings.TrimSpace(q.ID) == "" {
			q.ID = fmt.Sprintf("q%d", i+1)
		}
		out = append(out, q)
		if len(out) == 6 {
			break
		}
	}
	return out
}

func cardMessage(msg, fallback string) string {
	if msg = strings.TrimSpace(msg); msg != "" {
		return msg
	}
	return fallback
}

func limitStrings(in []string, n int) []string {
	var out []string
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
			if len(out) == n {
				break
			}
		}
	}
	return out
}

var (
	dateRe  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	clockRe = regexp.MustCompile(`^([01]?\d|2[0-3]):([0-5]\d)$`)
)

func validDate(s string) string {
	s = strings.TrimSpace(s)
	if !dateRe.MatchString(s) {
		return ""
	}
	return s
}

// validClock normalises "7:30" to "07:30"; anything else becomes "".
func validClock(s string) string {
	m := clockRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return ""
	}
	if len(m[1]) == 1 {
		m[1] = "0" + m[1]
	}
	return m[1] + ":" + m[2]
}

// addDays returns the YYYY-MM-DD date n days after start.
func addDays(start string, n int) string {
	t, err := time.Parse("2006-01-02", start)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, n).Format("2006-01-02")
}
