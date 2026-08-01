package httpapi

import (
	"net/http"

	"github.com/spotto/server/internal/auth"
)

// handleTwoFactorLoginVerify completes a login that returned
// {twoFactorRequired: true}: verifies the code against the pending user's
// active secret and, on success, issues a normal session token.
func (s *Server) handleTwoFactorLoginVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TwoFactorToken string `json:"twoFactorToken"`
		Code           string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	userID, err := s.Tokens.VerifyTwoFactorPending(req.TwoFactorToken)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid or expired login")
		return
	}
	limitKey := clientIP(r) + ":" + userID
	if !s.totpLimit.allow(limitKey) {
		writeTooManyRequests(w)
		return
	}
	u, err := s.Store.UserByID(r.Context(), userID)
	if err != nil {
		storeErr(w, err)
		return
	}
	secret, err := auth.DecryptSecret(s.TOTPKey, u.TOTPSecret)
	if err != nil || !auth.ValidateTOTP(secret, req.Code) {
		s.totpLimit.recordFailure(limitKey)
		writeErr(w, http.StatusUnauthorized, "invalid code")
		return
	}
	s.totpLimit.reset(limitKey)
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
	writeJSON(w, http.StatusOK, authResp{Token: token, User: u})
}

// handleTwoFactorSetup generates a new pending secret + otpauth:// URL for
// QR rendering. Not active until confirmed via handleTwoFactorEnable.
func (s *Server) handleTwoFactorSetup(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	secret, otpauthURL, err := auth.GenerateTOTPSecret(u.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	encrypted, err := auth.EncryptSecret(s.TOTPKey, secret)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.Store.SetUserTOTPPending(r.Context(), u.ID, encrypted); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	// The client only ever sees the plaintext secret/otpauthUrl for QR
	// rendering at setup time — nothing plaintext is ever persisted.
	writeJSON(w, http.StatusOK, map[string]string{"secret": secret, "otpauthUrl": otpauthURL})
}

// handleTwoFactorEnable confirms setup: verifies a code against the pending
// secret and promotes it to active.
func (s *Server) handleTwoFactorEnable(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if u.TOTPPendingSecret == "" {
		writeErr(w, http.StatusBadRequest, "no pending 2FA setup — call /auth/2fa/setup first")
		return
	}
	pendingSecret, err := auth.DecryptSecret(s.TOTPKey, u.TOTPPendingSecret)
	if err != nil || !auth.ValidateTOTP(pendingSecret, req.Code) {
		writeErr(w, http.StatusUnauthorized, "invalid code")
		return
	}
	// u.TOTPPendingSecret is already encrypted — EnableUserTOTP just moves
	// it from the pending column to the active one, no re-encryption needed.
	if err := s.Store.EnableUserTOTP(r.Context(), u.ID, u.TOTPPendingSecret); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": true})
}

// handleTwoFactorDisable requires the current code as proof of possession.
func (s *Server) handleTwoFactorDisable(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	limitKey := clientIP(r) + ":" + u.ID
	if !s.totpLimit.allow(limitKey) {
		writeTooManyRequests(w)
		return
	}
	secret, err := auth.DecryptSecret(s.TOTPKey, u.TOTPSecret)
	if !u.TOTPEnabled || err != nil || !auth.ValidateTOTP(secret, req.Code) {
		s.totpLimit.recordFailure(limitKey)
		writeErr(w, http.StatusUnauthorized, "invalid code")
		return
	}
	s.totpLimit.reset(limitKey)
	if err := s.Store.DisableUserTOTP(r.Context(), u.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	// DisableUserTOTP just bumped token_version — mirror that locally and
	// re-issue a token so this same session (which just proved possession
	// via a valid code) isn't immediately logged out by its own action.
	u.TOTPEnabled = false
	u.TokenVersion++
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": false, "token": token})
}
