// Command importspots wipes the places table and reloads it from the
// real-world dataset embedded in internal/importspots/spots.json. Runs
// automatically on every container boot — see server/docker-entrypoint.sh.
package main

import (
	"context"
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/importspots"
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
	if err := importspots.Run(ctx, store); err != nil {
		log.Fatalf("importspots: %v", err)
	}
}
