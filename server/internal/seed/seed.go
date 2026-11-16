package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

const demoPassword = "spotto123"

func ptr[T any](v T) *T { return &v }

// Run wipes all tables and reloads the demo dataset. Idempotent.
func Run(ctx context.Context, s *postgres.Store) error {
	if err := s.Wipe(ctx); err != nil {
		return fmt.Errorf("wipe: %w", err)
	}
	now := time.Now()
	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return err
	}
	// ── Vendor Categories ────────────────────────────────────────────────────────
	for _, c := range InitialCategories {
		if err := s.CreateVendorCategory(ctx, c); err != nil {
			return fmt.Errorf("create category %s: %w", c.ID, err)
		}
	}

	// Update domain memory cache so validation passes during the rest of the seed run
	cats, err := s.GetVendorCategories(ctx)
	if err != nil {
		return fmt.Errorf("get categories: %w", err)
	}
	domain.InitVendorCategories(cats)


	// ── Pre-defined Access Control Groups ────────────────────────────────
	systemGroups := []domain.AdminGroup{
		{
			ID:          "super_admin",
			Name:        "Super Administrator",
			Description: "Unrestricted master access across all platform modules, user security, and audit trails.",
			Icon:        "ShieldAlert",
			IsSystem:    true,
			Permissions: []string{"*"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "developers",
			Name:        "Developers & QA",
			Description: "Engineering & QA staff: location simulation, AI console, geo-telemetry, and system diagnostics.",
			Icon:        "Sparkles",
			IsSystem:    true,
			Permissions: []string{"users.demo_privilege", "local_guide.view_chats", "local_guide.configure_limits", "local_guide.system_health", "geo.view_heat", "geo.filter", "audit_log.view"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "vendor_approvals",
			Name:        "Vendor Approvals & Operations",
			Description: "Review and approve/reject merchant applications, edit business profiles, and manage category taxonomy.",
			Icon:        "UserCheck",
			IsSystem:    true,
			Permissions: []string{"queue.view", "queue.approve", "queue.reject", "queue.request_edits", "vendors.view", "vendors.edit", "vendors.suspend", "categories.view", "categories.manage"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "content_editors",
			Name:        "Content Editors & Curators",
			Description: "Create and publish travel stories, destination audio guides, and edit audio transcripts.",
			Icon:        "Newspaper",
			IsSystem:    true,
			Permissions: []string{"content.view", "content.publish", "content.unpublish", "content.manage_transcripts", "users.manage_publisher"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "support_moderators",
			Name:        "Support & Community Moderation",
			Description: "Assist travelers and vendors, manage device sessions, moderate flagged reviews, and void claims.",
			Icon:        "Users",
			IsSystem:    true,
			Permissions: []string{"users.view", "users.revoke_sessions", "vendors.view", "vendors.moderate_reviews", "vendors.void_claims", "access_grants.view", "access_grants.manage"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "finance_billing",
			Name:        "Commercial & Monetization",
			Description: "Oversee merchant subscription plans, manual tier upgrades, package matrix builder, and daily quest sponsorships.",
			Icon:        "CreditCard",
			IsSystem:    true,
			Permissions: []string{"subscriptions.view", "subscriptions.upgrade", "packages.view", "packages.manage", "quests.view", "quests.manage_sponsorships", "quests.schedule"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
	for _, g := range systemGroups {
		_ = s.CreateAdminGroup(ctx, g)
	}

	// ── Users ────────────────────────────────────────────────────────────
	type demoUser struct {
		id, email, name, country, style string
		role                            domain.Role
		vendorID                        *string
		interests                       []string
		adminGroupID                    *string
		demoPrivilege                   bool
		demoLocation                    string
		demoLat                         *float64
		demoLng                         *float64
	}
	strPtr := func(s string) *string { return &s }
	floatPtr := func(f float64) *float64 { return &f }

	users := []demoUser{
		{"usr_lena", "lena@spotto.dev", "Lena M.", "Germany", "nomad", domain.RoleTraveller, nil, []string{"coffee", "coworking", "surf"}, nil, false, "", nil, nil},
		{"usr_tom", "tom@spotto.dev", "Tom R.", "UK", "surf", domain.RoleTraveller, nil, []string{"surf", "beach"}, nil, false, "", nil, nil},
		{"usr_sara", "sara@spotto.dev", "Sara P.", "Spain", "nomad", domain.RoleTraveller, nil, []string{"coffee", "coworking"}, nil, false, "", nil, nil},
		{"usr_marco", "marco@spotto.dev", "Marco", "Italy", "surf", domain.RoleTraveller, nil, []string{"surf"}, nil, false, "", nil, nil},
		{"usr_aiko", "aiko@spotto.dev", "Aiko", "Japan", "nomad", domain.RoleTraveller, nil, []string{"coffee"}, nil, false, "", nil, nil},
		{"usr_sam", "sam@spotto.dev", "Sam", "Australia", "backpacker", domain.RoleTraveller, nil, []string{"beach"}, nil, false, "", nil, nil},
		{"usr_chloe", "chloe@spotto.dev", "Chloé", "France", "luxury", domain.RoleTraveller, nil, []string{"food"}, nil, false, "", nil, nil},
		{"usr_admin", "admin@spotto.dev", "Spotto Admin", "Sri Lanka", "", domain.RoleAdmin, nil, nil, strPtr("super_admin"), false, "", nil, nil},
		{"usr_dev", "developer@spotto.dev", "Lead Developer", "Sri Lanka", "", domain.RoleAdmin, nil, nil, strPtr("developers"), true, "Ratnapura", floatPtr(6.6828), floatPtr(80.3992)},
		{"usr_approvals", "approvals@spotto.dev", "Vendor Operations", "Sri Lanka", "", domain.RoleAdmin, nil, nil, strPtr("vendor_approvals"), false, "", nil, nil},
		{"usr_editor", "editor@spotto.dev", "Content Curator", "Sri Lanka", "", domain.RoleAdmin, nil, nil, strPtr("content_editors"), false, "", nil, nil},
		{"usr_support", "support@spotto.dev", "Support Specialist", "Sri Lanka", "", domain.RoleAdmin, nil, nil, strPtr("support_moderators"), false, "", nil, nil},
		{"usr_finance", "finance@spotto.dev", "Finance Lead", "Sri Lanka", "", domain.RoleAdmin, nil, nil, strPtr("finance_billing"), false, "", nil, nil},
	}
	for _, u := range users {
		var style []string
		if u.style != "" {
			style = []string{u.style}
		}
		if err := s.CreateUser(ctx, domain.User{
			ID: u.id, Email: u.email, PasswordHash: hash, Role: u.role, Name: u.name,
			Country: u.country, TravelStyle: style, Interests: u.interests,
			VendorID: u.vendorID, AdminGroupID: u.adminGroupID, DemoPrivilege: u.demoPrivilege,
			DemoLocation: u.demoLocation, DemoLat: u.demoLat, DemoLng: u.demoLng,
			CreatedAt: now.AddDate(0, -1, 0),
		}); err != nil {
			return fmt.Errorf("user %s: %w", u.email, err)
		}
	}


	// ── Group for Lena (groupSeed from places.dart) ──────────────────────
	// Members must be real user IDs (not the old mock traveler names) so
	// GET /me/group can resolve them to actual profiles.
	if err := s.UpsertGroup(ctx, domain.Group{
		OwnerID: "usr_lena", Name: "Spotto Crew", Code: "GRP-LENA01",
		MemberIDs: []string{"usr_tom", "usr_sara", "usr_marco"},
	}); err != nil {
		return fmt.Errorf("group: %w", err)
	}


	// ── Regions (REGIONS from admin-data.jsx) ────────────────────────────
	for _, r := range Regions {
		if err := s.UpsertRegion(ctx, r); err != nil {
			return fmt.Errorf("region %s: %w", r.ID, err)
		}
	}



	return nil
}
