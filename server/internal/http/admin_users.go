package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// handleAdminUsers searches users by ?q= (name, email or id).
func (s *Server) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	users, err := s.Store.SearchUsers(r.Context(), q, 50)
	if err != nil {
		log.Printf("admin users search q=%q: %v", q, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't search users. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// handleAdminUserDetail is one user's profile, signed-in devices, quest
// history and the admin actions taken on them.
func (s *Server) handleAdminUserDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	u, err := s.Store.UserByID(ctx, id)
	if err != nil {
		storeErr(w, err)
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
	sessions, err := s.Store.ActiveUserSessions(ctx, id)
	if err != nil {
		log.Printf("admin user detail sessions user=%s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't load this user's devices. Please try again.")
		return
	}
	quests, err := s.Store.ListQuestCompletions(ctx, domain.QuestCompletionFilter{TravellerID: id, Limit: 100})
	if err != nil {
		log.Printf("admin user detail quests user=%s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't load this user's quest history. Please try again.")
		return
	}
	audit, err := s.Store.AuditLogsByTarget(ctx, "user", id, 50)
	if err != nil {
		log.Printf("admin user detail audit user=%s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't load this user's history. Please try again.")
		return
	}
	if audit == nil {
		audit = []domain.AdminAuditLog{}
	}
	perms := []string{}
	if u.Role == domain.RoleAdmin {
		got, err := s.Store.AdminPermissions(ctx, id)
		if err != nil {
			log.Printf("admin user detail permissions user=%s: %v", id, err)
			writeErr(w, http.StatusInternalServerError, "Couldn't load this admin's permissions. Please try again.")
			return
		}
		perms = nonNilStrings(got)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":             u,
		"sessions":         sessions,
		"questCompletions": quests,
		"audit":            audit,
		"adminPermissions": perms,
		"isSuperAdmin":     u.Role == domain.RoleAdmin && s.isSuperAdmin(u.Email),
	})
}

// handleAdminRevokeUserSessions signs a user out of every device, e.g. when
// they report a lost phone. Every access and refresh token they hold stops
// working; they sign in again on the devices they still have.
func (s *Server) handleAdminRevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Reason string `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "reason required")
		return
	}
	ended, err := s.Store.RevokeAllUserSessions(r.Context(), id)
	if errors.Is(err, postgres.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		log.Printf("admin revoke sessions user=%s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't sign this user out. Nothing was changed; please try again.")
		return
	}
	s.logAdminAction(r.Context(), "user.revoke_sessions", "user", id, map[string]any{"reason": reason, "sessionsEnded": ended})
	writeJSON(w, http.StatusOK, map[string]any{"sessionsEnded": ended})
}
