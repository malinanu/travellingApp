package httpapi

// Local Guide: a conversational assistant that recommends real places and
// (eventually, in a fast-follow) plans trips. Uses OpenRouter's free-tier
// google/gemma-4-31b-it:free model (internal/openrouter) with tool calling
// — the model decides for itself whether it has enough information to
// search, or should ask a clarifying question first, steered entirely by
// guideSystemPrompt rather than separate orchestration code. The actual
// place-matching is internal/recommend's deterministic scoring algorithm;
// the LLM never invents a place, it only describes/explains ones
// search_places actually returned.
//
// Conversations persist (guide_conversations/guide_messages) so a
// traveller can close the app and resume later, but only user-visible
// turns are stored — each new message re-derives whatever tool calls it
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
	"github.com/spotto/server/internal/openrouter"
	"github.com/spotto/server/internal/recommend"
)

// guideMaxToolRounds bounds how many times the model can call tools before
// it must give a final answer — prevents a runaway loop if it never stops.
const guideMaxToolRounds = 3

// guideMaxHistoryMessages caps how much stored conversation gets replayed
// into each request — conversations persist indefinitely, but the model's
// context window doesn't, so only the most recent turns are sent.
const guideMaxHistoryMessages = 20

const guideSystemPromptTemplate = `You are "Local Guide," a friendly, knowledgeable local travel assistant for Spotto, helping travellers in Sri Lanka find great places and plan their time.

Traveller's saved travel styles: %s
Current weather where they are: %s

Guidelines:
- Keep replies short, warm, and conversational — like a knowledgeable local friend, not a search engine.
- If you don't have enough information to make a good recommendation (e.g. budget, interests, how much time they have, what mood they're in), ask ONE short clarifying question before searching — don't guess.
- Once you have enough to go on, call search_places to find real matching places. Never invent or recommend a place that search_places didn't actually return.
- search_places' structured filters (categories, tags, vibe, price, etc.) cover the common cases, but for anything that doesn't map cleanly onto them — a specific mood, activity, or phrase the traveller used — also pass it as the free-text "query" argument so it can be matched semantically.
- When you recommend places, mention them by name naturally in your reply so the traveller can tell which ones you mean.
- If it's currently raining, lean toward indoor recommendations unless the traveller says otherwise.`

func (s *Server) handleGuideChat(w http.ResponseWriter, r *http.Request) {
	if s.OpenRouterAPIKey == "" {
		writeErr(w, http.StatusNotImplemented, "local guide is not configured")
		return
	}
	c := auth.FromContext(r.Context())
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
		if errors.Is(err, openrouter.ErrRateLimited) {
			status = "rate_limited"
		} else {
			log.Printf("guide: chat: %v", err)
		}
		s.recordGuideRequest(r.Context(), convID, c.Subject, status, err.Error(), latencyMs, meta)
		if status == "rate_limited" {
			writeErr(w, http.StatusTooManyRequests, "Local Guide is busy right now — try again in a moment")
			return
		}
		writeErr(w, http.StatusInternalServerError, "internal error")
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
// dashboard — a failure here should never affect the traveller-facing
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

// guideRequestMeta accumulates telemetry about one runGuideChat call — fed
// into recordGuideRequest regardless of whether the call ultimately
// succeeds, so the admin dashboard can see what was attempted even on a
// failed request.
type guideRequestMeta struct {
	ToolsUsed         []string
	UsedSemanticQuery bool
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

	messages := []openrouter.Message{
		{Role: "system", Content: fmt.Sprintf(guideSystemPromptTemplate, stylesText, weatherCondition)},
	}
	if len(history) > guideMaxHistoryMessages {
		history = history[len(history)-guideMaxHistoryMessages:]
	}
	for _, m := range history {
		messages = append(messages, openrouter.Message{Role: m.Role, Content: m.Content})
	}
	messages = append(messages, openrouter.Message{Role: "user", Content: userMessage})

	tools := guideTools()
	var placeIDs []string

	for range guideMaxToolRounds {
		resp, err := openrouter.Complete(ctx, s.OpenRouterAPIKey, messages, tools)
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
			messages = append(messages, openrouter.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
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

func guideTools() []openrouter.Tool {
	return []openrouter.Tool{
		{
			Type: "function",
			Function: openrouter.ToolFunction{
				Name: "search_places",
				Description: "Search for real places matching the traveller's preferences. " +
					"Always use this before recommending specific places — never invent one.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"categories": map[string]any{
							"type": "array", "items": map[string]any{"type": "string"},
							"description": `e.g. ["cafe"], ["surf"]`,
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
			Function: openrouter.ToolFunction{
				Name:        "get_place_details",
				Description: "Get full details for one specific place by id (blurb, hours, contact) to describe it more richly.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{"placeId": map[string]any{"type": "string"}},
					"required":   []string{"placeId"},
				},
			},
		},
	}
}

// runGuideTool executes one tool call and returns its JSON result (fed
// back to the model as a "tool" message), any place ids it surfaced, and
// whether it used a free-text semantic query (search_places only).
func (s *Server) runGuideTool(ctx context.Context, tc openrouter.ToolCall, lat, lng float64, weather string) (result string, placeIDs []string, usedSemanticQuery bool) {
	switch tc.Function.Name {
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
			if vecs, err := openrouter.Embed(ctx, s.OpenRouterAPIKey, []string{q}); err != nil {
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
