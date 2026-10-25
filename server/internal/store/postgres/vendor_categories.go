package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spotto/server/internal/domain"
)

// GetVendorCategories retrieves all vendor categories ordered by segment and label.
func (s *Store) GetVendorCategories(ctx context.Context) ([]domain.VendorCategory, error) {
	const q = `
		SELECT id, label, segment, icon, relevant_vibe_dimensions, booking_type
		FROM vendor_categories
		ORDER BY segment, label
	`
	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []domain.VendorCategory
	for rows.Next() {
		var c domain.VendorCategory
		var bookingType *string
		if err := rows.Scan(&c.ID, &c.Label, &c.Segment, &c.Icon, &c.RelevantVibeDimensions, &bookingType); err != nil {
			return nil, err
		}
		if bookingType != nil {
			c.BookingType = domain.BookingType(*bookingType)
		}
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

// CreateVendorCategory inserts a new category.
func (s *Store) CreateVendorCategory(ctx context.Context, c domain.VendorCategory) error {
	const q = `
		INSERT INTO vendor_categories (id, label, segment, icon, relevant_vibe_dimensions, booking_type)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	var bt *string
	if c.BookingType != "" {
		s := string(c.BookingType)
		bt = &s
	}
	_, err := s.Pool.Exec(ctx, q, c.ID, c.Label, string(c.Segment), c.Icon, c.RelevantVibeDimensions, bt)
	return err
}

// EnsureVendorCategories inserts any of cats that don't exist yet and leaves
// existing rows untouched, so an admin's label/icon edits survive restarts.
// Run on API startup so a database that never had the full taxonomy seeded
// (only the legacy migration rows) still gets every category.
func (s *Store) EnsureVendorCategories(ctx context.Context, cats []domain.VendorCategory) (int, error) {
	const q = `
		INSERT INTO vendor_categories (id, label, segment, icon, relevant_vibe_dimensions, booking_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO NOTHING
	`
	added := 0
	for _, c := range cats {
		var bt *string
		if c.BookingType != "" {
			b := string(c.BookingType)
			bt = &b
		}
		tag, err := s.Pool.Exec(ctx, q, c.ID, c.Label, string(c.Segment), c.Icon, c.RelevantVibeDimensions, bt)
		if err != nil {
			return added, err
		}
		added += int(tag.RowsAffected())
	}
	return added, nil
}

// UpdateVendorCategoryRecord updates an existing category.
func (s *Store) UpdateVendorCategoryRecord(ctx context.Context, c domain.VendorCategory) error {
	const q = `
		UPDATE vendor_categories
		SET label = $2, segment = $3, icon = $4, relevant_vibe_dimensions = $5, booking_type = $6, updated_at = NOW()
		WHERE id = $1
	`
	var bt *string
	if c.BookingType != "" {
		s := string(c.BookingType)
		bt = &s
	}
	cmd, err := s.Pool.Exec(ctx, q, c.ID, c.Label, string(c.Segment), c.Icon, c.RelevantVibeDimensions, bt)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteVendorCategory removes a category. It will fail if in use due to FK constraints.
func (s *Store) DeleteVendorCategory(ctx context.Context, id string) error {
	const q = `DELETE FROM vendor_categories WHERE id = $1`
	cmd, err := s.Pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DropImportedVendorNameSuffix renames businesses the Sri Lanka importer
// called "<place name> Vendor" back to their place's name. Only an exact
// match is touched, so a business genuinely named "... Vendor" stays. Safe
// to run on every start: once renamed nothing matches.
func (s *Store) DropImportedVendorNameSuffix(ctx context.Context) (int, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE vendors v SET name = p.name
		FROM places p
		WHERE p.vendor_id = v.id AND p.name <> '' AND v.name = p.name || ' Vendor'`)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// RecategorizeUncategorizedVendors gives every business still filed under
// the catch-all "uncategorized" category (migration 032 put every empty
// category there — mostly places from an early import) its real category,
// worked out from its map place: the place's own categoryId when that's a
// known category, else its type and tags (domain.CategoryForPlaceData, as
// the importer does today). The place's categoryId and pin colour are
// brought in line too. Businesses without a place are left alone. Safe to
// run on every start: once they're fixed nothing matches. Needs the
// category cache loaded (domain.InitVendorCategories).
func (s *Store) RecategorizeUncategorizedVendors(ctx context.Context) (int, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT ON (v.id) v.id, v.category, p.id, p.data
		FROM vendors v JOIN places p ON p.vendor_id = v.id
		WHERE v.category IN ('uncategorized', 'general', '')
		ORDER BY v.id, p.id`)
	if err != nil {
		return 0, err
	}
	type pending struct {
		vendorID, oldCategory, placeID string
		data                           []byte
	}
	var todo []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.vendorID, &p.oldCategory, &p.placeID, &p.data); err != nil {
			rows.Close()
			return 0, err
		}
		todo = append(todo, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	fixed := 0
	for _, p := range todo {
		var data map[string]any
		if err := json.Unmarshal(p.data, &data); err != nil || data == nil {
			data = map[string]any{}
		}
		id, _ := data["categoryId"].(string)
		cat, ok := domain.CategoryByID(id)
		if !ok || cat.ID == "uncategorized" || cat.ID == "general" {
			if cat, ok = domain.CategoryByID(domain.CategoryForPlaceData(data)); !ok {
				continue
			}
		}
		data["categoryId"] = cat.ID
		data["color"] = domain.PinColorForSegment(cat.Segment)
		b, err := json.Marshal(data)
		if err != nil {
			return fixed, err
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return fixed, err
		}
		tag, err := tx.Exec(ctx, `UPDATE vendors SET category=$2 WHERE id=$1 AND category=$3`, p.vendorID, cat.ID, p.oldCategory)
		if err == nil && tag.RowsAffected() == 1 {
			_, err = tx.Exec(ctx, `UPDATE places SET data=$2 WHERE id=$1`, p.placeID, b)
		}
		if err != nil {
			tx.Rollback(ctx)
			return fixed, err
		}
		if err := tx.Commit(ctx); err != nil {
			return fixed, err
		}
		if tag.RowsAffected() == 1 {
			fixed++
		}
	}
	return fixed, nil
}

// Point is a lat/lng with a weight: one business, or one place's count of
// traveller actions.
type Point struct {
	Lat, Lng float64
	Weight   int
}

// LiveVendorPoints lists every approved business that has a location.
func (s *Store) LiveVendorPoints(ctx context.Context) ([]Point, error) {
	return collectPoints(ctx, s, `
		SELECT lat, lng, 1 FROM vendors
		WHERE status = 'approved' AND (lat <> 0 OR lng <> 0)`)
}

// PlaceActivitySince counts traveller actions — check-ins, vibe check-ins,
// reviews and saved places — per place since the given time, with the
// place's location.
func (s *Store) PlaceActivitySince(ctx context.Context, since time.Time) ([]Point, error) {
	return collectPoints(ctx, s, `
		SELECT (p.data->>'lat')::float8, (p.data->>'lng')::float8, a.n::int
		FROM (
			SELECT place_id, count(*) AS n FROM (
				SELECT place_id FROM checkins WHERE created_at >= $1
				UNION ALL SELECT place_id FROM vibe_checkins WHERE created_at >= $1
				UNION ALL SELECT place_id FROM reviews WHERE created_at >= $1
				UNION ALL SELECT place_id FROM saved_places WHERE created_at >= $1
			) acts GROUP BY place_id
		) a JOIN places p ON p.id = a.place_id
		WHERE jsonb_typeof(p.data->'lat') = 'number' AND jsonb_typeof(p.data->'lng') = 'number'`, since)
}

func collectPoints(ctx context.Context, s *Store, sql string, args ...any) ([]Point, error) {
	rows, err := s.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Lat, &p.Lng, &p.Weight); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
