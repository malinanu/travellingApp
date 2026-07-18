package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// profilePinTTL bounds how long a generated PIN itself is redeemable —
// fixed and independent of the owner's chosen access duration, so a
// 15-minute grant doesn't silently burn down unread in an inbox.
const profilePinTTL = 10 * time.Minute

func newProfilePin() string {
	b := make([]byte, 6)
	rand.Read(b)
	out := make([]byte, 6)
	for i, v := range b {
		out[i] = byte('0') + v%10
	}
	return string(out)
}

// handleCreateVendorProfileGrant lets an owner generate a time-limited PIN
// giving a named manager temporary edit access to Business Profile, which
// managers otherwise have zero standing access to. Owner-only (see
// server.go's RequireVendorOwner).
func (s *Server) handleCreateVendorProfileGrant(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		GranteeID    string `json:"granteeId"`
		DurationMins int    `json:"durationMins"`
		Deliver      string `json:"deliver"` // "email" or "copy"
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.DurationMins != 15 && req.DurationMins != 30 && req.DurationMins != 60 {
		writeErr(w, http.StatusBadRequest, "durationMins must be 15, 30, or 60")
		return
	}
	members, err := s.Store.VendorTeamMembers(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	var grantee *domain.User
	for i := range members {
		if members[i].ID == req.GranteeID {
			grantee = &members[i]
			break
		}
	}
	if grantee == nil || grantee.VendorRole != domain.VendorRoleManager {
		writeErr(w, http.StatusBadRequest, "granteeId must be a manager on this vendor account")
		return
	}

	pin := newProfilePin()
	g := domain.VendorProfileGrant{
		ID:           newID("pgr"),
		VendorID:     v.ID,
		GranteeID:    grantee.ID,
		CreatedBy:    c.Subject,
		PinHash:      sha256Hex(pin),
		DurationMins: req.DurationMins,
		CreatedAt:    time.Now(),
		PinExpiresAt: time.Now().Add(profilePinTTL),
	}
	if err := s.Store.CreateVendorProfileGrant(r.Context(), g); err != nil {
		storeErr(w, err)
		return
	}
	if req.Deliver == "email" && grantee.Email != "" {
		email, subject, body := grantee.Email, "Your temporary Business Profile access PIN", profileGrantEmailHTML(v.Name, pin, req.DurationMins)
		s.Jobs.Enqueue(func() { s.Mail.Send(email, subject, body) })
	}
	// The raw PIN is returned once here regardless of delivery method (same
	// copy-fallback precedent as vendor invites) — it is never persisted.
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": g.ID, "granteeId": g.GranteeID, "durationMins": g.DurationMins,
		"pinExpiresAt": g.PinExpiresAt, "pin": pin,
	})
}

func profileGrantEmailHTML(vendorName, pin string, durationMins int) string {
	return fmt.Sprintf(
		`<p>You've been granted temporary Business Profile edit access for <strong>%s</strong> on Spotto.</p>
		<p>Your PIN: <strong style="font-size:20px;letter-spacing:2px">%s</strong></p>
		<p>Enter it on the Business Profile page within 10 minutes to unlock %d minutes of edit access.</p>`,
		vendorName, pin, durationMins)
}

// handleRedeemVendorProfileGrant lets the grantee unlock their access
// window by entering the PIN an owner shared with them. Rate-limited since
// a 6-digit PIN only has 10^6 possibilities.
func (s *Server) handleRedeemVendorProfileGrant(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	limitKey := clientIP(r) + ":" + c.Subject
	if !s.grantPinLimit.allow(limitKey) {
		writeTooManyRequests(w)
		return
	}
	var req struct {
		Pin string `json:"pin"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	g, err := s.Store.VendorProfileGrantByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.grantPinLimit.recordFailure(limitKey)
		storeErr(w, err)
		return
	}
	if g.GranteeID != c.Subject || g.RevokedAt != nil || g.RedeemedAt != nil || time.Now().After(g.PinExpiresAt) ||
		subtle.ConstantTimeCompare([]byte(sha256Hex(req.Pin)), []byte(g.PinHash)) != 1 {
		s.grantPinLimit.recordFailure(limitKey)
		writeErr(w, http.StatusForbidden, "invalid or expired PIN")
		return
	}
	s.grantPinLimit.reset(limitKey)
	if err := s.Store.RedeemVendorProfileGrant(r.Context(), g.ID); err != nil {
		storeErr(w, err)
		return
	}
	g, err = s.Store.VendorProfileGrantByID(r.Context(), g.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"accessExpiresAt": g.AccessExpiresAt})
}

// handleMyVendorProfileGrant returns the caller's latest non-revoked grant
// (pending-PIN or active), for the Profile page's countdown/PIN-entry UI.
func (s *Server) handleMyVendorProfileGrant(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	g, err := s.Store.LatestVendorProfileGrantForUser(r.Context(), v.ID, c.Subject)
	if err == postgres.ErrNotFound {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// handleListVendorProfileGrants lets an owner see every grant on their
// account (pending, active, expired, revoked) so they can revoke early.
func (s *Server) handleListVendorProfileGrants(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	grants, err := s.Store.VendorProfileGrants(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, grants)
}

// handleRevokeVendorProfileGrant lets an owner cut off a pending or active
// grant immediately.
func (s *Server) handleRevokeVendorProfileGrant(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.RevokeVendorProfileGrant(r.Context(), chi.URLParam(r, "id"), v.ID, c.Subject); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

// handleAdminVendorGrants lists profile-access grants across all vendors
// for the platform admin, optionally filtered by computed status.
func (s *Server) handleAdminVendorGrants(w http.ResponseWriter, r *http.Request) {
	grants, err := s.Store.AdminListVendorProfileGrants(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, grants)
}

// handleExtendVendorGrant lets a platform admin extend (positive minutes)
// or shorten (negative minutes) a grant's current expiry — one endpoint
// covers both directions.
func (s *Server) handleExtendVendorGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Minutes int `json:"minutes"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.Store.AdminAdjustVendorProfileGrant(r.Context(), id, req.Minutes); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "vendor_grant.adjust", "vendor_grant", id, map[string]any{"minutesDelta": req.Minutes})
	writeJSON(w, http.StatusOK, map[string]string{"status": "adjusted"})
}

// handleRevokeVendorGrantAdmin lets a platform admin revoke a grant on any
// vendor's behalf (e.g. in response to a support request or abuse report).
func (s *Server) handleRevokeVendorGrantAdmin(w http.ResponseWriter, r *http.Request) {
	g, err := s.Store.VendorProfileGrantByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	c := auth.FromContext(r.Context())
	if err := s.Store.RevokeVendorProfileGrant(r.Context(), g.ID, g.VendorID, c.Subject); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "vendor_grant.revoke", "vendor_grant", g.ID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
