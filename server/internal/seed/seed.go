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


	// ── Users ────────────────────────────────────────────────────────────
	type demoUser struct {
		id, email, name, country, style string
		role                            domain.Role
		vendorID                        *string
		interests                       []string
	}
	users := []demoUser{
		{"usr_lena", "lena@spotto.dev", "Lena M.", "Germany", "nomad", domain.RoleTraveller, nil, []string{"coffee", "coworking", "surf"}},
		{"usr_tom", "tom@spotto.dev", "Tom R.", "UK", "surf", domain.RoleTraveller, nil, []string{"surf", "beach"}},
		{"usr_sara", "sara@spotto.dev", "Sara P.", "Spain", "nomad", domain.RoleTraveller, nil, []string{"coffee", "coworking"}},
		{"usr_marco", "marco@spotto.dev", "Marco", "Italy", "surf", domain.RoleTraveller, nil, []string{"surf"}},
		{"usr_aiko", "aiko@spotto.dev", "Aiko", "Japan", "nomad", domain.RoleTraveller, nil, []string{"coffee"}},
		{"usr_sam", "sam@spotto.dev", "Sam", "Australia", "backpacker", domain.RoleTraveller, nil, []string{"beach"}},
		{"usr_chloe", "chloe@spotto.dev", "Chloé", "France", "luxury", domain.RoleTraveller, nil, []string{"food"}},
		{"usr_admin", "admin@spotto.dev", "Spotto Admin", "Sri Lanka", "", domain.RoleAdmin, nil, nil},
	}
	for _, u := range users {
		var style []string
		if u.style != "" {
			style = []string{u.style}
		}
		if err := s.CreateUser(ctx, domain.User{
			ID: u.id, Email: u.email, PasswordHash: hash, Role: u.role, Name: u.name,
			Country: u.country, TravelStyle: style, Interests: u.interests,
			VendorID: u.vendorID, CreatedAt: now.AddDate(0, -1, 0),
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
