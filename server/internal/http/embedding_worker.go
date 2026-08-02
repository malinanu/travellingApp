package httpapi

// The embedding worker backfills semantic-search vectors for places, off
// the request path — same ticker-driven, Postgres-state-not-in-memory shape
// as thumbnail_worker.go/weather_scheduler.go/holiday_scheduler.go/
// br_cert_scheduler.go. Places start with embedding = NULL and this sweep
// fills them in a few at a time via one batched OpenRouter call per tick.

import (
	"context"
	"log"
	"time"

	"github.com/spotto/server/internal/openrouter"
	"github.com/spotto/server/internal/recommend"
)

// embeddingWorkerInterval — no tight latency requirement here (unlike
// thumbnails, nothing in the UI is waiting on a place's embedding to
// appear), so this paces gently against the free-tier embedding model's
// rate limits.
const embeddingWorkerInterval = 30 * time.Second

// embeddingBatchSize caps how many places one tick embeds; a bigger
// backlog just spans more ticks.
const embeddingBatchSize = 20

// StartEmbeddingWorker blocks until ctx is cancelled; call with `go`.
func (s *Server) StartEmbeddingWorker(ctx context.Context) {
	s.runEmbeddingSweep(ctx)
	ticker := time.NewTicker(embeddingWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runEmbeddingSweep(ctx)
		}
	}
}

func (s *Server) runEmbeddingSweep(ctx context.Context) {
	if s.OpenRouterAPIKey == "" {
		return // local guide unconfigured
	}
	places, err := s.Store.PlacesMissingEmbedding(ctx, embeddingBatchSize)
	if err != nil {
		log.Printf("embedding worker: list pending: %v", err)
		return
	}
	if len(places) == 0 {
		return
	}

	texts := make([]string, len(places))
	for i, p := range places {
		texts[i] = recommend.SummarizeForEmbedding(p)
	}

	embeddings, err := openrouter.Embed(ctx, s.OpenRouterAPIKey, texts)
	if err != nil {
		// Transient (rate limit/network) — left NULL, picked up again next
		// tick, same as the thumbnail worker's transient-failure handling.
		log.Printf("embedding worker: embed batch: %v", err)
		return
	}

	done := 0
	for i, p := range places {
		if err := s.Store.SetPlaceEmbedding(ctx, p.ID, embeddings[i]); err != nil {
			log.Printf("embedding worker: store %s: %v", p.ID, err)
			continue
		}
		done++
	}
	log.Printf("embedding worker: %d/%d places embedded", done, len(places))
}
