package main

import (
	"context"
	"log"

	"github.com/spotto/server/internal/config"
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

	for _, c := range seed.InitialCategories {
		// Use UPSERT so it doesn't fail if they already exist
		if _, err := store.Pool.Exec(ctx, `
			INSERT INTO vendor_categories (id, label, segment, icon, relevant_vibe_dimensions, booking_type)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				label = EXCLUDED.label,
				segment = EXCLUDED.segment,
				icon = EXCLUDED.icon,
				relevant_vibe_dimensions = EXCLUDED.relevant_vibe_dimensions,
				booking_type = EXCLUDED.booking_type
		`, c.ID, c.Label, c.Segment, c.Icon, c.RelevantVibeDimensions, c.BookingType); err != nil {
			log.Fatalf("failed to upsert category %s: %v", c.ID, err)
		}
	}

	// Delete unwanted categories
	if _, err := store.Pool.Exec(ctx, `DELETE FROM vendor_categories WHERE id IN ('general', 'uncategorized')`); err != nil {
		log.Printf("warning: failed to delete general/uncategorized: %v", err)
	}

	log.Println("Successfully seeded all 50 vendor categories.")
}
