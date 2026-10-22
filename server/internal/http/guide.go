package httpapi

// Local Guide: a conversational assistant that recommends real places and
// plans trips (the planner's tools are in guide_planner.go). Uses the configured Gemini chat
// model (GEMINI_CHAT_MODEL, see internal/gemini) via its OpenAI-compatibility endpoint, with tool
// calling — the model decides for itself whether it has enough information to
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
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/recommend"
)

// guideMaxToolRounds bounds how many times the model can call tools before
// it must give a final answer — prevents a runaway loop if it never stops.
// Planning a trip takes several: find_places a few times, then the plan.
const guideMaxToolRounds = 6

// guideBudget bounds one whole guide reply (all tool rounds plus retries).
const guideBudget = 90 * time.Second

// guideMaxHistoryMessages caps how much stored conversation gets replayed
// into each request — conversations persist indefinitely, but the model's
// context window doesn't, so only the most recent turns are sent.
const guideMaxHistoryMessages = 20

const guideSystemPromptTemplate = `You are "Local Guide," a friendly, knowledgeable travel assistant in the Spotto app. You help travellers anywhere in the world find great places and plan their trips.

Today's date: %s
Traveller's location: %s
Traveller's saved travel styles: %s
Current weather where they are: %s

QUICK ASKS ("any good restaurants near me?", "where can I get coffee?", "something fun to do now"):
- Call search_places straight away — don't ask anything first. Pick categories from what they asked for, and lean on their saved travel styles and the weather. When their location is known, never ask which city or town they're in. If they ask about somewhere else by name, use find_places with that place in the query instead.
- Recommend 2-3 of the places the tools returned, by their exact name, each with one short reason. Mention distance (distanceKm), price level or rating when it helps.
- Then end with ONE short follow-up offering to narrow it down (budget, a different vibe, somewhere closer). Only one question.
- If their saved travel styles are "none stated", still search; make the follow-up about what vibe they like (quiet, lively, nature…). When they tell you, call save_user_vibe.

TRIP PLANNING ("plan a trip to…", "I'm visiting Sigiriya, create a trip", multi-day or dated plans):
1. First call ask_trip_questions — never write a plan straight away. If they haven't said where they're going, the first question is where (kind "text"); if they only named a country or a big region, ask which towns or areas. Ask only what they haven't told you, usually: travel dates or how many nights (kind "dates"), whether they've booked a hotel and which one (kind "text" with options like "Not booked yet"), whether they're staying only in that area or open to nearby day trips (kind "single"), what they enjoy (kind "multi": e.g. history & culture, hiking, nature & wildlife, food, relaxing, photography), and pace (relaxed / balanced / packed). 3-5 questions.
2. Turn any relative dates ("this weekend", "next Friday", "in two weeks") into real dates using today's date, and pass startDate as YYYY-MM-DD. When they answer, call find_places a few times with specific queries (the main sights, things matching their interests, good food near where they stay), then call propose_trip_plan. Use only refs from the tools. Put each day's stops in a sensible order with realistic times (e.g. climb rocks early, sunset spots late), 3-5 stops a day including a meal. Every note is 1-2 sentences: why it's worth it plus one practical tip.
3. The message for a card tool is 1-2 short sentences. Never repeat the questions or the plan as text — the app shows them as cards.
4. When they ask to add the plan, or some places, to their calendar or trip ("add this to my calendar", "put Pidurangala on day 2"), call add_to_calendar. Use mode "ask" unless they said whether it's a new trip or their current one. For only some places, pass just those as stops (refs from the plan or find_places). To add a place that isn't in the plan yet, find it with find_places first.
5. If they ask to change the plan (more relaxed, swap a place, add a day), call propose_trip_plan again with the updated plan.

ALWAYS:
- Keep replies short, warm and conversational — like a knowledgeable local friend, not a search engine.
- Never invent a place, distance, price or rating; only use what the tools returned.
- search_places' structured filters cover the common cases; for anything else (a mood, an activity, a phrase they used) also pass it as the free-text "query".
- Don't put links, URLs, ids or refs in your text — the app shows a card for every place.
- If it's currently raining, lean toward indoor recommendations unless the traveller says otherwise.
- "Current weather where they are" is only for the traveller's own location. For weather somewhere else by name, call get_weather.`

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
			"error": "Local Guide rate limit exceeded — please wait a moment before sending another message",
		})
		return
	}

	var req struct {
		ConversationID string  `json:"conversationId"`
		Message        string  `json:"message"`
		Lat            float64 `json:"lat"`
		Lng            float64 `json:"lng"`
		// False when the app has no real fix (permission denied, location
		// off); absent from older app builds — see newGuideLocation.
		LocationKnown *bool `json:"locationKnown"`
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

	// The model call runs on a context detached from the request: if the
	// app gives up waiting (slow model, bad network), the answer is still
	// produced and saved below, so it appears when the traveller reopens the
	// conversation instead of being thrown away mid-call. guideBudget bounds
	// it instead, and the write deadline is pushed past the server-wide 30s
	// WriteTimeout so a slow-but-successful answer can still be delivered.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(guideBudget + 15*time.Second))
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), guideBudget)
	defer cancel()

	start := time.Now()
	loc := newGuideLocation(req.Lat, req.Lng, req.LocationKnown)
	res, meta, err := s.runGuideChat(ctx, c.Subject, history, req.Message, loc)
	latencyMs := int(time.Since(start).Milliseconds())
	if err != nil {
		status := "error"
		if errors.Is(err, gemini.ErrRateLimited) {
			status = "rate_limited"
		}
		log.Printf("guide: chat (%s): %v", status, err)
		s.recordGuideRequest(ctx, convID, c.Subject, status, err.Error(), latencyMs, meta)
		// Includes conversationId even on failure — the conversation row
		// already exists (created above), and without this the client has
		// no way to learn its id and ends up creating a brand new
		// conversation on every retry instead of continuing this one.
		if errors.Is(err, gemini.ErrDailyQuota) {
			// Google's daily caps reset at midnight Pacific; "try again in a
			// moment" would just send the traveller into the same wall.
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":          "Local Guide has reached today's AI usage limit. It resets at midnight Pacific time (1:30 PM in Sri Lanka).",
				"conversationId": convID,
			})
			return
		}
		if status == "rate_limited" {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error":          "Local Guide is busy right now — try again in a moment",
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
	s.recordGuideRequest(ctx, convID, c.Subject, "ok", "", latencyMs, meta)

	reply := res.Reply
	// A card (questions, plan, calendar) shows its own places.
	var cards []guidePlaceCard
	if res.Payload == nil {
		cards = s.guideCardsForReply(ctx, reply, res.PlaceIDs, loc)
	}
	placeIDs := make([]string, 0, len(cards))
	for _, card := range cards {
		placeIDs = append(placeIDs, card.ID)
	}
	var payload json.RawMessage
	if res.Payload != nil {
		payload, _ = json.Marshal(res.Payload)
	}

	if err := s.Store.InsertGuideMessage(ctx, domain.GuideMessage{
		ConversationID: convID, Role: "user", Content: req.Message,
	}); err != nil {
		log.Printf("guide: save user message: %v", err)
	}
	if err := s.Store.InsertGuideMessage(ctx, domain.GuideMessage{
		ConversationID: convID, Role: "assistant", Content: reply, PlaceIDs: placeIDs, Payload: payload,
	}); err != nil {
		log.Printf("guide: save assistant message: %v", err)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"conversationId": convID,
		"reply":          reply,
		"placeIds":       placeIDs, // kept for app builds older than "places"
		"places":         cards,
		"payload":        payload,
	})
}

// guideCardsForReply turns every place the tools surfaced into the cards
// shown under the reply — the ones it names, in the order it names them.
func (s *Server) guideCardsForReply(ctx context.Context, reply string, surfacedIDs []string, loc guideLocation) []guidePlaceCard {
	byID := s.guidePlacesByID(ctx, surfacedIDs)
	names := make(map[string]string, len(byID))
	for id, p := range byID {
		names[id] = p.Name
	}
	var cards []guidePlaceCard
	for _, id := range guideCardOrder(reply, surfacedIDs, names) {
		if p, ok := byID[id]; ok {
			cards = append(cards, guidePlaceCardFor(p, s.placeJSON(ctx, p), loc))
		}
	}
	return cards
}

// guidePlacesByID looks up a reply's places (a handful at most — the tools
// return up to 8 each). A place deleted since is just skipped.
func (s *Server) guidePlacesByID(ctx context.Context, ids []string) map[string]domain.Place {
	ids = dedupe(ids)
	if len(ids) > 24 {
		ids = ids[:24]
	}
	out := make(map[string]domain.Place, len(ids))
	for _, id := range ids {
		if p, err := s.Store.PlaceByID(ctx, id); err == nil {
			out[id] = p
		}
	}
	return out
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
	// Cards for the places past replies named, keyed by id, so a resumed
	// conversation shows the same photo cards (without distance — where the
	// traveller was then isn't stored).
	var ids []string
	for _, m := range messages {
		ids = append(ids, m.PlaceIDs...)
	}
	if len(ids) > 60 {
		ids = ids[len(ids)-60:]
	}
	cards := map[string]guidePlaceCard{}
	for id, p := range s.guidePlacesByID(r.Context(), ids) {
		cards[id] = guidePlaceCardFor(p, s.placeJSON(r.Context(), p), guideLocation{})
	}
	// Spotto photo links in stored cards carry short-lived tokens.
	for i := range messages {
		messages[i].Payload = s.resignPayload(messages[i].Payload)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"conversationId": conv.ID,
		"messages":       messages,
		"places":         cards,
	})
}

// guideRequestMeta accumulates telemetry about one runGuideChat call — fed
// into recordGuideRequest regardless of whether the call ultimately
// succeeds, so the admin dashboard can see what was attempted even on a
// failed request.
type guideRequestMeta struct {
	ToolsUsed         []string
	UsedSemanticQuery bool
	InputTokens       int
	OutputTokens      int
}

// guideChatResult is one reply: its text, every catalog place the tools
// surfaced (for the inline cards), and a card payload when a planner tool
// produced one.
type guideChatResult struct {
	Reply    string
	PlaceIDs []string
	Payload  *guidePayload
}

// runGuideChat builds the conversation + tools, and runs the tool-calling
// loop until the model gives a final text answer, calls a card tool, or
// guideMaxToolRounds is exhausted.
func (s *Server) runGuideChat(ctx context.Context, travellerID string, history []domain.GuideMessage, userMessage string, loc guideLocation) (guideChatResult, guideRequestMeta, error) {
	var meta guideRequestMeta
	weatherCondition := s.guideWeatherCondition(ctx, loc.Lat, loc.Lng)
	styles := s.guideTravellerStyles(ctx, travellerID)
	stylesText := "none stated"
	if len(styles) > 0 {
		stylesText = strings.Join(styles, ", ")
	}

	messages := []gemini.Message{
		{Role: "system", Content: guideSystemPrompt(time.Now(), loc, stylesText, weatherCondition)},
	}
	// Plans from the whole conversation stay addable to the calendar; only
	// the turns replayed to the model are capped.
	turn := newGuideTurn(history)
	fullHistory := history
	if len(history) > guideMaxHistoryMessages {
		history = history[len(history)-guideMaxHistoryMessages:]
	}
	for _, m := range history {
		content := m.Content
		if sum := guidePayloadSummary(m.Payload); sum != "" {
			content += "\n" + sum
		}
		messages = append(messages, gemini.Message{Role: m.Role, Content: content})
	}
	messages = append(messages, gemini.Message{Role: "user", Content: userMessage})

	tools := guideTools()
	var placeIDs []string
	// base is the conversation without any tool turns; toolResults collects
	// every tool result as text, for the final round below.
	base := append([]gemini.Message(nil), messages...)
	var toolResults []string

	for round := range guideMaxToolRounds {
		roundMessages, roundTools := messages, tools
		if round == guideMaxToolRounds-1 {
			// Last round: no more searching. Gemini keeps calling tools
			// whenever the history holds tool turns — even with no tools
			// declared or tool_choice "none" — so the final request drops
			// the tool turns and hands their results over as a note instead.
			// The card tools stay available, so a trip plan that took every
			// round still arrives as a plan card rather than a wall of text.
			note := "Results from the tools you already called:\n" + strings.Join(toolResults, "\n") +
				"\nAnswer the traveller now using only these results — don't search again. If they asked for a trip plan, " +
				"call propose_trip_plan with these places; otherwise reply in text."
			roundMessages = append(base[:len(base):len(base)], gemini.Message{Role: "system", Content: note})
			roundTools = plannerCardTools()
		}
		resp, usage, err := s.aiComplete(ctx, aiFeatureGuideChat, roundMessages, roundTools)
		if usage != nil {
			meta.InputTokens += usage.PromptTokens
			// Same thinking-token accounting as recordAIUsage, so per-user
			// totals match the dashboard's usage numbers.
			meta.OutputTokens += max(usage.CompletionTokens, usage.TotalTokens-usage.PromptTokens)
		}
		if err != nil {
			return guideChatResult{}, meta, err
		}
		if len(resp.ToolCalls) == 0 {
			if strings.TrimSpace(resp.Content) == "" {
				return guideChatResult{}, meta, errors.New("guide: model returned an empty reply")
			}
			return guideChatResult{Reply: resp.Content, PlaceIDs: dedupe(placeIDs)}, meta, nil
		}
		messages = append(messages, *resp)
		for _, tc := range resp.ToolCalls {
			if result, ok := s.runPlannerTool(ctx, tc, turn, fullHistory); ok {
				meta.ToolsUsed = append(meta.ToolsUsed, tc.Function.Name)
				messages = append(messages, gemini.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
				toolResults = append(toolResults, tc.Function.Name+": "+result)
				continue
			}
			result, ids, usedSemantic := s.runGuideTool(ctx, tc, loc, weatherCondition)
			placeIDs = append(placeIDs, ids...)
			meta.ToolsUsed = append(meta.ToolsUsed, tc.Function.Name)
			if usedSemantic {
				meta.UsedSemanticQuery = true
			}
			messages = append(messages, gemini.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
			toolResults = append(toolResults, tc.Function.Name+": "+result)
		}
		// A card ends the reply: the card is the answer, with its short
		// message as the text.
		if turn.payload != nil {
			return guideChatResult{Reply: turn.message, PlaceIDs: dedupe(placeIDs), Payload: turn.payload}, meta, nil
		}
	}
	return guideChatResult{}, meta, fmt.Errorf("guide: model kept calling tools after %d rounds", guideMaxToolRounds)
}

func guideSystemPrompt(now time.Time, loc guideLocation, stylesText, weatherCondition string) string {
	return fmt.Sprintf(guideSystemPromptTemplate, now.Format("Monday 2 January 2006"), loc.promptLine(), stylesText, weatherCondition)
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
	return append([]gemini.Tool{
		{
			Type: "function",
			Function: gemini.ToolFunction{
				Name: "search_places",
				Description: "Search Spotto's own listings near the traveller (cafés, restaurants, stays, activities with " +
					"vibe, price, deals and opening data). Use this for quick asks near them; use find_places for anywhere else.",
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
				Description: "Get the current weather/rain forecast for a named place " +
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
	}, plannerTools()...)
}

// runGuideTool executes one tool call and returns its JSON result (fed
// back to the model as a "tool" message), any place ids it surfaced, and
// whether it used a free-text semantic query (search_places only).
func (s *Server) runGuideTool(ctx context.Context, tc gemini.ToolCall, loc guideLocation, weather string) (result string, placeIDs []string, usedSemanticQuery bool) {
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
			if vecs, err := s.aiEmbed(ctx, aiFeatureGuideEmbed, []string{q}); err != nil {
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
			WantsDeals: args.WantsDeals, Lat: loc.Lat, Lng: loc.Lng, QueryEmbedding: queryEmbedding,
		}, weather)
		if len(ranked) > 8 {
			ranked = ranked[:8]
		}

		out := guideSearchResults(ranked, loc)
		for _, p := range out {
			placeIDs = append(placeIDs, p.ID)
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
		var facts struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}
		name := strings.TrimSpace(args.Location)
		if p, err := s.Store.PlaceByName(ctx, name); err == nil {
			name = p.Name
			_ = json.Unmarshal(p.Data, &facts)
		} else if s.Places != nil {
			// Not a Spotto listing (most towns aren't): find it on Google.
			g, err := s.Places.SearchText(ctx, name)
			if err != nil {
				return `{"error":"couldn't find that location"}`, nil, false
			}
			name, facts.Lat, facts.Lng = g.Name, g.Lat, g.Lng
		} else {
			return `{"error":"couldn't find that location"}`, nil, false
		}
		if facts.Lat == 0 && facts.Lng == 0 {
			return `{"error":"no coordinates for this location"}`, nil, false
		}
		snap, _, err := s.getOrFetchWeather(ctx, facts.Lat, facts.Lng)
		if err != nil {
			return `{"error":"weather unavailable right now"}`, nil, false
		}
		b, _ := json.Marshal(map[string]any{
			"location":          name,
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
