package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spotto/server/internal/domain"
)

// handleGoogleLogin redirects the browser to Google's consent screen.
// 501s if Google OAuth isn't configured (no GOOGLE_CLIENT_ID/SECRET) so
// the frontend can hide the button instead of it silently failing.
func (s *Server) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if s.GoogleOAuth == nil {
		writeErr(w, http.StatusNotImplemented, "google sign-in is not configured")
		return
	}
	state := newID("st")
	// Stateless CSRF guard: state round-trips through Google and back to
	// us; a short-lived cookie lets us confirm it's the same browser that
	// started the flow without server-side session storage for it.
	http.SetCookie(w, &http.Cookie{
		Name: "google_oauth_state", Value: state, Path: "/", MaxAge: 300,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, s.GoogleOAuth.AuthCodeURL(state), http.StatusFound)
}

// handleGoogleCallback exchanges the auth code, looks up a matching
// vendor account by Google ID (or, first time, by email), and hands the
// frontend a one-time authCode to exchange for the same JSON body
// /auth/login would return (including the 2FA-pending shape, if the
// matched account has 2FA enabled â€” Google sign-in does not bypass it).
// If no account matches, it redirects to registration with the Google
// name/email prefilled instead of auto-creating a vendor, since
// registration still needs business type/town Google can't supply.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if s.GoogleOAuth == nil {
		writeErr(w, http.StatusNotImplemented, "google sign-in is not configured")
		return
	}
	cookie, err := r.Cookie("google_oauth_state")
	if err != nil || r.URL.Query().Get("state") == "" || r.URL.Query().Get("state") != cookie.Value {
		writeErr(w, http.StatusBadRequest, "invalid oauth state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		writeErr(w, http.StatusBadRequest, "missing code")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	tok, err := s.GoogleOAuth.Exchange(ctx, code)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "google sign-in failed")
		return
	}
	info, err := fetchGoogleUserInfo(ctx, s.GoogleOAuth.Client(ctx, tok))
	if err != nil || info.Email == "" {
		writeErr(w, http.StatusUnauthorized, "google sign-in failed")
		return
	}

	u, err := s.Store.UserByGoogleID(ctx, info.Sub)
	if err != nil {
		u, err = s.Store.UserByEmail(ctx, strings.ToLower(info.Email))
	}
	if err != nil || u.Role != domain.RoleVendor {
		// No matching vendor account â€” send them to registration with
		// their Google details prefilled rather than guessing.
		q := url.Values{"email": {info.Email}, "name": {info.Name}}
		http.Redirect(w, r, s.VendorAppURL+"/login?mode=register&"+q.Encode(), http.StatusFound)
		return
	}
	if u.GoogleID == nil {
		s.Store.LinkUserGoogleID(ctx, u.ID, info.Sub)
	}

	var payload any
	if u.TOTPEnabled {
		pending, err := s.Tokens.IssueTwoFactorPending(u.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
			return
		}
		payload = map[string]any{"twoFactorRequired": true, "twoFactorToken": pending}
	} else {
		token, refresh, err := s.startSession(r, u)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
			return
		}
		payload = authResp{Token: token, RefreshToken: refresh, User: u}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	exchangeCode := newID("oac")
	s.oauthCodes.put(exchangeCode, b)
	http.Redirect(w, r, s.VendorAppURL+"/login?authCode="+exchangeCode, http.StatusFound)
}

// handleGoogleMobileAuth signs a traveller in from the mobile app's Google
// Sign-In. Only the signed Google ID token is trusted: id and email come from
// the verified token, never from the request body. Like Apple sign-in it
// skips password and 2FA, so it only opens traveller accounts.
func (s *Server) handleGoogleMobileAuth(w http.ResponseWriter, r *http.Request) {
	if s.GoogleMobileAuth == nil {
		writeErr(w, http.StatusNotImplemented, "google sign-in is not configured")
		return
	}
	var req struct {
		IdentityToken string `json:"idToken"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.IdentityToken == "" {
		writeErr(w, http.StatusBadRequest, "idToken required")
		return
	}
	identity, err := s.GoogleMobileAuth.Verify(r.Context(), req.IdentityToken)
	if err != nil || !identity.EmailVerified || identity.Email == "" {
		writeErr(w, http.StatusUnauthorized, "invalid Google sign-in")
		return
	}
	email := strings.ToLower(strings.TrimSpace(identity.Email))

	u, err := s.Store.UserByGoogleID(r.Context(), identity.Subject)
	status := http.StatusOK
	if err != nil {
		if existing, emailErr := s.Store.UserByEmail(r.Context(), email); emailErr == nil {
			u = existing
			if u.Role == domain.RoleTraveller {
				_ = s.Store.LinkUserGoogleID(r.Context(), u.ID, identity.Subject)
			}
		} else {
			name := sanitizeText(identity.Name, 100)
			if name == "" {
				name = "Google User"
			}
			sub := identity.Subject
			newUser := domain.User{
				ID:        newID("usr"),
				Email:     email,
				Name:      name,
				Role:      domain.RoleTraveller,
				GoogleID:  &sub,
				CreatedAt: time.Now(),
			}
			if err := s.Store.CreateUser(r.Context(), newUser); err != nil {
				writeErr(w, http.StatusInternalServerError, "could not create user account")
				return
			}
			_ = s.Store.LinkUserGoogleID(r.Context(), newUser.ID, sub)
			u, err = s.Store.UserByID(r.Context(), newUser.ID)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
				return
			}
			status = http.StatusCreated
		}
	}
	if u.Role != domain.RoleTraveller {
		writeErr(w, http.StatusForbidden, "this account must sign in with email and password")
		return
	}
	token, refresh, err := s.startSession(r, u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	writeJSON(w, status, map[string]any{"token": token, "refreshToken": refresh, "user": u})
}

// handleAuthExchange trades a one-time authCode (minted by
// handleGoogleCallback) for the login payload it was holding. Codes are
// single-use and expire in 2 minutes.
func (s *Server) handleAuthExchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	payload, ok := s.oauthCodes.take(req.Code)
	if !ok {
		writeErr(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(payload)
}

type googleUserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func fetchGoogleUserInfo(ctx context.Context, client *http.Client) (googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return googleUserInfo{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return googleUserInfo{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return googleUserInfo{}, err
	}
	var info googleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return googleUserInfo{}, err
	}
	return info, nil
}
