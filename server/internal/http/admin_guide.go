package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleAdminGuideStats powers the Local Guide dashboard's stat-card grid
// and tool-usage breakdown — usage/reliability over the trailing 7 days
// plus the embedding backfill's overall progress.
func (s *Server) handleAdminGuideStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.GuideStats(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	usage, err := s.Store.GuideToolUsage(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	limit, err := s.Store.GetGuideRateLimitRPM(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":        stats,
		"toolUsage":    usage,
		"rateLimitRPM": limit,
	})
}

// handleAdminGuideUsers lists the most recently active Local Guide users
// for the dashboard's browsable list, along with their analytics.
func (s *Server) handleAdminGuideUsers(w http.ResponseWriter, r *http.Request) {
	const limit = 50
	users, err := s.Store.RecentGuideUsers(r.Context(), limit)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// handleAdminGuideConversationDetail returns one conversation's full
// transcript — an admin can inspect any conversation, unlike the
// traveller-facing endpoint which enforces ownership.
func (s *Server) handleAdminGuideConversationDetail(w http.ResponseWriter, r *http.Request) {
	messages, err := s.Store.GuideMessagesByConversation(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

// handleAdminGuideUserMessages returns all guide messages for a given traveller
func (s *Server) handleAdminGuideUserMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := s.Store.GuideMessagesByTraveller(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (s *Server) handleAdminPutGuideRateLimit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RateLimitRPM int `json:"rateLimitRPM"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.RateLimitRPM < 1 {
		writeErr(w, http.StatusBadRequest, "rate limit must be >= 1")
		return
	}
	if err := s.Store.SetGuideRateLimitRPM(r.Context(), body.RateLimitRPM); err != nil {
		storeErr(w, err)
		return
	}
	s.guideLimit.setMax(body.RateLimitRPM)
	writeJSON(w, http.StatusOK, map[string]int{"rateLimitRPM": body.RateLimitRPM})
}
