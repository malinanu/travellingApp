// Command gendata wipes the database and populates it with a large,
// randomized dataset for manual testing — unlike cmd/seed's small fixed
// demo, every run generates different names/emails/businesses/reviews.
package main

import (
	"context"
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/gendata"
	"github.com/spotto/server/internal/store/postgres"
)

func main() {
	cfg := config.Load()
	if cfg.IsProduction() {
		log.Fatal("refusing to generate data: APP_ENV=production, and this wipes every table and adds logins that share one password")
	}
	ctx := context.Background()

	store, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	sum, err := gendata.Run(ctx, store)
	if err != nil {
		log.Fatalf("gendata: %v", err)
	}

	log.Printf("generated: %d travellers, %d vendors, %d places, %d offers, %d claims, %d reviews, %d groups, %d subscriptions, %d audio guides",
		sum.Travellers, sum.Vendors, sum.Places, sum.Offers, sum.Claims, sum.Reviews, sum.Groups, sum.Subscriptions, sum.AudioGuides)
	log.Println("sample logins (all accounts share this run's password):")
	for _, line := range sum.SampleLogins {
		log.Println("  " + line)
	}
}
