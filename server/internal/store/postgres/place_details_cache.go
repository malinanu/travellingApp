package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// GetPlaceDetailsCache returns a still-valid cached Google Places Details
// response for a Google place ID, if one exists — backs
// internal/placedetails's in-memory cache with something that survives a
// process restart (see migrations/015_place_details_cache.sql).
func (s *Store) GetPlaceDetailsCache(ctx context.Context, googlePlaceID string) (json.RawMessage, bool, error) {
	var details json.RawMessage
	var expiresAt time.Time
	err := s.Pool.QueryRow(ctx,
		`SELECT details, expires_at FROM place_details_cache WHERE place_id=$1`, googlePlaceID,
	).Scan(&details, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Now().After(expiresAt) {
		return nil, false, nil
	}
	return details, true, nil
}

// SetPlaceDetailsCache upserts a fresh Google Places Details response for a
// Google place ID.
func (s *Store) SetPlaceDetailsCache(ctx context.Context, googlePlaceID string, details json.RawMessage, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO place_details_cache (place_id, details, expires_at) VALUES ($1,$2,$3)
		ON CONFLICT (place_id) DO UPDATE SET details=EXCLUDED.details, expires_at=EXCLUDED.expires_at`,
		googlePlaceID, details, expiresAt)
	return err
}
