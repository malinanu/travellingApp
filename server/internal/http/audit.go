package httpapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
)

// logAdminAction is a best-effort audit write, called after an admin
// mutation has already succeeded — a failure here is logged, not
// propagated, since the parent request already did what it promised.
func (s *Server) logAdminAction(ctx context.Context, action, targetType, targetID string, metadata any) {
	c := auth.FromContext(ctx)
	if c == nil {
		return
	}
	// A zero-value User (email/name blank) on lookup failure still leaves a
	// usable entry — actorID alone is enough to trace the action.
	actor, _ := s.Store.UserByID(ctx, c.Subject)
	var raw json.RawMessage
	if metadata != nil {
		raw, _ = json.Marshal(metadata)
	}
	entry := domain.AdminAuditLog{
		ActorID:    c.Subject,
		ActorEmail: actor.Email,
		ActorName:  actor.Name,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Metadata:   raw,
		CreatedAt:  time.Now(),
	}
	if err := s.Store.InsertAdminAuditLog(ctx, entry); err != nil {
		log.Printf("admin audit log insert failed (action=%s target=%s/%s): %v", action, targetType, targetID, err)
	}
}

// handleAdminAuditLog lists audit entries most-recent-first. Gated on
// RequireSuperAdmin at the route level (see server.go).
func (s *Server) handleAdminAuditLog(w http.ResponseWriter, r *http.Request) {
	beforeID, _ := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64)
	entries, err := s.Store.AdminAuditLogs(r.Context(), beforeID,
		r.URL.Query().Get("targetType"), r.URL.Query().Get("actorId"), 100)
	if err != nil {
		storeErr(w, err)
		return
	}
	if entries == nil {
		entries = []domain.AdminAuditLog{}
	}
	writeJSON(w, http.StatusOK, entries)
}
