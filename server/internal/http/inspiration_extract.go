package httpapi

// Inspiration Board "extract places" — given an Instagram reel/post URL
// already saved to the trip's inspiration list, best-effort pull the real
// places mentioned in its caption so the traveller can add them to their
// trip or favorite them, instead of just holding a bookmarked link.
//
// There's no official Instagram API for arbitrary public content, so this
// shells out to yt-dlp (a static binary baked into the server image, see
// Dockerfile) to fetch the post's caption without downloading video. The
// caption then goes through the same two steps Local Guide already uses:
// Gemini extracts candidate place names, and each name is checked against
// the real catalog via search_places (internal/http/guide.go's
// runGuideTool) — a name search_places doesn't match is returned as a
// loose, name-only suggestion rather than dropped, mirroring how
// CustomStopInfo already treats non-curated stops on the Flutter side.

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/spotto/server/internal/gemini"
)

type extractedPlace struct {
	Name     string  `json:"name"`
	Matched  bool    `json:"matched"`
	PlaceID  string  `json:"placeId,omitempty"`
	Lat      float64 `json:"lat,omitempty"`
	Lng      float64 `json:"lng,omitempty"`
	Category string  `json:"category,omitempty"`
}

// maxExtractedPlaceNames bounds how many search_places lookups one request
// can trigger, so a long/spammy caption can't blow up latency or cost.
const maxExtractedPlaceNames = 8

func (s *Server) handleExtractPlacesFromLink(w http.ResponseWriter, r *http.Request) {
	if s.GeminiAPIKey == "" {
		writeErr(w, http.StatusNotImplemented, "place extraction is not configured")
		return
	}
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		writeErr(w, http.StatusNotImplemented, "place extraction is not available in this environment")
		return
	}

	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, err := url.Parse(strings.TrimSpace(body.URL))
	if err != nil || !isInstagramHost(u.Host) {
		writeErr(w, http.StatusBadRequest, "only instagram.com links are supported")
		return
	}

	caption, err := fetchInstagramCaption(r.Context(), u.String())
	if err != nil {
		writeErr(w, http.StatusBadGateway, "couldn't read that link — it may be private or unavailable")
		return
	}
	if strings.TrimSpace(caption) == "" {
		writeJSON(w, http.StatusOK, map[string]any{"places": []extractedPlace{}})
		return
	}

	names, err := s.extractPlaceNames(r.Context(), caption)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "couldn't analyze that link right now")
		return
	}
	if len(names) > maxExtractedPlaceNames {
		names = names[:maxExtractedPlaceNames]
	}

	places := make([]extractedPlace, 0, len(names))
	for _, name := range names {
		places = append(places, s.matchExtractedPlaceName(r.Context(), name))
	}
	writeJSON(w, http.StatusOK, map[string]any{"places": places})
}

// isInstagramHost requires an exact instagram.com host (or a subdomain of
// it) — a substring check like strings.Contains(host, "instagram.com")
// would wrongly accept an attacker-controlled host such as
// "instagram.com.evil.example", letting a caller point the server's
// yt-dlp fetch at an arbitrary URL.
func isInstagramHost(host string) bool {
	host = strings.ToLower(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return host == "instagram.com" || strings.HasSuffix(host, ".instagram.com")
}

// fetchInstagramCaption shells out to yt-dlp for the post's metadata
// (without downloading the video) and returns its caption/description.
func fetchInstagramCaption(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", "--dump-json", "--skip-download", "--no-warnings", url)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp: %w", err)
	}

	var meta struct {
		Description string `json:"description"`
		Title       string `json:"title"`
	}
	if err := json.Unmarshal(out, &meta); err != nil {
		return "", fmt.Errorf("yt-dlp: parse metadata: %w", err)
	}
	if meta.Description != "" {
		return meta.Description, nil
	}
	return meta.Title, nil
}

// extractPlaceNames asks Gemini for a plain JSON array of real, named
// places mentioned in the caption — no tools, just extraction, kept
// separate from the matching step so each is easy to reason about.
func (s *Server) extractPlaceNames(ctx context.Context, caption string) ([]string, error) {
	prompt := "Below is an Instagram post caption from a travel video about Sri Lanka. " +
		"List any real, specific, named places mentioned or clearly implied (restaurants, cafes, " +
		"hotels, beaches, attractions, viewpoints, etc.) — not generic terms like \"the beach\" or \"a cafe\". " +
		"Respond with ONLY a JSON array of place name strings, nothing else. If none, respond with []." +
		"\n\nCaption:\n" + caption

	msg, _, err := s.aiComplete(ctx, aiFeatureInspiration, []gemini.Message{
		{Role: "user", Content: prompt},
	}, nil)
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(msg.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var names []string
	if err := json.Unmarshal([]byte(raw), &names); err != nil {
		return nil, fmt.Errorf("parse place names: %w", err)
	}
	return names, nil
}

// matchExtractedPlaceName checks one caption-derived name against the real
// catalog by reusing runGuideTool's search_places case directly — never a
// second, parallel place-search implementation. A name it can't confidently
// match comes back as a loose, name-only suggestion.
func (s *Server) matchExtractedPlaceName(ctx context.Context, name string) extractedPlace {
	args, _ := json.Marshal(map[string]string{"query": name})
	_, placeIDs, _ := s.runGuideTool(ctx, gemini.ToolCall{
		Function: gemini.FunctionCall{Name: "search_places", Arguments: string(args)},
	}, 0, 0, "")

	if len(placeIDs) == 0 {
		return extractedPlace{Name: name, Matched: false}
	}

	p, err := s.Store.PlaceByID(ctx, placeIDs[0])
	if err != nil {
		return extractedPlace{Name: name, Matched: false}
	}
	var facts struct {
		Lat        float64 `json:"lat"`
		Lng        float64 `json:"lng"`
		CategoryID string  `json:"categoryId"`
	}
	_ = json.Unmarshal(p.Data, &facts)
	return extractedPlace{
		Name:     p.Name,
		Matched:  true,
		PlaceID:  p.ID,
		Lat:      facts.Lat,
		Lng:      facts.Lng,
		Category: facts.CategoryID,
	}
}
