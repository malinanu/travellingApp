package httpapi

import (
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
	toolUsage, err := s.Store.GuideToolUsage(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":     stats,
		"toolUsage": toolUsage,
	})
}

// handleAdminGuideConversations lists the most recently active Local Guide
// conversations for the dashboard's browsable list.
func (s *Server) handleAdminGuideConversations(w http.ResponseWriter, r *http.Request) {
	const limit = 50
	conversations, err := s.Store.RecentGuideConversations(r.Context(), limit)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conversations)
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
