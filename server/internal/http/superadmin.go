package httpapi

import (
	"context"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/spotto/server/internal/auth"
)

// superAdminSet builds a lowercased, trimmed lookup set from
// config.Config.SuperAdminEmails.
func superAdminSet(emails []string) map[string]bool {
	set := make(map[string]bool, len(emails))
	for _, e := range emails {
		set[strings.ToLower(strings.TrimSpace(e))] = true
	}
	return set
}

// isSuperAdmin reports whether email is listed in SUPER_ADMIN_EMAILS.
func (s *Server) isSuperAdmin(email string) bool {
	return s.SuperAdminEmails[strings.ToLower(strings.TrimSpace(email))]
}

// RequireSuperAdmin further restricts an admin-role route (use after
// auth.RequireRole(domain.RoleAdmin)) to accounts listed in
// SUPER_ADMIN_EMAILS. It lives here rather than in package auth because it
// needs a Store lookup (email isn't in the JWT claims) â€” matching how every
// other DB-dependent check in this codebase (Authenticated itself, the
// vendor-grant access checks) is a *Server method, not a package-auth func.
func (s *Server) RequireSuperAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := auth.FromContext(r.Context())
		if c == nil {
			writeErr(w, http.StatusForbidden, "super-admin only")
			return
		}
		u, err := s.Store.UserByID(r.Context(), c.Subject)
		if err != nil || !s.isSuperAdmin(u.Email) {
			writeErr(w, http.StatusForbidden, "super-admin only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// PermVendorLogins lets an admin change a vendor account's login email and
// password (handing a business over to its real owner). Super-admins grant
// it per admin; they always hold it themselves.
const PermVendorLogins = "vendor_logins"

// hasAdminPermission reports whether this admin may use perm: super-admins
// always can, other admins only when a super-admin granted it.
func (s *Server) hasAdminPermission(ctx context.Context, userID, email, perm string) bool {
	if s.isSuperAdmin(email) {
		return true
	}
	perms, err := s.Store.AdminPermissions(ctx, userID)
	if err != nil {
		log.Printf("admin permissions user=%s: %v", userID, err)
		return false
	}
	return slices.Contains(perms, perm)
}

// RequireAdminPermission restricts an admin-role route to admins holding
// perm (see hasAdminPermission). Same shape as RequireSuperAdmin.
func (s *Server) RequireAdminPermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := auth.FromContext(r.Context())
			if c == nil {
				writeErr(w, http.StatusForbidden, "you don't have permission to do this")
				return
			}
			u, err := s.Store.UserByID(r.Context(), c.Subject)
			if err != nil || !s.hasAdminPermission(r.Context(), u.ID, u.Email, perm) {
				writeErr(w, http.StatusForbidden, "you don't have permission to do this; ask a super-admin")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
