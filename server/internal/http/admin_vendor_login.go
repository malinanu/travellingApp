package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// setupLinkTTL is longer than a normal reset link: the new owner may not
// check their email the same hour the admin hands the account over.
const setupLinkTTL = 24 * time.Hour

// handleAdminVendorMemberLogin hands a vendor account to someone else by
// changing its login email and/or password. Places are often created by
// someone other than the real owner (an importer placeholder, or a person
// registering a business for its owner), so when the owner turns up, an
// admin moves the login over to them. Modes:
//   - "password": the admin sets the password and passes it on.
//   - "link":     the old password stops working, and the new email gets a
//     24-hour link to choose one.
//   - "none":     only the email changes.
//
// Whoever held the account is signed out everywhere either way.
func (s *Server) handleAdminVendorMemberLogin(w http.ResponseWriter, r *http.Request) {
	vendorID, userID := chi.URLParam(r, "id"), chi.URLParam(r, "userId")
	var req struct {
		Email    string `json:"email"`
		Mode     string `json:"mode"`
		Password string `json:"password"`
		Reason   string `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "reason required")
		return
	}
	if req.Mode != "password" && req.Mode != "link" && req.Mode != "none" {
		writeErr(w, http.StatusBadRequest, `mode must be "password", "link" or "none"`)
		return
	}

	// Only vendor accounts of this vendor. This route must never reach an
	// admin or traveller account.
	u, err := s.Store.UserByID(ctx, userID)
	if err != nil || u.Role != domain.RoleVendor || u.VendorID == nil || *u.VendorID != vendorID {
		writeErr(w, http.StatusNotFound, "no such member of this business")
		return
	}
	v, err := s.Store.VendorByID(ctx, vendorID)
	if err != nil {
		storeErr(w, err)
		return
	}

	oldEmail := u.Email
	newEmail := strings.TrimSpace(strings.ToLower(req.Email))
	if newEmail == "" {
		newEmail = oldEmail
	}
	if !validateTextLength(w, "Email", newEmail, 254) {
		return
	}
	if !strings.Contains(newEmail, "@") {
		writeErr(w, http.StatusBadRequest, "enter a valid email address")
		return
	}
	emailChanged := newEmail != oldEmail
	if emailChanged {
		if _, err := s.Store.UserByEmail(ctx, newEmail); err == nil {
			writeErr(w, http.StatusConflict, "email already registered")
			return
		} else if !errors.Is(err, postgres.ErrNotFound) {
			log.Printf("vendor login handover lookup email user=%s: %v", userID, err)
			writeErr(w, http.StatusInternalServerError, "Couldn't check that email. Nothing was changed; please try again.")
			return
		}
	}
	if !emailChanged && req.Mode == "none" {
		writeErr(w, http.StatusBadRequest, "nothing to change: enter a new email or choose a password option")
		return
	}
	var hash string
	if req.Mode == "password" {
		if len(req.Password) < 10 {
			writeErr(w, http.StatusBadRequest, "password must be at least 10 characters")
			return
		}
		if !validateTextLength(w, "Password", req.Password, 128) {
			return
		}
		if isWeakPassword(req.Password, newEmail) {
			writeErr(w, http.StatusBadRequest, "password is too weak: avoid all-digit, all-repeated, or email-derived passwords")
			return
		}
		if hash, err = auth.HashPassword(req.Password); err != nil {
			log.Printf("vendor login handover hash user=%s: %v", userID, err)
			writeErr(w, http.StatusInternalServerError, "Couldn't set that password. Nothing was changed; please try again.")
			return
		}
	}

	fail := func(step string, err error) {
		log.Printf("vendor login handover %s vendor=%s user=%s: %v", step, vendorID, userID, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't finish the hand-over ("+step+"). Some changes may already be saved; check the Team tab and History before retrying.")
	}
	if emailChanged {
		if err := s.Store.UpdateUserEmail(ctx, userID, newEmail); err != nil {
			fail("update email", err)
			return
		}
		// Any earlier reset link went to the old inbox.
		if err := s.Store.InvalidatePasswordResetTokens(ctx, userID); err != nil {
			fail("cancel old reset links", err)
			return
		}
	}
	switch req.Mode {
	case "password":
		err = s.Store.UpdateUserPassword(ctx, userID, hash)
	case "link":
		err = s.Store.LockUserPassword(ctx, userID)
	}
	if err != nil {
		fail("set password", err)
		return
	}
	if _, err := s.Store.RevokeAllUserSessions(ctx, userID); err != nil {
		fail("sign out old sessions", err)
		return
	}

	resp := map[string]any{"mode": req.Mode, "email": newEmail, "emailSent": false}
	if req.Mode == "link" {
		raw := newResetToken()
		if err := s.Store.CreatePasswordResetToken(ctx, newID("prt"), userID, hashResetToken(raw), time.Now().Add(setupLinkTTL)); err != nil {
			fail("create set-password link", err)
			return
		}
		link := s.VendorAppURL + "/login?resetToken=" + raw
		if !s.Mail.Enabled() {
			// No SMTP: hand the link to the admin to pass on instead of
			// pretending it was emailed.
			resp["setupLink"] = link
		} else if err := s.Mail.Send(newEmail, "Set up your "+v.Name+" account on Spotto", accountSetupEmailHTML(v.Name, link)); err != nil {
			log.Printf("vendor login handover setup email user=%s: %v", userID, err)
			resp["setupLink"] = link
			resp["emailError"] = "The set-password email couldn't be sent. Give the owner this link instead (valid for 24 hours)."
		} else {
			resp["emailSent"] = true
		}
	}
	// Tell the old address, unless it was a placeholder nobody reads.
	if emailChanged && !strings.HasSuffix(oldEmail, "@spotto.dev") {
		to, body := oldEmail, loginEmailChangedEmailHTML(v.Name)
		s.Jobs.Enqueue(func() { s.Mail.Send(to, "Your Spotto login was changed", body) })
	}

	// Never the password itself.
	s.logAdminAction(ctx, "vendor.login_change", "vendor", vendorID, map[string]any{
		"userId": userID, "oldEmail": oldEmail, "newEmail": newEmail, "mode": req.Mode, "reason": reason,
	})
	writeJSON(w, http.StatusOK, resp)
}

// handleAdminSetUserPermissions lets a super-admin grant or remove an
// admin's extra permissions (currently only vendor-login hand-over).
func (s *Server) handleAdminSetUserPermissions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		VendorLogins *bool  `json:"vendorLogins"`
		Reason       string `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if req.VendorLogins == nil || reason == "" {
		writeErr(w, http.StatusBadRequest, "vendorLogins and reason are required")
		return
	}
	u, err := s.Store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	if u.Role != domain.RoleAdmin {
		writeErr(w, http.StatusBadRequest, "permissions can only be given to admin accounts")
		return
	}
	if err := s.Store.SetAdminPermission(r.Context(), id, PermVendorLogins, *req.VendorLogins); err != nil {
		log.Printf("set admin permission user=%s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't save that permission. Please try again.")
		return
	}
	s.logAdminAction(r.Context(), "user.permissions", "user", id, map[string]any{
		"permission": PermVendorLogins, "granted": *req.VendorLogins, "reason": reason,
	})
	perms, err := s.Store.AdminPermissions(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"adminPermissions": nonNilStrings(perms)})
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
