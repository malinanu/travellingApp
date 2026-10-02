package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// dummyPasswordHash is compared against when a login's email doesn't match
// any user, so CheckPassword always pays its bcrypt cost either way â€”
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
// of every breach-corpus frequency list â€” cheap to block outright even
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
// strength-scoring system â€” just cheap, high-value rejections.
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
	Phone        string `json:"phone,omitempty"`
	Description  string `json:"description,omitempty"`
	// Lat/Lng/Contact/Hours are populated only when the owner found their
	// business via the registration wizard's Google Places search step
	// (web/vendor/app/login/page.tsx) rather than typing everything by
	// hand â€” carried directly in this request because a freshly-registered
	// vendor is `pending` and can't yet call the approval-gated
	// PUT /vendor/business / PATCH /vendor/business/identity endpoints to
	// set these itself.
	Lat     float64 `json:"lat,omitempty"`
	Lng     float64 `json:"lng,omitempty"`
	Contact *struct {
		Phone   string `json:"phone"`
		Website string `json:"website"`
	} `json:"contact,omitempty"`
	Hours map[string]dayHoursDoc `json:"hours,omitempty"`
	// BusinessCountry is an ISO 3166-1 alpha-2 code (e.g. "LK", "US") read
	// from the Places search result's address_components â€” distinct from
	// the traveller-registration Country field below (that one is the
	// traveller's own home country, an unrelated concept). Empty defaults
	// to "LK" in Store.UpsertVendor, matching every vendor that predates
	// this field.
	BusinessCountry string `json:"businessCountry,omitempty"`
	// GooglePlaceID is the Google Maps Place ID from the registration
	// wizard's "Find your business" Places search â€” captured so we can
	// cache it on the vendor row for later Place Details enrichment (see
	// internal/placedetails) instead of only having it in the JSONB blob.
	GooglePlaceID string `json:"googlePlaceId,omitempty"`
	// InviteCode joins an existing vendor as the invite's assigned role
	// (staff or manager) instead of creating a new pending vendor â€” see
	// the team-invite flow in handleRegister.
	InviteCode string `json:"inviteCode,omitempty"`
	// Traveller extras
	Country     string   `json:"country,omitempty"`
	TravelStyle []string `json:"travelStyle,omitempty"`
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
	req.Name = sanitizeText(req.Name, 100)
	if req.Email == "" || len(req.Password) < 10 {
		writeErr(w, http.StatusBadRequest, "email and a password of at least 10 characters are required")
		return
	}
	if !validateTextLength(w, "Email", req.Email, 254) || !validateTextLength(w, "Password", req.Password, 128) || !validateTextLength(w, "Name", req.Name, 100) {
		return
	}
	if isWeakPassword(req.Password, req.Email) {
		writeErr(w, http.StatusBadRequest, "password is too weak â€” avoid all-digit, all-repeated, or email-derived passwords")
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
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
		Phone:        req.Phone,
		CreatedAt:    time.Now(),
	}

	var invite *domain.VendorInvite
	if role == domain.RoleVendor && req.InviteCode != "" {
		// Joining an existing vendor as whatever role the invite specified
		// (staff or manager) â€” validated below, after we know the invite
		// exists, isn't revoked/accepted/expired.
		inv, err := s.Store.VendorInviteByCode(r.Context(), req.InviteCode)
		if err != nil || inv.Revoked || inv.AcceptedAt != nil || time.Now().After(inv.ExpiresAt) {
			writeErr(w, http.StatusBadRequest, "invalid or expired invite code")
			return
		}
		invite = &inv
		u.VendorID = &inv.VendorID
		u.VendorRole = inv.Role
		if u.VendorRole == "" {
			// Invites created before the role column existed default to staff.
			u.VendorRole = domain.VendorRoleStaff
		}
	} else if role == domain.RoleVendor {
		if !domain.ValidCategoryID(req.Category) {
			writeErr(w, http.StatusBadRequest, "unknown business category")
			return
		}
		// Vendor registration creates a pending vendor gated until admin approval.
		var googlePlaceID *string
		if req.GooglePlaceID != "" {
			googlePlaceID = &req.GooglePlaceID
		}
		v := domain.Vendor{
			ID:            newID("ven"),
			Name:          req.BusinessName,
			Category:      req.Category,
			Town:          req.Town,
			Owner:         req.Name,
			Status:        domain.VendorPending,
			Priority:      "normal",
			Flags:         []string{},
			Tier:          "Free",
			SubmittedAt:   time.Now(),
			Lat:           req.Lat,
			Lng:           req.Lng,
			Country:       req.BusinessCountry,
			GooglePlaceID: googlePlaceID,
		}
		if v.Name == "" {
			v.Name = req.Name
		}
		profile := map[string]any{}
		if req.Description != "" {
			profile["blurb"] = req.Description
		}
		if req.Contact != nil && (req.Contact.Phone != "" || req.Contact.Website != "") {
			profile["contact"] = map[string]any{"phone": req.Contact.Phone, "website": req.Contact.Website}
		}
		if len(req.Hours) > 0 {
			profile["hours"] = req.Hours
		}
		if len(profile) > 0 {
			if b, err := json.Marshal(profile); err == nil {
				v.Profile = b
			}
		}
		if err := s.Store.UpsertVendor(r.Context(), v); err != nil {
			writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
			return
		}
		u.VendorID = &v.ID
		u.VendorRole = domain.VendorRoleOwner
	}

	if err := s.Store.CreateUser(r.Context(), u); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if invite != nil {
		if err := s.Store.AcceptVendorInvite(r.Context(), invite.ID, u.ID); err != nil {
			writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
			return
		}
	}
	// Re-fetch rather than reusing the in-memory u: several columns (the
	// notification-preference defaults in particular) are only set by the
	// database's own DEFAULT, not by this handler, so the local struct's
	// Go zero values (false) would otherwise be echoed back instead of
	// what was actually just stored.
	u, err = s.Store.UserByID(r.Context(), u.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
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
			writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
	writeJSON(w, http.StatusOK, authResp{Token: token, User: u})
}

// meResp includes vendor status so vendor clients can show the pending gate.
type meResp struct {
	domain.User
	VendorStatus *domain.VendorStatus `json:"vendorStatus,omitempty"`
	// IsSuperAdmin is only ever true for RoleAdmin accounts listed in
	// SUPER_ADMIN_EMAILS â€” see superadmin.go.
	IsSuperAdmin bool `json:"isSuperAdmin,omitempty"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
	resp := meResp{User: u}
	if u.VendorID != nil {
		if v, err := s.Store.VendorByID(r.Context(), *u.VendorID); err == nil {
			resp.VendorStatus = &v.Status
		}
	}
	resp.IsSuperAdmin = u.Role == domain.RoleAdmin && s.isSuperAdmin(u.Email)
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
		TravelStyle *[]string `json:"travelStyle"`
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	u.AvatarURL = s.signMediaPath(u.AvatarURL)
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
		writeErr(w, http.StatusBadRequest, "password is too weak â€” avoid all-digit, all-repeated, or email-derived passwords")
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if err := s.Store.UpdateUserPassword(r.Context(), u.ID, hash); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	// UpdateUserPassword just bumped token_version in the DB â€” mirror that
	// locally so the token we issue below matches, instead of immediately
	// failing Server.Authenticated's version check on the very next request.
	u.TokenVersion++
	// Re-issue a token for THIS session (the one that just proved current
	// possession) since its old token_version no longer matches.
	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
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
		PublicProfile bool `json:"publicProfile"`
		ShowActivity  bool `json:"showActivity"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.SetShareLocation(r.Context(), c.Subject, req.ShareLocation); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

// handleUpdateMyContact updates the account page's editable identity
// fields (name/email/phone). Email doubles as the login credential, so a
// change here is checked for uniqueness just like registration.
func (s *Server) handleUpdateMyContact(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		writeErr(w, http.StatusBadRequest, "email is required")
		return
	}
	if email != u.Email {
		if _, err := s.Store.UserByEmail(r.Context(), email); err == nil {
			writeErr(w, http.StatusConflict, "email already registered")
			return
		}
	}
	if err := s.Store.UpdateUserContact(r.Context(), u.ID, req.Name, email, req.Phone); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	u.Name, u.Email, u.Phone = req.Name, email, req.Phone
	writeJSON(w, http.StatusOK, u)
}

// handleUpdateNotificationPrefs saves the account page's notification
// toggles. Nothing in the app sends notifications yet â€” this just persists
// the preference for whenever that exists.
func (s *Server) handleUpdateNotificationPrefs(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		Redemptions bool `json:"notifyRedemptions"`
		Reviews     bool `json:"notifyReviews"`
		Payouts     bool `json:"notifyPayouts"`
		GrowthTips  bool `json:"notifyGrowthTips"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.UpdateNotificationPrefs(r.Context(), c.Subject, req.Redemptions, req.Reviews, req.Payouts, req.GrowthTips); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, req)
}

func newResetToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashResetToken(raw string) string { return sha256Hex(raw) }

// sha256Hex hashes a secret (reset token, profile-grant PIN, ...) for
// at-rest storage â€” callers compare by hash, never store the raw value.
func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// handleForgotPassword always responds 200 regardless of whether the
// email matches an account, to avoid account enumeration via response
// differences. If it does match, a reset email is queued (see
// internal/jobs) rather than sent inline, so this handler doesn't block
// on SMTP latency. The emailed link's host is picked by the matched
// user's role (VendorAppURL vs AdminAppURL) â€” travellers don't have a
// reset UI yet, so they fall back to the vendor app's like before.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	limitKey := clientIP(r) + ":" + email
	if !s.resetLimit.allow(limitKey) {
		writeTooManyRequests(w)
		return
	}
	s.resetLimit.recordFailure(limitKey) // bounds requests per key; never reset â€” resends should stay rate-limited too

	if u, err := s.Store.UserByEmail(r.Context(), email); err == nil {
		raw := newResetToken()
		if err := s.Store.CreatePasswordResetToken(r.Context(), newID("prt"), u.ID, hashResetToken(raw), time.Now().Add(1*time.Hour)); err == nil {
			appURL := s.VendorAppURL
			if u.Role == domain.RoleAdmin {
				appURL = s.AdminAppURL
			}
			link := appURL + "/login?resetToken=" + raw
			to, body := u.Email, resetEmailHTML(link)
			s.Jobs.Enqueue(func() { s.Mail.Send(to, "Reset your Spotto password", body) })
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func resetEmailHTML(link string) string {
	return `<p>Someone requested a password reset for your Spotto account.</p>` +
		`<p><a href="` + link + `">Reset your password</a> (expires in 1 hour).</p>` +
		`<p>If this wasn't you, you can safely ignore this email.</p>`
}

// handleResetPassword validates the token (by hash â€” the raw token is
// never stored), applies the same password-strength rules as
// registration, and bumps token_version so any existing sessions are
// signed out (same revocation mechanism as a 2FA disable).
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Token == "" {
		writeErr(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}
	rt, err := s.Store.PasswordResetTokenByHash(r.Context(), hashResetToken(req.Token))
	if err != nil || rt.UsedAt != nil || time.Now().After(rt.ExpiresAt) {
		writeErr(w, http.StatusBadRequest, "invalid or expired reset link")
		return
	}
	u, err := s.Store.UserByID(r.Context(), rt.UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if len(req.NewPassword) < 10 {
		writeErr(w, http.StatusBadRequest, "new password must be at least 10 characters")
		return
	}
	if isWeakPassword(req.NewPassword, u.Email) {
		writeErr(w, http.StatusBadRequest, "password is too weak â€” avoid all-digit, all-repeated, or email-derived passwords")
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if err := s.Store.UpdateUserPassword(r.Context(), u.ID, hash); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if err := s.Store.MarkPasswordResetTokenUsed(r.Context(), rt.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

// requireApprovedVendor additionally gates on admin approval. Use for
// anything that depends on the business actually being live to travelers
// (offers, growth tools, redemption validation, reports).
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

// requireEditableVendor allows pending/edits-status vendors through in
// addition to approved ones, so they can set up their business profile,
// category, and menu while awaiting admin approval â€” only rejected/
// suspended vendors are blocked. Use for writes that just describe the
// business rather than making it live to travelers.
func (s *Server) requireEditableVendor(w http.ResponseWriter, r *http.Request) (domain.Vendor, bool) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return v, false
	}
	if v.Status == domain.VendorRejected || v.Status == domain.VendorSuspended {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"error":        "vendor_not_approved",
			"vendorStatus": v.Status,
		})
		return v, false
	}
	return v, true
}

// handleDeleteAccount removes the user account and associated personal data.
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	if err := s.Store.DeleteUser(r.Context(), c.Subject); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// handleExportData returns an archive of user profile data and activity.
func (s *Server) handleExportData(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user":        u,
		"exported_at": time.Now().UTC(),
	})
}

// handleAppleAuth handles Sign in with Apple logins & registrations.
func (s *Server) handleAppleAuth(w http.ResponseWriter, r *http.Request) {
	// Only the signed identityToken is trusted. The client-supplied appleId
	// and email are deliberately not read: anyone can send those, and
	// trusting them let a caller sign in as any existing account.
	var req struct {
		Name          string `json:"name"`
		IdentityToken string `json:"identityToken"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.IdentityToken == "" {
		writeErr(w, http.StatusBadRequest, "identityToken required")
		return
	}
	identity, err := s.AppleAuth.Verify(r.Context(), req.IdentityToken)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid Apple sign-in")
		return
	}
	appleID := identity.Subject
	verifiedEmail := ""
	if identity.EmailVerified {
		verifiedEmail = strings.ToLower(strings.TrimSpace(identity.Email))
	}
	// Apple sign-in skips password and 2FA, so it may only open traveller
	// accounts; vendor/admin accounts must use the normal login flow.
	travellerOnly := func(u domain.User) bool {
		if u.Role != domain.RoleTraveller {
			writeErr(w, http.StatusForbidden, "this account must sign in with email and password")
			return false
		}
		return true
	}

	// 1. Check if user already exists by appleId
	u, err := s.Store.GetUserByAppleID(r.Context(), appleID)
	if err == nil {
		if !travellerOnly(u) {
			return
		}
		token, err := s.Tokens.Issue(u)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "token generation failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
		return
	}

	// 2. Check if user exists by email
	if verifiedEmail != "" {
		existing, err := s.Store.UserByEmail(r.Context(), verifiedEmail)
		if err == nil {
			if !travellerOnly(existing) {
				return
			}
			_ = s.Store.LinkUserAppleID(r.Context(), existing.ID, appleID)
			token, err := s.Tokens.Issue(existing)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "token generation failed")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": existing})
			return
		}
	}

	// 3. Register new user via Apple
	name := sanitizeText(req.Name, 100)
	if name == "" {
		name = "Apple User"
	}
	email := verifiedEmail
	if email == "" {
		email = fmt.Sprintf("apple_%s@privaterelay.appleid.com", appleID)
	}

	newUser := domain.User{
		ID:        newID("usr"),
		Email:     email,
		Name:      name,
		Role:      domain.RoleTraveller,
		AppleID:   &appleID,
		CreatedAt: time.Now(),
	}

	if err := s.Store.CreateUser(r.Context(), newUser); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not create user account")
		return
	}

	_ = s.Store.LinkUserAppleID(r.Context(), newUser.ID, appleID)

	u, err = s.Store.UserByID(r.Context(), newUser.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}

	token, err := s.Tokens.Issue(u)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": u})
}
