// Command api runs the Spotto HTTP server.
package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/seed"
	"github.com/spotto/server/internal/store/postgres"

	"github.com/spotto/server/internal/domain"
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

	go feedGenerator(ctx, store)

	srv := httpapi.New(store, auth.NewTokens(cfg.JWTSecret))
	log.Printf("spotto api listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, srv.Router(cfg)); err != nil {
		log.Fatal(err)
	}
}

// feedGenerator inserts a templated feed item every ~3s so the admin content
// monitor has something to stream, pruning to ~200 rows.
func feedGenerator(ctx context.Context, store *postgres.Store) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	n := 0
	for range ticker.C {
		tpl := seed.FeedTemplates[rand.Intn(len(seed.FeedTemplates))]
		item := domain.FeedItem{
			Vendor:    tpl.Vendor,
			Town:      tpl.Town,
			Type:      tpl.Type,
			Text:      tpl.Text,
			Status:    domain.FeedStatus(tpl.Status),
			Reason:    tpl.Reason,
			CreatedAt: time.Now(),
		}
		if _, err := store.InsertFeedItem(ctx, item); err != nil {
			log.Printf("feed generator: %v", err)
			continue
		}
		n++
		if n%20 == 0 {
			if err := store.PruneFeed(ctx, 200); err != nil {
				log.Printf("feed prune: %v", err)
			}
		}
	}
}
