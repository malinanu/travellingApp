// Command api runs the Spotto HTTP server.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
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

	srv := httpapi.New(store, auth.NewTokens(cfg.JWTSecret), cfg)
	go srv.StartHolidayScheduler(ctx)
	log.Printf("spotto api listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, srv.Router(cfg)); err != nil {
		log.Fatal(err)
	}
}
