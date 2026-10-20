// Command api runs the Spotto HTTP server.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
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

	if cfg.AdminEmail != "" {
		if err := seed.EnsureAdmin(ctx, store, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatalf("ensure admin account: %v", err)
		}
		log.Printf("admin account %s is ready", cfg.AdminEmail)
	}
	// The demo and import logins are public knowledge (README, public
	// dataset), so production never lets them in, even on a DB that was
	// seeded once.
	if cfg.IsProduction() {
		n, err := seed.LockDemoAccounts(ctx, store, cfg.AdminEmail)
		if err != nil {
			log.Fatalf("lock demo accounts: %v", err)
		}
		if n > 0 {
			log.Printf("production: locked %d demo/import accounts", n)
		}
	}

	// Make sure the full category taxonomy exists (insert-only, never
	// overwrites admin edits), then load it into the in-memory cache.
	if added, err := store.EnsureVendorCategories(ctx, seed.InitialCategories); err != nil {
		log.Fatalf("ensure vendor categories: %v", err)
	} else if added > 0 {
		log.Printf("added %d missing vendor categories", added)
	}
	cats, err := store.GetVendorCategories(ctx)
	if err != nil {
		log.Fatalf("failed to load vendor categories: %v", err)
	}
	domain.InitVendorCategories(cats)

	srv := httpapi.New(store, auth.NewTokens(cfg.JWTSecret), cfg)
	go srv.StartHolidayScheduler(ctx)
	go srv.StartWeatherScheduler(ctx)
	go srv.StartBRCertScheduler(ctx)
	go srv.StartBookingReminderScheduler(ctx)
	go srv.StartVendorReportScheduler(ctx)
	go srv.StartThumbnailWorker(ctx)
	go srv.StartEmbeddingWorker(ctx)
	go srv.StartInspirationWorker(ctx)

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
