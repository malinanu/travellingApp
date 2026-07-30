package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// GetWeatherCache returns a still-valid cached Tomorrow.io response for
// cellKey, if one exists — backs internal/weather's in-memory cache with
// something that survives a process restart (see migrations/013_weather_
// cache.sql for why that matters against the free-tier rate limit).
func (s *Store) GetWeatherCache(ctx context.Context, cellKey string) (json.RawMessage, bool, error) {
	var snapshot json.RawMessage
	var expiresAt time.Time
	err := s.Pool.QueryRow(ctx,
		`SELECT snapshot, expires_at FROM weather_cache WHERE cell_key=$1`, cellKey,
	).Scan(&snapshot, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Now().After(expiresAt) {
		return nil, false, nil
	}
	return snapshot, true, nil
}

// SetWeatherCache upserts a fresh Tomorrow.io response for cellKey.
func (s *Store) SetWeatherCache(ctx context.Context, cellKey string, snapshot json.RawMessage, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO weather_cache (cell_key, snapshot, expires_at) VALUES ($1,$2,$3)
		ON CONFLICT (cell_key) DO UPDATE SET snapshot=EXCLUDED.snapshot, expires_at=EXCLUDED.expires_at`,
		cellKey, snapshot, expiresAt)
	return err
}

// CellCoord is a representative lat/lng for one coarse (~55km, 0.5°)
// weather cache cell.
type CellCoord struct {
	Lat float64
	Lng float64
}

// DistinctVendorCells returns one representative point per distinct cache
// cell that the real vendor dataset actually spans — used by the weather
// scheduler (internal/http/weather_scheduler.go) to proactively warm every
// location travellers can realistically be near, rather than guessing at a
// fixed region list. vendors.lat/lng are real per-place columns set at
// import time (server/cmd/importsrilanka), not the seeded admin-only
// regions table.
func (s *Store) DistinctVendorCells(ctx context.Context) ([]CellCoord, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT ROUND(CAST(lat AS numeric) * 2) / 2 AS cell_lat,
		                ROUND(CAST(lng AS numeric) * 2) / 2 AS cell_lng
		FROM vendors
		WHERE lat != 0 OR lng != 0`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CellCoord
	for rows.Next() {
		var c CellCoord
		if err := rows.Scan(&c.Lat, &c.Lng); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
