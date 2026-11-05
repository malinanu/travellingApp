package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spotto/server/internal/domain"
)

var (
	pkgMemMu  sync.RWMutex
	pkgMemCfg *domain.PackagesConfig
)

// GetVendorPackages returns the vendor subscription packages and feature pool configuration.
// If not yet saved in platform_settings, it returns the standard default packages ($3.99, $11.99, $22.99).
func (s *Store) GetVendorPackages(ctx context.Context) (domain.PackagesConfig, error) {
	if s == nil || s.Pool == nil {
		pkgMemMu.RLock()
		defer pkgMemMu.RUnlock()
		if pkgMemCfg != nil {
			return *pkgMemCfg, nil
		}
		return domain.DefaultPackagesConfig(), nil
	}

	var val string
	err := s.Pool.QueryRow(ctx, `SELECT value FROM platform_settings WHERE key = 'vendor_packages'`).Scan(&val)
	if err == pgx.ErrNoRows {
		def := domain.DefaultPackagesConfig()
		return def, nil
	}
	if err != nil {
		pkgMemMu.RLock()
		defer pkgMemMu.RUnlock()
		if pkgMemCfg != nil {
			return *pkgMemCfg, nil
		}
		return domain.DefaultPackagesConfig(), err
	}

	var cfg domain.PackagesConfig
	if err := json.Unmarshal([]byte(val), &cfg); err != nil {
		return domain.DefaultPackagesConfig(), nil
	}

	// Safety check: ensure packages array is not empty
	if len(cfg.Packages) == 0 {
		return domain.DefaultPackagesConfig(), nil
	}

	return cfg, nil
}

// SaveVendorPackages updates the vendor subscription packages and master feature pool in platform_settings.
func (s *Store) SaveVendorPackages(ctx context.Context, cfg domain.PackagesConfig) error {
	cfg.UpdatedAt = time.Now()

	pkgMemMu.Lock()
	pkgMemCfg = &cfg
	pkgMemMu.Unlock()

	if s == nil || s.Pool == nil {
		return nil
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	_, err = s.Pool.Exec(ctx, `
		INSERT INTO platform_settings (key, value) VALUES ('vendor_packages', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, b)
	return err
}
