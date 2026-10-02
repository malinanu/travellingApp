package postgres

import (
	"context"
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
