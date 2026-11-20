package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// FxRate returns the exchange rate finance set for currency.
func (s *Store) FxRate(ctx context.Context, currency string) (domain.FxRate, error) {
	var r domain.FxRate
	err := s.Pool.QueryRow(ctx, `SELECT currency, lkr_per_unit_micros, set_by, set_at FROM fx_rates WHERE currency=$1`, currency).
		Scan(&r.Currency, &r.LKRPerUnitMicros, &r.SetBy, &r.SetAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// FxRates lists every exchange rate set.
func (s *Store) FxRates(ctx context.Context) ([]domain.FxRate, error) {
	rows, err := s.Pool.Query(ctx, `SELECT currency, lkr_per_unit_micros, set_by, set_at FROM fx_rates ORDER BY currency`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.FxRate{}
	for rows.Next() {
		var r domain.FxRate
		if err := rows.Scan(&r.Currency, &r.LKRPerUnitMicros, &r.SetBy, &r.SetAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetFxRate stores a new rate for currency, stamped now.
func (s *Store) SetFxRate(ctx context.Context, currency string, lkrPerUnitMicros int64, setBy string) (domain.FxRate, error) {
	var r domain.FxRate
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO fx_rates (currency, lkr_per_unit_micros, set_by, set_at) VALUES ($1,$2,$3,now())
		ON CONFLICT (currency) DO UPDATE SET lkr_per_unit_micros=EXCLUDED.lkr_per_unit_micros, set_by=EXCLUDED.set_by, set_at=now()
		RETURNING currency, lkr_per_unit_micros, set_by, set_at`, currency, lkrPerUnitMicros, setBy).
		Scan(&r.Currency, &r.LKRPerUnitMicros, &r.SetBy, &r.SetAt)
	return r, err
}
