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
	u, err := s.Store.UserByID(r.Context(), userID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if !auth.ValidateTOTP(u.TOTPSecret, req.Code) {
		writeErr(w, http.StatusUnauthorized, "invalid code")
		return
	}
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
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
	if err := s.Store.SetUserTOTPPending(r.Context(), u.ID, secret); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
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
	if !auth.ValidateTOTP(u.TOTPPendingSecret, req.Code) {
		writeErr(w, http.StatusUnauthorized, "invalid code")
		return
	}
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
	if !u.TOTPEnabled || !auth.ValidateTOTP(u.TOTPSecret, req.Code) {
		writeErr(w, http.StatusUnauthorized, "invalid code")
		return
	}
	if err := s.Store.DisableUserTOTP(r.Context(), u.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": false})
}
