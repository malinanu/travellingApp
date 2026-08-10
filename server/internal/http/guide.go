package httpapi

// Local Guide: a conversational assistant that recommends real places and
// (eventually, in a fast-follow) plans trips. Uses OpenRouter's free-tier
// google/gemma-4-31b-it:free model (internal/openrouter) with tool calling
// â€” the model decides for itself whether it has enough information to
// search, or should ask a clarifying question first, steered entirely by
// guideSystemPrompt rather than separate orchestration code. The actual
// place-matching is internal/recommend's deterministic scoring algorithm;
// the LLM never invents a place, it only describes/explains ones
// search_places actually returned.
//
// Conversations persist (guide_conversations/guide_messages) so a
// traveller can close the app and resume later, but only user-visible
// turns are stored â€” each new message re-derives whatever tool calls it
// needs from the prior plain-text turns, not from replaying tool state.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/recommend"
)

// guideMaxToolRounds bounds how many times the model can call tools before
// it must give a final answer â€” prevents a runaway loop if it never stops.
const guideMaxToolRounds = 3

// guideMaxHistoryMessages caps how much stored conversation gets replayed
// into each request â€” conversations persist indefinitely, but the model's
// context window doesn't, so only the most recent turns are sent.
const guideMaxHistoryMessages = 20

const guideSystemPromptTemplate = `You are "Local Guide," a friendly, knowledgeable local travel assistant for Spotto, helping travellers in Sri Lanka find great places and plan their time.

Traveller's saved travel styles: %s
Current weather where they are: %s

Guidelines:
- Keep replies short, warm, and conversational â€” like a knowledgeable local friend, not a search engine.
- TRIP PLANNING: If the traveller asks to plan a trip (e.g., specific dates or locations), DO NOT search immediately. First, ask them about their budget (e.g., luxury $$$, mid-range $$, or budget $).
- VIBE CHECK: If "Traveller's saved travel styles" is "none stated" and they ask for recommendations or a trip, you must ask what kind of vibe they prefer (e.g. quiet, lively, nature) before searching. Once they reply, call save_user_vibe to save their preferences.
- If you don't have enough information to make a good recommendation (e.g. budget, interests, how much time they have, what mood they're in), ask ONE short clarifying question before searching â€” don't guess.
- Once you have enough to go on (including budget and vibe), call search_places to find real matching places. Never invent or recommend a place that search_places didn't actually return. 
- For trip plans, call save_trip to permanently save the generated itinerary using the retrieved places.
- search_places' structured filters (categories, tags, vibe, price, etc.) cover the common cases, but for anything that doesn't map cleanly onto them â€” a specific mood, activity, or phrase the traveller used â€” also pass it as the free-text "query" argument so it can be matched semantically.
- When you recommend places, mention them by name naturally in your reply so the traveller can tell which ones you mean.
- If it's currently raining, lean toward indoor recommendations unless the traveller says otherwise.
- "Current weather where they are" above is only for the traveller's own location. If they ask about weather somewhere else by name (e.g. "is it going to rain in Ella today?"), call get_weather with that place name instead of guessing.`

func (s *Server) handleGuideChat(w http.ResponseWriter, r *http.Request) {
	if s.GeminiAPIKey == "" {
		writeErr(w, http.StatusNotImplemented, "local guide is not configured")
		return
	}
	c := auth.FromContext(r.Context())
	limitKey := c.Subject
	if limitKey == "" {
		limitKey = clientIP(r)
	}
	if s.guideLimit != nil && !s.guideLimit.allowInc(limitKey) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error": "Local Guide rate limit exceeded â€” please wait a moment before sending another message",
		})
		return
	}

	var req struct {
		ConversationID string  `json:"conversationId"`
		Message        string  `json:"message"`
		Lat            float64 `json:"lat"`
		Lng            float64 `json:"lng"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		writeErr(w, http.StatusBadRequest, "message is required")
		return
	}
	if !validateTextLength(w, "Message", req.Message, 1000) {
		return
	}

	convID := req.ConversationID
	if convID == "" {
		convID = newID("gconv")
		if err := s.Store.CreateGuideConversation(r.Context(), convID, c.Subject); err != nil {
			storeErr(w, err)
			return
		}
	} else {
		owner, err := s.Store.GuideConversationOwner(r.Context(), convID)
		if err != nil {
			storeErr(w, err)
			return
		}
		if owner != c.Subject {
			writeErr(w, http.StatusForbidden, "not allowed")
			return
		}
	}

	history, err := s.Store.GuideMessagesByConversation(r.Context(), convID)
	if err != nil {
		storeErr(w, err)
		return
	}

	start := time.Now()
	reply, placeIDs, meta, err := s.runGuideChat(r.Context(), c.Subject, history, req.Message, req.Lat, req.Lng)
	latencyMs := int(time.Since(start).Milliseconds())
	if err != nil {
		status := "error"
		if errors.Is(err, gemini.ErrRateLimited) {
			status = "rate_limited"
		} else {
			log.Printf("guide: chat: %v", err)
		}
		s.recordGuideRequest(r.Context(), convID, c.Subject, status, err.Error(), latencyMs, meta)
		// Includes conversationId even on failure â€” the conversation row
		// already exists (created above), and without this the client has
		// no way to learn its id and ends up creating a brand new
		// conversation on every retry instead of continuing this one.
		if status == "rate_limited" {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":          "Local Guide is busy right now â€” try again in a moment",
				"conversationId": convID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":          "An unexpected error occurred. Please try again later.",
			"conversationId": convID,
		})
		return
	}
	s.recordGuideRequest(r.Context(), convID, c.Subject, "ok", "", latencyMs, meta)

	if err := s.Store.InsertGuideMessage(r.Context(), domain.GuideMessage{
		ConversationID: convID, Role: "user", Content: req.Message,
	}); err != nil {
		log.Printf("guide: save user message: %v", err)
	}
	if err := s.Store.InsertGuideMessage(r.Context(), domain.GuideMessage{
		ConversationID: convID, Role: "assistant", Content: reply, PlaceIDs: placeIDs,
	}); err != nil {
		log.Printf("guide: save assistant message: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"conversationId": convID,
		"reply":          reply,
		"placeIds":       placeIDs,
	})
}

// recordGuideRequest is best-effort telemetry for the admin Local Guide
// dashboard â€” a failure here should never affect the traveller-facing
// response, same convention as InsertGuideMessage's error handling.
func (s *Server) recordGuideRequest(ctx context.Context, convID, travellerID, status, errMsg string, latencyMs int, meta guideRequestMeta) {
	if err := s.Store.InsertGuideRequest(ctx, domain.GuideRequest{
		ConversationID:    convID,
		TravellerID:       travellerID,
		Status:            status,
		ErrorMessage:      errMsg,
		LatencyMs:         latencyMs,
		ToolsUsed:         dedupe(meta.ToolsUsed),
		UsedSemanticQuery: meta.UsedSemanticQuery,
		InputTokens:       meta.InputTokens,
		OutputTokens:      meta.OutputTokens,
	}); err != nil {
		log.Printf("guide: record request: %v", err)
	}
}

// handleGuideLatestConversation lets the app resume a traveller's most
// recent Local Guide conversation on reopen.
func (s *Server) handleGuideLatestConversation(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	conv, err := s.Store.LatestGuideConversation(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	messages, err := s.Store.GuideMessagesByConversation(r.Context(), conv.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversationId": conv.ID,
		"messages":       messages,
	})
}

// guideRequestMeta accumulates telemetry about one runGuideChat call â€” fed
// into recordGuideRequest regardless of whether the call ultimately
// succeeds, so the admin dashboard can see what was attempted even on a
// failed request.
type guideRequestMeta struct {
	ToolsUsed         []string
	UsedSemanticQuery bool
	InputTokens       int
	OutputTokens      int
}

// runGuideChat builds the conversation + tools, and runs the tool-calling
// loop until the model gives a final text answer (or guideMaxToolRounds is
// exhausted). placeIDs accumulates every place any search_places call
// actually returned, across all rounds, for the response's inline cards.
func (s *Server) runGuideChat(ctx context.Context, travellerID string, history []domain.GuideMessage, userMessage string, lat, lng float64) (string, []string, guideRequestMeta, error) {
	var meta guideRequestMeta
	weatherCondition := s.guideWeatherCondition(ctx, lat, lng)
	styles := s.guideTravellerStyles(ctx, travellerID)
	stylesText := "none stated"
	if len(styles) > 0 {
		stylesText = strings.Join(styles, ", ")
	}

	messages := []gemini.Message{
		{Role: "system", Content: fmt.Sprintf(guideSystemPromptTemplate, stylesText, weatherCondition)},
	}
	if len(history) > guideMaxHistoryMessages {
		history = history[len(history)-guideMaxHistoryMessages:]
	}
	for _, m := range history {
		messages = append(messages, gemini.Message{Role: m.Role, Content: m.Content})
	}
	messages = append(messages, gemini.Message{Role: "user", Content: userMessage})

	tools := guideTools()
	var placeIDs []string

	for range guideMaxToolRounds {
		resp, usage, err := gemini.Complete(ctx, s.GeminiAPIKey, messages, tools)
		if usage != nil {
			meta.InputTokens += usage.PromptTokens
			meta.OutputTokens += usage.CompletionTokens
		}
		if err != nil {
			return "", nil, meta, err
		}
		if len(resp.ToolCalls) == 0 {
			return resp.Content, dedupe(placeIDs), meta, nil
		}
		messages = append(messages, *resp)
		for _, tc := range resp.ToolCalls {
			result, ids, usedSemantic := s.runGuideTool(ctx, tc, lat, lng, weatherCondition)
			placeIDs = append(placeIDs, ids...)
			meta.ToolsUsed = append(meta.ToolsUsed, tc.Function.Name)
			if usedSemantic {
				meta.UsedSemanticQuery = true
			}
			messages = append(messages, gemini.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
		}
	}
	return "", nil, meta, fmt.Errorf("guide: exceeded %d tool-calling rounds", guideMaxToolRounds)
}

func (s *Server) guideWeatherCondition(ctx context.Context, lat, lng float64) string {
	if s.TomorrowIOAPIKey == "" || (lat == 0 && lng == 0) {
		return "unknown"
	}
	snap, _, err := s.getOrFetchWeather(ctx, lat, lng)
	if err != nil {
		return "unknown"
	}
	return snap.Now.Condition
}

func (s *Server) guideTravellerStyles(ctx context.Context, travellerID string) []string {
	u, err := s.Store.UserByID(ctx, travellerID)
	if err != nil {
		return nil
	}
	return u.TravelStyle
}

// vendorCategoryIDs lists every real canonical category id, so the
// search_places tool description stays in sync with domain.VendorCategories
// automatically rather than drifting like the old free-text example did.
func vendorCategoryIDs() []string {
	ids := make([]string, 0, len(domain.VendorCategories))
	for _, c := range domain.VendorCategories {
		ids = append(ids, c.ID)
	}
	return ids
}

func guideTools() []gemini.Tool {
	return []gemini.Tool{
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "search_places",
				Description: "Search for real places matching the traveller's preferences. " +
					"Always use this before recommending specific places â€” never invent one.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"categories": map[string]any{
							"type": "array", "items": map[string]any{"type": "string"},
							"description": "Canonical category ids, e.g. [\"cafe\"], [\"surf-school\"]. Real options: " +
								strings.Join(vendorCategoryIDs(), ", "),
						},
						"tags": map[string]any{
							"type": "array", "items": map[string]any{"type": "string"},
							"description": `free-form interests, e.g. ["quiet", "pottery"]`,
						},
						"query": map[string]any{"type": "string",
							"description": "A free-text description of what the traveller wants, for semantic matching " +
								"beyond the structured filters above (e.g. 'a quiet spot to work with good coffee and wifi')."},
						"desiredProductive": map[string]any{"type": "number",
							"description": "1-5, how work-friendly/productive the traveller wants it, omit if no preference"},
						"desiredLoud": map[string]any{"type": "number",
							"description": "1-5, how lively/loud the traveller wants it (1=quiet), omit if no preference"},
						"desiredCrowded": map[string]any{"type": "number",
							"description": "1-5, how crowded the traveller wants it (1=empty), omit if no preference"},
						"maxDistanceKm":  map[string]any{"type": "number"},
						"maxPriceLevel":  map[string]any{"type": "string", "enum": []string{"$", "$$", "$$$"}},
						"requireIndoor":  map[string]any{"type": "boolean"},
						"requireOpenNow": map[string]any{"type": "boolean"},
						"requireSeaView": map[string]any{"type": "boolean"},
						"wantsDeals":     map[string]any{"type": "boolean"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name:        "get_place_details",
				Description: "Get full details for one specific place by id (blurb, hours, contact) to describe it more richly.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{"placeId": map[string]any{"type": "string"}},
					"required":   []string{"placeId"},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "get_weather",
				Description: "Get the current weather/rain forecast for a named place in Sri Lanka " +
					"(town, landmark, or attraction), for when the traveller asks about somewhere other " +
					"than their own current location.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{"type": "string",
							"description": `e.g. "Ella", "Sigiriya", "Galle"`},
					},
					"required": []string{"location"},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name:        "save_user_vibe",
				Description: "Permanently save the traveller's preferred travel styles and vibes to their profile. Use this after asking a new user for their vibe.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{"vibes": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
					"required":   []string{"vibes"},
				},
			},
		},
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name:        "save_trip",
				Description: "Automatically save a generated itinerary to the traveller's account.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string", "description": "Name of the trip"},
						"startDate": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"days": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"date": map[string]any{"type": "string"},
									"places": map[string]any{"type": "array", "items": map[string]any{"type": "string", "description": "Place IDs"}},
								},
							},
						},
					},
					"required":   []string{"name", "startDate", "days"},
				},
			},
		},
	}
}

// runGuideTool executes one tool call and returns its JSON result (fed
// back to the model as a "tool" message), any place ids it surfaced, and
// whether it used a free-text semantic query (search_places only).
func (s *Server) runGuideTool(ctx context.Context, tc gemini.ToolCall, lat, lng float64, weather string) (result string, placeIDs []string, usedSemanticQuery bool) {
	switch tc.Function.Name {
	case "save_user_vibe":
		var args struct {
			Vibes []string `json:"vibes"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		c := auth.FromContext(ctx)
		if u, err := s.Store.UserByID(ctx, c.Subject); err == nil {
			if err := s.Store.UpdateUserProfile(ctx, u.ID, u.Name, u.Country, args.Vibes, u.Interests); err == nil {
				return `{"status":"ok", "message":"vibes saved"}`, nil, false
			}
		}
		return `{"error":"failed to save vibes"}`, nil, false

	case "save_trip":
		var args struct {
			Name      string `json:"name"`
			StartDate string `json:"startDate"`
			Days      []any  `json:"days"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		c := auth.FromContext(ctx)
		
		daysJSON, _ := json.Marshal(args.Days)
		trip := domain.Trip{
			OwnerID:   c.Subject,
			Name:      args.Name,
			StartDate: args.StartDate,
			Days:      daysJSON,
			UpdatedAt: time.Now(),
		}
		
		if existing, err := s.Store.TripByOwner(ctx, c.Subject); err == nil {
			trip.Code = existing.Code
			trip.Collab = existing.Collab
		} else {
			trip.Code = newID("trp")
		}

		// Collect all place IDs to return so the model knows which places are in the trip
		for _, dayAny := range args.Days {
			if dayMap, ok := dayAny.(map[string]any); ok {
				if placesAny, ok := dayMap["places"].([]any); ok {
					for _, p := range placesAny {
						if pStr, ok := p.(string); ok {
							placeIDs = append(placeIDs, pStr)
						}
					}
				}
			}
		}
		
		if err := s.Store.UpsertTrip(ctx, trip); err == nil {
			return `{"status":"ok", "message":"trip saved successfully"}`, placeIDs, false
		}
		return `{"error":"failed to save trip"}`, nil, false

	case "search_places":
		var args struct {
			Categories        []string `json:"categories"`
			Tags              []string `json:"tags"`
			Query             string   `json:"query"`
			DesiredProductive *float64 `json:"desiredProductive"`
			DesiredLoud       *float64 `json:"desiredLoud"`
			DesiredCrowded    *float64 `json:"desiredCrowded"`
			MaxDistanceKm     *float64 `json:"maxDistanceKm"`
			MaxPriceLevel     string   `json:"maxPriceLevel"`
			RequireIndoor     *bool    `json:"requireIndoor"`
			RequireOpenNow    bool     `json:"requireOpenNow"`
			RequireSeaView    *bool    `json:"requireSeaView"`
			WantsDeals        bool     `json:"wantsDeals"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

		if args.MaxPriceLevel != "$" && args.MaxPriceLevel != "$$" && args.MaxPriceLevel != "$$$" {
			args.MaxPriceLevel = ""
		}
		args.DesiredProductive = clampVibe(args.DesiredProductive)
		args.DesiredLoud = clampVibe(args.DesiredLoud)
		args.DesiredCrowded = clampVibe(args.DesiredCrowded)

		places, err := s.Store.Places(ctx)
		if err != nil {
			return `{"error":"could not load places"}`, nil, false
		}

		// Best-effort: a failed/unconfigured embed call just drops the
		// semantic factor back to neutral (see recommend.semanticScore),
		// same graceful-degradation convention as guideWeatherCondition.
		var queryEmbedding []float64
		if q := strings.TrimSpace(args.Query); q != "" {
			if vecs, err := gemini.Embed(ctx, s.GeminiAPIKey, []string{q}); err != nil {
				log.Printf("guide: embed query: %v", err)
			} else {
				queryEmbedding = vecs[0]
			}
		}

		ranked := recommend.Rank(places, recommend.Preferences{
			Categories: args.Categories, Tags: args.Tags,
			DesiredProductive: args.DesiredProductive, DesiredLoud: args.DesiredLoud, DesiredCrowded: args.DesiredCrowded,
			MaxDistanceKm: args.MaxDistanceKm, MaxPriceLevel: args.MaxPriceLevel,
			RequireIndoor: args.RequireIndoor, RequireOpenNow: args.RequireOpenNow, RequireSeaView: args.RequireSeaView,
			WantsDeals: args.WantsDeals, Lat: lat, Lng: lng, QueryEmbedding: queryEmbedding,
		}, weather)
		if len(ranked) > 8 {
			ranked = ranked[:8]
		}

		type resultPlace struct {
			ID      string   `json:"id"`
			Name    string   `json:"name"`
			Reasons []string `json:"reasons"`
		}
		out := make([]resultPlace, 0, len(ranked))
		for _, sc := range ranked {
			out = append(out, resultPlace{ID: sc.Place.ID, Name: sc.Place.Name, Reasons: sc.Reasons})
			placeIDs = append(placeIDs, sc.Place.ID)
		}
		b, _ := json.Marshal(map[string]any{"places": out})
		return string(b), placeIDs, len(queryEmbedding) > 0

	case "get_place_details":
		var args struct {
			PlaceID string `json:"placeId"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		p, err := s.Store.PlaceByID(ctx, args.PlaceID)
		if err != nil {
			return `{"error":"place not found"}`, nil, false
		}
		b, _ := json.Marshal(s.placeJSON(ctx, p))
		return string(b), []string{p.ID}, false

	case "get_weather":
		var args struct {
			Location string `json:"location"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		if s.TomorrowIOAPIKey == "" {
			return `{"error":"weather is not configured"}`, nil, false
		}
		p, err := s.Store.PlaceByName(ctx, strings.TrimSpace(args.Location))
		if err != nil {
			return `{"error":"couldn't find that location"}`, nil, false
		}
		var facts struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}
		_ = json.Unmarshal(p.Data, &facts)
		if facts.Lat == 0 && facts.Lng == 0 {
			return `{"error":"no coordinates for this location"}`, nil, false
		}
		snap, _, err := s.getOrFetchWeather(ctx, facts.Lat, facts.Lng)
		if err != nil {
			return `{"error":"weather unavailable right now"}`, nil, false
		}
		b, _ := json.Marshal(map[string]any{
			"location":          p.Name,
			"condition":         snap.Now.Condition,
			"conditionLabel":    snap.Now.ConditionLabel,
			"tempC":             snap.Now.Temp,
			"precipProbability": snap.Now.PrecipProbability,
			"high":              snap.High,
			"low":               snap.Low,
		})
		return string(b), nil, false

	default:
		return fmt.Sprintf(`{"error":"unknown tool %s"}`, tc.Function.Name), nil, false
	}
}

// clampVibe keeps an LLM-supplied desired-vibe value inside the 1-5 range
// the tool schema promises, rather than letting an out-of-range value
// (hallucinated or otherwise) distort recommend.vibeScore's distance math.
func clampVibe(v *float64) *float64 {
	if v == nil {
		return nil
	}
	c := *v
	if c < 1 {
		c = 1
	} else if c > 5 {
		c = 5
	}
	return &c
}

func dedupe(ids []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

