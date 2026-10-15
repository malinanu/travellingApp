package httpapi

// The embedding worker backfills semantic-search vectors for places, off
// the request path — same ticker-driven, Postgres-state-not-in-memory shape
// as thumbnail_worker.go/weather_scheduler.go/holiday_scheduler.go/
// br_cert_scheduler.go. A place needs (re-)embedding when its embedding is
// NULL or was produced by a different model than GEMINI_EMBED_MODEL; this
// sweep fills them in a batch at a time via one batchEmbedContents call.

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/recommend"
)

// embeddingWorkerInterval — nothing in the UI waits on a place's embedding,
// so this paces gently. Each embedded item counts against the embedding
// model's daily request quota (see the admin dashboard's quota bars), and a
// few hundred places at 20 per 2 minutes backfill well inside an hour.
const embeddingWorkerInterval = 2 * time.Minute

// embeddingBatchSize caps how many places one tick embeds; a bigger
// backlog just spans more ticks.
const embeddingBatchSize = 20

// embeddingRateLimitPause is how long the worker stands down after Google
// answers 429 — retrying every tick would only add more rejected calls.
const embeddingRateLimitPause = time.Hour

// embeddingQueryReserve is the share of the daily embed quota the backfill
// leaves untouched, so traveller searches and Local Guide queries (which
// embed the query text live) still work on a day the backlog is large.
func embeddingQueryReserve(perDay int) int {
	return max(50, perDay/5)
}

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
	if s.GeminiAPIKey == "" {
		return // local guide unconfigured
	}
	if until := s.embedStatus.pausedUntil(); time.Now().Before(until) {
		return
	}
	batch := embeddingBatchSize
	if quota, err := s.Store.GetAIQuota(ctx); err == nil && quota.EmbedRequestsPerDay > 0 {
		used, err := s.Store.AIQuotaUsageToday(ctx)
		if err == nil {
			batch = min(batch, quota.EmbedRequestsPerDay-embeddingQueryReserve(quota.EmbedRequestsPerDay)-used.EmbedRequestsToday)
		}
	}
	if batch <= 0 {
		return // today's backfill share is spent; resumes after the Pacific-midnight reset
	}
	model := s.Gemini.EmbedModel
	places, err := s.Store.PlacesMissingEmbedding(ctx, model, batch)
	if err != nil {
		log.Printf("embedding worker: list pending: %v", err)
		s.embedStatus.set(0, err)
		return
	}
	if len(places) == 0 {
		return
	}

	texts := make([]string, len(places))
	for i, p := range places {
		texts[i] = recommend.SummarizeForEmbedding(p)
	}

	embeddings, err := s.aiEmbed(ctx, aiFeatureEmbedWorker, texts)
	if err != nil {
		s.embedStatus.set(0, err)
		if errors.Is(err, gemini.ErrRateLimited) {
			s.embedStatus.pause(embeddingRateLimitPause)
		}
		// Transient (rate limit/network) â€” left NULL, picked up again next
		// tick, same as the thumbnail worker's transient-failure handling.
		log.Printf("embedding worker: embed batch: %v", err)
		return
	}

	done := 0
	for i, p := range places {
		if err := s.Store.SetPlaceEmbedding(ctx, p.ID, embeddings[i], model); err != nil {
			log.Printf("embedding worker: store %s: %v", p.ID, err)
			continue
		}
		done++
	}
	s.embedStatus.set(done, nil)
	log.Printf("embedding worker: %d/%d places embedded", done, len(places))
}
