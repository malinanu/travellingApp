package httpapi

import (
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/spotto/server/internal/domain"
)

// MasterCommandCenterFeatures lists all features in the Command Center and
// breaks each one down into granular to-do / access control permissions.
var MasterCommandCenterFeatures = []domain.CommandCenterFeature{
	{
		ID:          "queue",
		Name:        "Vendor Approval Queue",
		Description: "Review incoming vendor applications, business registration certificates, and storefront profiles before publishing.",
		Category:    "operations",
		Icon:        "UserCheck",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "queue.view", FeatureID: "queue", Name: "View approval queue", Description: "Browse incoming vendor registration queue and uploaded proof docs", Risk: "low"},
			{ID: "queue.approve", FeatureID: "queue", Name: "Approve vendor", Description: "Approve vendor applications and grant immediate public listing", Risk: "high"},
			{ID: "queue.reject", FeatureID: "queue", Name: "Reject vendor", Description: "Decline vendor applications and provide rejection feedback", Risk: "high"},
			{ID: "queue.request_edits", FeatureID: "queue", Name: "Request application edits", Description: "Ask vendors to revise business docs, photos, or descriptions", Risk: "medium"},
		},
	},
	{
		ID:          "vendors",
		Name:        "Vendor Directory & Moderation",
		Description: "Oversee active vendors, edit metadata, manage operating status, moderate reviews, and void claims.",
		Category:    "operations",
		Icon:        "Store",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "vendors.view", FeatureID: "vendors", Name: "View vendor directory", Description: "Access vendor profiles, menus, media, and statistics", Risk: "low"},
			{ID: "vendors.edit", FeatureID: "vendors", Name: "Edit vendor details", Description: "Modify vendor profile fields, category, town, and tags", Risk: "medium"},
			{ID: "vendors.suspend", FeatureID: "vendors", Name: "Suspend / Reinstate", Description: "Temporarily pause or reactivate merchant store operations", Risk: "high"},
			{ID: "vendors.takedown", FeatureID: "vendors", Name: "Takedown vendor", Description: "Delist vendor profiles from discovery maps and listings", Risk: "critical"},
			{ID: "vendors.moderate_reviews", FeatureID: "vendors", Name: "Moderate customer reviews", Description: "Audit and delete flagged or fraudulent customer reviews", Risk: "medium"},
			{ID: "vendors.void_claims", FeatureID: "vendors", Name: "Void reward claims", Description: "Void fraudulent or disputed customer reward redemption claims", Risk: "medium"},
		},
	},
	{
		ID:          "vendor_logins",
		Name:        "Vendor Login Handover",
		Description: "Hand vendor accounts over to real business owners, change registered emails, and issue setup links.",
		Category:    "security",
		Icon:        "KeyRound",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "vendor_logins.transfer_email", FeatureID: "vendor_logins", Name: "Transfer login email", Description: "Reassign business login to verified owner email address", Risk: "critical"},
			{ID: "vendor_logins.reset_password", FeatureID: "vendor_logins", Name: "Reset owner password", Description: "Set temporary password or email 24-hour credential setup link", Risk: "critical"},
		},
	},
	{
		ID:          "users",
		Name:        "User & Device Session Control",
		Description: "Manage traveler and vendor accounts, revoke device sessions, grant publisher role, and set demo overrides.",
		Category:    "security",
		Icon:        "Users",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "users.view", FeatureID: "users", Name: "View accounts & devices", Description: "Search users and inspect signed-in devices and quest history", Risk: "low"},
			{ID: "users.revoke_sessions", FeatureID: "users", Name: "Revoke active sessions", Description: "Sign user out of all devices and terminate session tokens", Risk: "high"},
			{ID: "users.manage_publisher", FeatureID: "users", Name: "Manage publisher access", Description: "Grant or revoke travel story and guide writing permissions", Risk: "medium"},
			{ID: "users.demo_privilege", FeatureID: "users", Name: "Set developer demo privilege", Description: "Enable location simulation override for investor presentations", Risk: "low"},
			{ID: "users.change_role", FeatureID: "users", Name: "Convert account role", Description: "Change account between traveller and vendor types", Risk: "critical"},
			{ID: "users.delete", FeatureID: "users", Name: "Delete user account", Description: "Permanently delete user profile and associated data", Risk: "critical"},
		},
	},
	{
		ID:          "quests",
		Name:        "Daily Quests & Sponsorships",
		Description: "Review vendor sponsored rewards, allocate daily quest calendar schedules, and track quest completions.",
		Category:    "monetization",
		Icon:        "Activity",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "quests.view", FeatureID: "quests", Name: "View quests & completions", Description: "Track daily quest progress, completions, and merchant sponsorships", Risk: "low"},
			{ID: "quests.manage_sponsorships", FeatureID: "quests", Name: "Manage sponsorships", Description: "Approve or decline vendor quest reward offers", Risk: "medium"},
			{ID: "quests.schedule", FeatureID: "quests", Name: "Schedule daily quests", Description: "Assign vendor sponsorships to calendar dates", Risk: "medium"},
		},
	},
	{
		ID:          "geo",
		Name:        "Geo-analytics & Foot-traffic",
		Description: "Live geographic density heatmaps, traveler presence telemetry, and regional saturation metrics.",
		Category:    "intelligence",
		Icon:        "Map",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "geo.view_heat", FeatureID: "geo", Name: "View density heatmaps", Description: "Inspect live traveller density clusters and foot-traffic maps", Risk: "low"},
			{ID: "geo.filter", FeatureID: "geo", Name: "Filter regional clusters", Description: "Filter telemetry by town, beach zone, and time intervals", Risk: "low"},
		},
	},
	{
		ID:          "subscriptions",
		Name:        "Subscriptions & Billing",
		Description: "Monitor merchant subscription tiers, payment statuses, and apply manual package tier upgrades.",
		Category:    "monetization",
		Icon:        "CreditCard",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "subscriptions.view", FeatureID: "subscriptions", Name: "View subscription roster", Description: "Inspect active billing tiers and payment statuses", Risk: "low"},
			{ID: "subscriptions.upgrade", FeatureID: "subscriptions", Name: "Manual tier upgrade", Description: "Manually adjust vendor tier without Stripe payment", Risk: "high"},
		},
	},
	{
		ID:          "packages",
		Name:        "Vendor Package Builder",
		Description: "Interactive Kanban package allocator to configure pricing, feature pool allocations, and limits.",
		Category:    "monetization",
		Icon:        "Layers",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "packages.view", FeatureID: "packages", Name: "View package matrix", Description: "Inspect pricing tiers, limits, and feature allocations", Risk: "low"},
			{ID: "packages.manage", FeatureID: "packages", Name: "Publish package tiers", Description: "Reconfigure pricing, create tiers, and deploy package config", Risk: "critical"},
		},
	},
	{
		ID:          "content",
		Name:        "Travel Content Studio",
		Description: "Destination editorial CMS for travel blogs, audio guides, transcript editing, and publishing.",
		Category:    "editorial",
		Icon:        "Newspaper",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "content.view", FeatureID: "content", Name: "View stories & guides", Description: "Browse travel stories, guides, and drafts in editorial CMS", Risk: "low"},
			{ID: "content.publish", FeatureID: "content", Name: "Publish travel content", Description: "Approve submitted articles and push live to mobile apps", Risk: "medium"},
			{ID: "content.unpublish", FeatureID: "content", Name: "Unpublish articles", Description: "Remove live content and return stories to draft mode", Risk: "medium"},
			{ID: "content.manage_transcripts", FeatureID: "content", Name: "Manage audio transcripts", Description: "Edit speech transcripts and trigger AI reprocessing", Risk: "medium"},
		},
	},
	{
		ID:          "local_guide",
		Name:        "Local Guide AI Console",
		Description: "Traveller conversational AI telemetry, dynamic rate limits, token quotas, and system health diagnostics.",
		Category:    "intelligence",
		Icon:        "Sparkles",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "local_guide.view_chats", FeatureID: "local_guide", Name: "Inspect AI conversations", Description: "Review traveller conversational logs and AI suggestions", Risk: "low"},
			{ID: "local_guide.configure_limits", FeatureID: "local_guide", Name: "Configure AI rate limits", Description: "Adjust dynamic rate limits and daily token quotas", Risk: "high"},
			{ID: "local_guide.system_health", FeatureID: "local_guide", Name: "System health & re-embed", Description: "Trigger vector re-embeddings and diagnose model health", Risk: "high"},
		},
	},
	{
		ID:          "categories",
		Name:        "Directory Taxonomy & Tags",
		Description: "Master taxonomy of merchant categories, discovery tags, amenity badges, and map filters.",
		Category:    "operations",
		Icon:        "Shapes",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "categories.view", FeatureID: "categories", Name: "View taxonomy", Description: "Inspect business categories, icons, and search tags", Risk: "low"},
			{ID: "categories.manage", FeatureID: "categories", Name: "Manage categories", Description: "Create, rename, reorder, or delete merchant categories", Risk: "medium"},
		},
	},
	{
		ID:          "access_grants",
		Name:        "Access Grants PIN Manager",
		Description: "Audit temporary manager PIN delegations across vendors, extend durations, or revoke immediately.",
		Category:    "security",
		Icon:        "KeyRound",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "access_grants.view", FeatureID: "access_grants", Name: "View active PIN grants", Description: "Monitor temporary merchant profile edit PIN delegations", Risk: "low"},
			{ID: "access_grants.manage", FeatureID: "access_grants", Name: "Extend / Revoke PINs", Description: "Extend or immediately revoke temporary PIN delegations", Risk: "medium"},
		},
	},
	{
		ID:          "audit_log",
		Name:        "Security & Audit Trail",
		Description: "Immutable cross-admin security audit trail of all destructive actions, actor identities, and IP metadata.",
		Category:    "security",
		Icon:        "History",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "audit_log.view", FeatureID: "audit_log", Name: "View audit trail", Description: "Browse security audit log of administrative mutations", Risk: "medium"},
			{ID: "audit_log.inspect", FeatureID: "audit_log", Name: "Inspect mutation metadata", Description: "View before/after payloads, actor IP addresses, and reasons", Risk: "high"},
		},
	},
	{
		ID:          "settings",
		Name:        "Platform Settings & Phone Limits",
		Description: "Global system thresholds, daily OTP SMS/WhatsApp verification limits, and anti-abuse safeguards.",
		Category:    "operations",
		Icon:        "ShieldAlert",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "settings.view", FeatureID: "settings", Name: "View system settings", Description: "Inspect platform configurations and phone-code thresholds", Risk: "low"},
			{ID: "settings.manage_limits", FeatureID: "settings", Name: "Update verification limits", Description: "Adjust daily SMS/WhatsApp OTP verification budgets", Risk: "high"},
		},
	},
	{
		ID:          "access_control",
		Name:        "Command Center User Access Control",
		Description: "Operator permission management, feature breaking tool, and granular To-Do access allocations.",
		Category:    "security",
		Icon:        "ShieldCheck",
		SubPermissions: []domain.CommandCenterPermission{
			{ID: "access_control.view", FeatureID: "access_control", Name: "View operator access controls", Description: "Browse command center access control boards and operator permissions", Risk: "low"},
			{ID: "access_control.manage", FeatureID: "access_control", Name: "Manage & break access controls", Description: "Break features down, reassign permissions, and save operator access allocations", Risk: "critical"},
		},
	},
}

// handleAdminAccessControlCatalog returns the complete Command Center feature catalog
// and broken-down granular permissions.
func (s *Server) handleAdminAccessControlCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"features": MasterCommandCenterFeatures,
	})
}

// handleAdminAccessControlAdmins returns all Command Center admin accounts with
// their current permission set and super-admin status.
func (s *Server) handleAdminAccessControlAdmins(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	admins, err := s.Store.ListAdminUsers(ctx)
	if err != nil {
		log.Printf("list admin users: %v", err)
		writeErr(w, http.StatusInternalServerError, "Couldn't load Command Center admin users.")
		return
	}
	for i := range admins {
		admins[i].IsSuperAdmin = s.isSuperAdmin(admins[i].Email)
	}
	writeJSON(w, http.StatusOK, admins)
}

// handleAdminAccessControlUpdate updates an admin user's Command Center permissions.
func (s *Server) handleAdminAccessControlUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Permissions []string `json:"permissions"`
		Reason      string   `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if req.Permissions == nil || reason == "" {
		writeErr(w, http.StatusBadRequest, "permissions and reason are required")
		return
	}
	u, err := s.Store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	if u.Role != domain.RoleAdmin {
		writeErr(w, http.StatusBadRequest, "permissions can only be configured for admin accounts")
		return
	}

	cleaned := make([]string, 0, len(req.Permissions))
	seen := make(map[string]bool, len(req.Permissions))
	for _, p := range req.Permissions {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			cleaned = append(cleaned, p)
		}
	}

	if err := s.Store.SetAdminPermissions(r.Context(), id, cleaned); err != nil {
		log.Printf("set admin permissions user=%s: %v", id, err)
		writeErr(w, http.StatusInternalServerError, "Couldn't save permissions. Please try again.")
		return
	}
	s.logAdminAction(r.Context(), "user.permissions", "user", id, map[string]any{
		"permissions": cleaned,
		"reason":      reason,
	})
	perms, err := s.Store.AdminPermissions(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":           id,
		"adminPermissions": nonNilStrings(perms),
	})
}
