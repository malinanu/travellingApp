package httpapi

import (
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// dummyPasswordHash is compared against when a login's email doesn't match
// any user, so CheckPassword always pays its bcrypt cost either way —
// otherwise response latency alone reveals which emails are registered
// (the "||" short-circuit skips bcrypt entirely for unknown emails).
// Computed once at startup via package-var init, not per request.
var dummyPasswordHash = mustHashDummyPassword()

func mustHashDummyPassword() string {
	h, err := auth.HashPassword("not-a-real-password-used-only-for-constant-time-login-checks")
	if err != nil {
		panic(err)
	}
	return h
}

// commonPasswords are the handful of weak passwords that show up at the top
// of every breach-corpus frequency list — cheap to block outright even
// though they'd pass the other structural checks below (e.g. "password1"
// isn't all-digit or all-repeated-char).
var commonPasswords = map[string]bool{
	"password": true, "password1": true, "password123": true,
	"12345678": true, "123456789": true, "1234567890": true,
	"qwertyui": true, "qwertyuiop": true, "letmein11": true,
	"iloveyou1": true, "welcome123": true, "admin1234": true,
	"abc12345": true, "changeme1": true, "trustno1x": true,
}

// isWeakPassword catches the most common trivially-guessable patterns
// (length is already checked separately by the caller). Not a full
// strength-scoring system — just cheap, high-value rejections.
func isWeakPassword(password, email string) bool {
	allDigits := true
	allSameChar := true
	for i, r := range password {
		if !unicode.IsDigit(r) {
			allDigits = false
		}
		if i > 0 && r != rune(password[0]) {
			allSameChar = false
		}
	}
	if allDigits || allSameChar {
		return true
	}
	lower := strings.ToLower(password)
	if commonPasswords[lower] {
		return true
	}
	localPart, _, _ := strings.Cut(strings.ToLower(email), "@")
	return lower == strings.ToLower(email) || (localPart != "" && lower == localPart)
}

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"` // traveller (default) or vendor
	// Vendor registration extras
	BusinessName string `json:"businessName,omitempty"`
	Category     string `json:"category,omitempty"`
	Town         string `json:"town,omitempty"`
	// Traveller extras
	Country     string   `json:"country,omitempty"`
	TravelStyle string   `json:"travelStyle,omitempty"`
	Interests   []string `json:"interests,omitempty"`
}

type authResp struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if !readJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || len(req.Password) < 10 {
		writeErr(w, http.StatusBadRequest, "email and a password of at least 10 characters are required")
		return
	}
	if isWeakPassword(req.Password, req.Email) {
		writeErr(w, http.StatusBadRequest, "password is too weak — avoid all-digit, all-repeated, or email-derived passwords")
		return
	}
	if _, err := s.Store.UserByEmail(r.Context(), req.Email); err == nil {
		writeErr(w, http.StatusConflict, "email already registered")
		return
	}

	role := domain.RoleTraveller
	if req.Role == string(domain.RoleVendor) {
		role = domain.RoleVendor
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}

	u := domain.User{
		ID:           newID("usr"),
		Email:        req.Email,
		PasswordHash: hash,
		Role:         role,
		Name:         req.Name,
		Country:      req.Country,
		TravelStyle:  req.TravelStyle,
		Interests:    req.Interests,
		CreatedAt:    time.Now(),
	}

	// Vendor registration creates a pending vendor gated until admin approval.
	if role == domain.RoleVendor {
		v := domain.Vendor{
			ID:          newID("ven"),
			Name:        req.BusinessName,
			Category:    req.Category,
			Town:        req.Town,
			Owner:       req.Name,
			Status:      domain.VendorPending,
			Priority:    "normal",
			Flags:       []string{},
			Tier:        "Free",
			SubmittedAt: time.Now(),
		}
		if v.Name == "" {
			v.Name = req.Name
		}
		if err := s.Store.UpsertVendor(r.Context(), v); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		u.VendorID = &v.ID
	}

	if err := s.Store.CreateUser(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, authResp{Token: token, User: u})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	limitKey := clientIP(r) + ":" + email
	if !s.loginLimit.allow(limitKey) {
		writeTooManyRequests(w)
		return
	}
	u, err := s.Store.UserByEmail(r.Context(), email)
	hash := dummyPasswordHash
	if err == nil {
		hash = u.PasswordHash
	}
	passwordOK := auth.CheckPassword(hash, req.Password)
	if err != nil || !passwordOK {
		s.loginLimit.recordFailure(limitKey)
		writeErr(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	s.loginLimit.reset(limitKey)
	if u.TOTPEnabled {
		pending, err := s.Tokens.IssueTwoFactorPending(u.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"twoFactorRequired": true,
			"twoFactorToken":    pending,
		})
		return
	}
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, authResp{Token: token, User: u})
}

// meResp includes vendor status so vendor clients can show the pending gate.
type meResp struct {
	domain.User
	VendorStatus *domain.VendorStatus `json:"vendorStatus,omitempty"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	resp := meResp{User: u}
	if u.VendorID != nil {
		if v, err := s.Store.VendorByID(r.Context(), *u.VendorID); err == nil {
			resp.VendorStatus = &v.Status
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Name        *string   `json:"name"`
		Country     *string   `json:"country"`
		TravelStyle *string   `json:"travelStyle"`
		Interests   *[]string `json:"interests"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Country != nil {
		u.Country = *req.Country
	}
	if req.TravelStyle != nil {
		u.TravelStyle = *req.TravelStyle
	}
	if req.Interests != nil {
		u.Interests = *req.Interests
	}
	if err := s.Store.UpdateUserProfile(r.Context(), u.ID, u.Name, u.Country, u.TravelStyle, u.Interests); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// handleChangePassword requires proof of the current password, then applies
// the same strength rules as registration to the new one. Bumps
// token_version (see Store.UpdateUserPassword) so other logged-in sessions
// are signed out.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if !auth.CheckPassword(u.PasswordHash, req.CurrentPassword) {
		writeErr(w, http.StatusUnauthorized, "current password is incorrect")
		return
	}
	if len(req.NewPassword) < 10 {
		writeErr(w, http.StatusBadRequest, "new password must be at least 10 characters")
		return
	}
	if isWeakPassword(req.NewPassword, u.Email) {
		writeErr(w, http.StatusBadRequest, "password is too weak — avoid all-digit, all-repeated, or email-derived passwords")
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.Store.UpdateUserPassword(r.Context(), u.ID, hash); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	// UpdateUserPassword just bumped token_version in the DB — mirror that
	// locally so the token we issue below matches, instead of immediately
	// failing Server.Authenticated's version check on the very next request.
	u.TokenVersion++
	// Re-issue a token for THIS session (the one that just proved current
	// possession) since its old token_version no longer matches.
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// handleUpdatePrivacy toggles the traveller's nearby-travellers consent.
// Turning it off clears any previously reported position (see
// Store.SetShareLocation) rather than leaving it stale and merely unused.
func (s *Server) handleUpdatePrivacy(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		ShareLocation bool `json:"shareLocation"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.SetShareLocation(r.Context(), c.Subject, req.ShareLocation); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"shareLocation": req.ShareLocation})
}

// requireVendor loads the caller's vendor, or writes an error and returns false.
func (s *Server) requireVendor(w http.ResponseWriter, r *http.Request) (domain.Vendor, bool) {
	c := auth.FromContext(r.Context())
	if c.VendorID == "" {
		writeErr(w, http.StatusForbidden, "no vendor account")
		return domain.Vendor{}, false
	}
	v, err := s.Store.VendorByID(r.Context(), c.VendorID)
	if err != nil {
		if err == postgres.ErrNotFound {
			writeErr(w, http.StatusForbidden, "no vendor account")
		} else {
			storeErr(w, err)
		}
		return domain.Vendor{}, false
	}
	return v, true
}

// requireApprovedVendor additionally gates on admin approval.
func (s *Server) requireApprovedVendor(w http.ResponseWriter, r *http.Request) (domain.Vendor, bool) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return v, false
	}
	if v.Status != domain.VendorApproved {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":        "vendor_not_approved",
			"vendorStatus": v.Status,
		})
		return v, false
	}
	return v, true
}
