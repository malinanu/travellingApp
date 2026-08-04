package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
)

// handleRegisterPushToken saves a user's device push notification token.
func (s *Server) handleRegisterPushToken(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		DeviceToken string `json:"deviceToken"`
		Platform    string `json:"platform"` // android | ios | web
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.DeviceToken == "" {
		writeErr(w, http.StatusBadRequest, "deviceToken required")
		return
	}
	if req.Platform == "" {
		req.Platform = "android"
	}

	token := domain.UserPushToken{
		ID:          newID("ptok"),
		UserID:      c.Subject,
		DeviceToken: req.DeviceToken,
		Platform:    req.Platform,
		CreatedAt:   time.Now(),
	}

	if err := s.Store.UpsertUserPushToken(r.Context(), token); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not save push token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleUnregisterPushToken deletes a user's device push notification token on sign out.
func (s *Server) handleUnregisterPushToken(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		DeviceToken string `json:"deviceToken"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.DeviceToken == "" {
		writeErr(w, http.StatusBadRequest, "deviceToken required")
		return
	}

	if err := s.Store.DeleteUserPushToken(r.Context(), c.Subject, req.DeviceToken); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not unregister push token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SendUserPushNotification dispatches a push notification to all registered devices of a user.
func (s *Server) SendUserPushNotification(ctx context.Context, userID, title, body string, data map[string]string) {
	tokens, err := s.Store.UserPushTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}

	log.Printf("[PushNotification] Sending alert to user=%s (devices=%d): '%s' - '%s'",
		userID, len(tokens), title, body)

	// Simulated / FCM transport dispatch loop
	for _, tok := range tokens {
		log.Printf("[FCM] Dispatching push to token=%s payload=%v", tok, data)
	}
}
