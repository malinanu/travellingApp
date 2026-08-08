// Command seed wipes the database and reloads the demo dataset.
package main

import (
	"context"
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/seed"
	"github.com/spotto/server/internal/store/postgres"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	store, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Load dynamic vendor categories into memory cache.
	cats, err := store.GetVendorCategories(ctx)
	if err != nil {
		log.Fatalf("failed to load vendor categories: %v", err)
	}
	domain.InitVendorCategories(cats)
	if err := seed.Run(ctx, store); err != nil {
		log.Fatalf("seed: %v", err)
	}
	log.Println("seeded demo data — accounts: lena@spotto.dev / vendor@driftwood.dev / admin@spotto.dev (password spotto123)")
}
