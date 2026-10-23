package httpapi

// Super-admin account controls: move an account between traveller and
// vendor, and — as a last resort — permanently delete a user or a business.
// All three sit behind RequireSuperAdmin, need a reason, and are written to
// the admin audit log with enough detail to know what was there afterwards.
// Admin accounts are out of reach here (ADMIN_EMAIL is recreated at boot
// anyway).

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func isVendorOwner(u domain.User) bool {
	// An empty vendor_role predates the column and means owner (see
	// auth.RequireVendorOwner).
	return u.Role == domain.RoleVendor && u.VendorID != nil &&
		(u.VendorRole == domain.VendorRoleOwner || u.VendorRole == "")
}

// ownerHandover works out who takes over when u leaves their business. It
// returns nil when u isn't its owner, when another owner remains, or when
// nobody else is on the team (the business is then left ownerless). It
// returns an error message when a new owner has to be chosen and wasn't.
func (s *Server) ownerHandover(ctx context.Context, u domain.User, newOwnerID string) (*postgres.OwnerHandover, string, error) {
	if !isVendorOwner(u) {
		return nil, "", nil
	}
	team, err := s.Store.VendorTeamMembers(ctx, *u.VendorID)
	if err != nil {
		return nil, "", err
	}
	var others []domain.User
	for _, m := range team {
		if m.ID == u.ID {
			continue
		}
		if isVendorOwner(m) {
			return nil, "", nil
		}
		others = append(others, m)
	}
	if len(others) == 0 {
		return nil, "", nil
	}
	for _, m := range others {
		if m.ID == newOwnerID {
			return &postgres.OwnerHandover{UserID: m.ID, VendorID: *u.VendorID}, "", nil
		}
	}
	return nil, "This person owns a business with other team members — choose who becomes the owner.", nil
}

// deleteMediaFor removes an owner's media rows and, when uploads are
// configured, their R2 objects. Best effort: failures are logged.
func (s *Server) deleteMediaFor(ctx context.Context, ownerType, ownerID string) {
	items, err := s.Store.MediaAllByOwner(ctx, ownerType, ownerID)
	if err != nil {
		log.Printf("delete media for %s/%s: %v", ownerType, ownerID, err)
		return
	}
	for _, m := range items {
		if s.Media != nil {
			_ = s.Media.Delete(ctx, m.ObjectKey)
			if m.ThumbKey != "" {
				_ = s.Media.Delete(ctx, m.ThumbKey)
			}
		}
		if err := s.Store.DeleteMedia(ctx, m.ID); err != nil && !errors.Is(err, postgres.ErrNotFound) {
			log.Printf("delete media %s: %v", m.ID, err)
		}
	}
}

func (s *Server) revokeSessionsBestEffort(ctx context.Context, userID string) {
	if _, err := s.Store.RevokeAllUserSessions(ctx, userID); err != nil && !errors.Is(err, postgres.ErrNotFound) {
		log.Printf("revoke sessions for %s: %v", userID, err)
	}
}

// handleAdminChangeUserRole moves an account between traveller and vendor.
// Becoming a vendor links them to an existing business (with a team role)
// or creates a new pending one; becoming a traveller unlinks them, handing
// ownership to a chosen team member when they owned it. They're signed out
// everywhere so they sign in again in the right app.
func (s *Server) handleAdminChangeUserRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Role        string `json:"role"`
		Reason      string `json:"reason"`
		VendorID    string `json:"vendorId"`
		VendorRole  string `json:"vendorRole"`
		NewBusiness *struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Town     string `json:"town"`
		} `json:"newBusiness"`
		NewOwnerID string `json:"newOwnerId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "a reason is required")
		return
	}
	target, err := s.Store.UserByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if target.Role == domain.RoleAdmin {
		writeErr(w, http.StatusBadRequest, "Admin accounts can't be changed here.")
		return
	}

	ch := postgres.RoleChange{UserID: target.ID}
	audit := map[string]any{"from": target.Role, "fromVendorId": target.VendorID, "reason": reason}
	switch domain.Role(req.Role) {
	case domain.RoleVendor:
		if target.Role == domain.RoleVendor && target.VendorID != nil {
			writeErr(w, http.StatusBadRequest, "This account is already a Vendor on a business. Change it to Traveller first to move it.")
			return
		}
		linking, creating := req.VendorID != "", req.NewBusiness != nil
		if linking == creating {
			writeErr(w, http.StatusBadRequest, "Choose an existing business or create a new one.")
			return
		}
		ch.Role = domain.RoleVendor
		if linking {
			v, err := s.Store.VendorByID(r.Context(), req.VendorID)
			if errors.Is(err, postgres.ErrNotFound) {
				writeErr(w, http.StatusBadRequest, "That business doesn't exist.")
				return
			} else if err != nil {
				storeErr(w, err)
				return
			}
			switch req.VendorRole {
			case domain.VendorRoleOwner, domain.VendorRoleManager, domain.VendorRoleStaff:
			default:
				writeErr(w, http.StatusBadRequest, "team role must be owner, manager or staff")
				return
			}
			ch.VendorID, ch.VendorRole = &v.ID, req.VendorRole
			audit["vendorId"], audit["vendorName"], audit["vendorRole"] = v.ID, v.Name, req.VendorRole
		} else {
			name := sanitizeText(req.NewBusiness.Name, 120)
			if name == "" {
				writeErr(w, http.StatusBadRequest, "the new business needs a name")
				return
			}
			if !domain.ValidCategoryID(req.NewBusiness.Category) {
				writeErr(w, http.StatusBadRequest, "unknown business category")
				return
			}
			v := domain.Vendor{
				ID: newID("ven"), Name: name, Category: req.NewBusiness.Category,
				Town: sanitizeText(req.NewBusiness.Town, 120), Owner: target.Name,
				Status: domain.VendorPending, Priority: "normal", Flags: []string{}, Tier: "Free",
				SubmittedAt: time.Now(),
			}
			ch.NewVendor, ch.VendorID, ch.VendorRole = &v, &v.ID, domain.VendorRoleOwner
			audit["newBusinessId"], audit["vendorName"], audit["vendorRole"] = v.ID, v.Name, domain.VendorRoleOwner
		}
	case domain.RoleTraveller:
		if target.Role == domain.RoleTraveller {
			writeErr(w, http.StatusBadRequest, "This account is already a Traveller.")
			return
		}
		handover, problem, err := s.ownerHandover(r.Context(), target, req.NewOwnerID)
		if err != nil {
			storeErr(w, err)
			return
		}
		if problem != "" {
			writeErr(w, http.StatusBadRequest, problem)
			return
		}
		ch.Role, ch.NewOwner = domain.RoleTraveller, handover
		if handover != nil {
			audit["newOwnerId"] = handover.UserID
		}
	default:
		writeErr(w, http.StatusBadRequest, "role must be traveller or vendor")
		return
	}
	audit["to"] = ch.Role

	if err := s.Store.ChangeUserRole(r.Context(), ch); err != nil {
		if errors.Is(err, postgres.ErrNotOnTeam) {
			writeErr(w, http.StatusBadRequest, "The new owner must be on the same business's team.")
			return
		}
		storeErr(w, err)
		return
	}
	s.revokeSessionsBestEffort(r.Context(), target.ID)
	if ch.NewOwner != nil {
		s.revokeSessionsBestEffort(r.Context(), ch.NewOwner.UserID)
	}
	s.logAdminAction(r.Context(), "user.role_change", "user", target.ID, audit)

	updated, err := s.Store.UserByID(r.Context(), target.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	updated.AvatarURL = s.signMediaPath(updated.AvatarURL)
	writeJSON(w, http.StatusOK, map[string]any{
		"user": updated,
		// Google/Apple-only accounts have no password; a new vendor needs
		// "Forgot password" on the vendor site to set one.
		"noPassword": target.PasswordHash == "",
	})
}

// handleAdminDeleteUser permanently deletes a traveller or vendor account.
// The caller must type the account's email to confirm.
func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason     string `json:"reason"`
		Confirm    string `json:"confirm"`
		NewOwnerID string `json:"newOwnerId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "a reason is required")
		return
	}
	target, err := s.Store.UserByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if target.Role == domain.RoleAdmin {
		writeErr(w, http.StatusBadRequest, "Admin accounts can't be deleted here.")
		return
	}
	if target.ID == auth.FromContext(r.Context()).Subject {
		writeErr(w, http.StatusBadRequest, "You can't delete your own account here.")
		return
	}
	if !strings.EqualFold(strings.TrimSpace(req.Confirm), target.Email) {
		writeErr(w, http.StatusBadRequest, "Type the account's email exactly to confirm.")
		return
	}
	handover, problem, err := s.ownerHandover(r.Context(), target, req.NewOwnerID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if problem != "" {
		writeErr(w, http.StatusBadRequest, problem)
		return
	}
	if err := s.Store.DeleteUserPermanently(r.Context(), target.ID, handover); err != nil {
		if errors.Is(err, postgres.ErrNotOnTeam) {
			writeErr(w, http.StatusBadRequest, "The new owner must be on the same business's team.")
			return
		}
		storeErr(w, err)
		return
	}
	s.deleteMediaFor(r.Context(), "user", target.ID)
	audit := map[string]any{
		"email": target.Email, "name": target.Name, "role": target.Role,
		"vendorId": target.VendorID, "vendorRole": target.VendorRole, "reason": reason,
	}
	if handover != nil {
		s.revokeSessionsBestEffort(r.Context(), handover.UserID)
		audit["newOwnerId"] = handover.UserID
	}
	s.logAdminAction(r.Context(), "user.delete", "user", target.ID, audit)
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// handleAdminDeleteVendor permanently deletes a business: its offers, menu,
// photos, bookings and map spot. Its team accounts are signed out and
// unlinked, not deleted. The caller must type the business name to confirm.
func (s *Server) handleAdminDeleteVendor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason  string `json:"reason"`
		Confirm string `json:"confirm"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "a reason is required")
		return
	}
	v, err := s.Store.VendorByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if !strings.EqualFold(strings.TrimSpace(req.Confirm), strings.TrimSpace(v.Name)) {
		writeErr(w, http.StatusBadRequest, "Type the business name exactly to confirm.")
		return
	}
	today, _ := questDayUTC(time.Now())
	days, err := s.Store.VendorQuestDaysFrom(r.Context(), v.ID, today)
	if err != nil {
		storeErr(w, err)
		return
	}
	if len(days) > 0 {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":     fmt.Sprintf("This business sponsors the daily quest on %s. Reassign those days first.", strings.Join(days, ", ")),
			"questDays": days,
		})
		return
	}
	team, err := s.Store.VendorTeamMembers(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	var owners []string
	for _, m := range team {
		if isVendorOwner(m) {
			owners = append(owners, m.Email)
		}
	}

	res, err := s.Store.DeleteVendorPermanently(r.Context(), v.ID, today)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, id := range res.DetachedUserIDs {
		s.revokeSessionsBestEffort(r.Context(), id)
	}
	s.deleteMediaFor(r.Context(), "vendor", v.ID)
	for _, placeID := range res.PlaceIDs {
		s.deleteMediaFor(r.Context(), "place", placeID)
	}
	s.logAdminAction(r.Context(), "vendor.delete", "vendor", v.ID, map[string]any{
		"name": v.Name, "status": v.Status, "ownerEmails": owners, "placeIds": res.PlaceIDs,
		"detachedMembers": len(res.DetachedUserIDs), "reason": reason,
	})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "detachedMembers": len(res.DetachedUserIDs)})
}
