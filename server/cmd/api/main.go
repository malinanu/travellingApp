// Command api runs the Spotto HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

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
	go srv.StartWeatherScheduler(ctx)
	go srv.StartBRCertScheduler(ctx)
	go srv.StartBookingReminderScheduler(ctx)
	go srv.StartVendorReportScheduler(ctx)
	go srv.StartThumbnailWorker(ctx)
	go srv.StartEmbeddingWorker(ctx)

	// Explicit timeouts rather than the zero-value (unbounded) defaults —
	// ReadHeaderTimeout guards against slow-header (slowloris-style)
	// connections, ReadTimeout/WriteTimeout bound the rest of a request's
	// lifecycle. Uploads go straight to R2 via presigned URLs rather than
	// streaming through this server, so 30s is generous, not tight.
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(cfg),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Printf("spotto api listening on :%s", cfg.Port)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
