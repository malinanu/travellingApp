package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// refreshReuseGrace is how soon after a rotation the replaced refresh token
// may turn up again without being treated as stolen — two tabs or retries
// refreshing at the same moment send the same token.
const refreshReuseGrace = time.Minute

// sessionClient is which client a sign-in came from, from the
// X-Spotto-Client header the app and both web apps send. Anything else
// (older app builds included) counts as the app.
func sessionClient(r *http.Request) string {
	switch c := r.Header.Get("X-Spotto-Client"); c {
	case "admin", "vendor":
		return c
	}
	return "app"
}

// deviceLabel is a short human description of the signing-in device, shown
// to admins on the Users page.
func deviceLabel(r *http.Request) string {
	label := strings.TrimSpace(r.Header.Get("X-Spotto-Device"))
	if label == "" {
		label = strings.TrimSpace(r.UserAgent())
	}
	return sanitizeText(label, 120)
}

// startSession signs u in on this device: it records a user_sessions row and
// returns a short-lived access token plus the refresh token that keeps the
// device signed in.
func (s *Server) startSession(r *http.Request, u domain.User) (access, refresh string, err error) {
	refresh, hash, err := auth.NewRefreshToken()
	if err != nil {
		return "", "", err
	}
	client := sessionClient(r)
	sess := domain.Session{
		ID:          newID("ses"),
		UserID:      u.ID,
		Client:      client,
		DeviceLabel: deviceLabel(r),
		ExpiresAt:   time.Now().Add(auth.SessionIdleLimit(client)),
	}
	if err := s.Store.CreateSession(r.Context(), sess, hash); err != nil {
		return "", "", err
	}
	access, err = s.Tokens.IssueForSession(u, sess.ID)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

type refreshReq struct {
	RefreshToken string `json:"refreshToken"`
}

// handleRefresh trades a refresh token for a new access token and a new
// refresh token; the old refresh token stops working.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.RefreshToken == "" {
		writeErr(w, http.StatusBadRequest, "refreshToken required")
		return
	}
	ctx := r.Context()
	hash := auth.HashRefreshToken(req.RefreshToken)
	sess, err := s.Store.SessionByRefreshHash(ctx, hash)
	if errors.Is(err, postgres.ErrNotFound) {
		s.rejectReplacedRefresh(w, r, hash)
		return
	}
	if err != nil {
		log.Printf("refresh: session lookup: %v", err)
		writeErr(w, http.StatusInternalServerError, "Couldn't refresh your session. Please try again.")
		return
	}
	if sess.RevokedAt != nil || time.Now().After(sess.ExpiresAt) {
		writeErr(w, http.StatusUnauthorized, "session ended, please sign in again")
		return
	}
	u, err := s.Store.UserByID(ctx, sess.UserID)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "session ended, please sign in again")
		return
	}
	newRefresh, newHash, err := auth.NewRefreshToken()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Couldn't refresh your session. Please try again.")
		return
	}
	ok, err := s.Store.RotateSession(ctx, sess.ID, hash, newHash, time.Now().Add(auth.SessionIdleLimit(sess.Client)))
	if err != nil {
		log.Printf("refresh: rotate session=%s: %v", sess.ID, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't refresh your session. Please try again.")
		return
	}
	if !ok {
		// Another request rotated (or ended) it between the lookup and now.
		writeErr(w, http.StatusUnauthorized, "refresh token already used")
		return
	}
	access, err := s.Tokens.IssueForSession(u, sess.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "Couldn't refresh your session. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": access, "refreshToken": newRefresh})
}

// rejectReplacedRefresh answers a refresh token that isn't any session's
// current one. If it's one a session recently replaced, that's two requests
// racing; if it was replaced longer ago, someone is replaying an old token,
// so that session is ended.
func (s *Server) rejectReplacedRefresh(w http.ResponseWriter, r *http.Request, hash string) {
	sess, err := s.Store.SessionByPreviousHash(r.Context(), hash)
	if err == nil && sess.RevokedAt == nil && (sess.RotatedAt == nil || time.Since(*sess.RotatedAt) > refreshReuseGrace) {
		if err := s.Store.RevokeSession(r.Context(), sess.ID); err != nil {
			log.Printf("refresh: revoke replayed session=%s: %v", sess.ID, err)
		} else {
			log.Printf("refresh: old refresh token replayed, ended session=%s user=%s", sess.ID, sess.UserID)
		}
	}
	writeErr(w, http.StatusUnauthorized, "refresh token already used")
}

// handleLogout ends the session a refresh token belongs to. Always succeeds,
// so signing out never gets stuck on a bad or already-ended token.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.RefreshToken != "" {
		sess, err := s.Store.SessionByRefreshHash(r.Context(), auth.HashRefreshToken(req.RefreshToken))
		if err == nil {
			if err := s.Store.RevokeSession(r.Context(), sess.ID); err != nil {
				log.Printf("logout: revoke session=%s: %v", sess.ID, err)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleSessionUpgrade gives a device still holding an access token from
// before sessions existed a refresh token, so this deploy doesn't sign
// anyone out.
func (s *Server) handleSessionUpgrade(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	if c.SessionID != "" {
		writeErr(w, http.StatusConflict, "this sign-in already has a session")
		return
	}
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	access, refresh, err := s.startSession(r, u)
	if err != nil {
		log.Printf("session upgrade user=%s: %v", u.ID, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't start a session. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": access, "refreshToken": refresh})
}

// keepOnlyThisSession ends every other session after a credential change
// (password change, 2FA disable) and re-issues this device's access token
// for its bumped token_version. Devices signed in before sessions existed
// have no session to keep, so they get a plain token and all sessions end.
func (s *Server) keepOnlyThisSession(r *http.Request, u domain.User) (string, error) {
	c := auth.FromContext(r.Context())
	if _, err := s.Store.RevokeOtherUserSessions(r.Context(), u.ID, c.SessionID); err != nil {
		return "", err
	}
	return s.Tokens.IssueForSession(u, c.SessionID)
}
