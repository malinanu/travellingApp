package httpapi

import (
	"net/http"
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
// needs a Store lookup (email isn't in the JWT claims) — matching how every
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
