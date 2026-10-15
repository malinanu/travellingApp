package httpapi

// Every upstream Gemini call goes through one of these wrappers so it lands
// in ai_usage_events — the admin Local Guide dashboard's token usage, quota
// and cross-feature error views are built from that table. Recording is
// best-effort, same convention as recordGuideRequest: a failed insert never
// affects the caller.

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
)

// Feature names recorded in ai_usage_events.feature.
const (
	aiFeatureGuideChat   = "guide_chat"
	aiFeatureGuideEmbed  = "guide_query_embed"
	aiFeatureSearchEmbed = "search_embed"
	aiFeatureEmbedWorker = "embed_worker"
	aiFeatureBRCert      = "br_cert"
	aiFeatureInspiration = "inspiration"
	aiFeatureHealthCheck = "health_check"
)

func aiStatus(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, gemini.ErrRateLimited):
		return "rate_limited"
	default:
		return "error"
	}
}

func (s *Server) recordAIUsage(ctx context.Context, feature, kind, model string, items int, usage *gemini.Usage, start time.Time, err error) {
	e := domain.AIUsageEvent{
		Feature:   feature,
		Kind:      kind,
		Model:     model,
		Status:    aiStatus(err),
		Items:     items,
		LatencyMs: int(time.Since(start).Milliseconds()),
	}
	if err != nil {
		e.ErrorMessage = err.Error()
	}
	if usage != nil {
		e.InputTokens = usage.PromptTokens
		e.OutputTokens = usage.CompletionTokens
		// Gemini 3's thinking tokens appear only in the total, but are
		// billed (and rate-limited) as output — count them as such.
		if thinking := usage.TotalTokens - usage.PromptTokens - usage.CompletionTokens; thinking > 0 {
			e.OutputTokens += thinking
		}
		e.TokensEstimated = usage.Estimated
	}
	// WithoutCancel: a traveller closing the app mid-request shouldn't drop
	// the record of a call Google already counted against the quota.
	if err := s.Store.InsertAIUsage(context.WithoutCancel(ctx), e); err != nil {
		log.Printf("ai usage: record %s: %v", feature, err)
	}
}

func (s *Server) aiComplete(ctx context.Context, feature string, messages []gemini.Message, tools []gemini.Tool) (*gemini.Message, *gemini.Usage, error) {
	start := time.Now()
	msg, usage, err := s.Gemini.Complete(ctx, messages, tools)
	s.recordAIUsage(ctx, feature, "chat", s.Gemini.ChatModel, 1, usage, start, err)
	return msg, usage, err
}

func (s *Server) aiEmbed(ctx context.Context, feature string, inputs []string) ([][]float64, error) {
	start := time.Now()
	vecs, usage, err := s.Gemini.Embed(ctx, inputs)
	s.recordAIUsage(ctx, feature, "embed", s.Gemini.EmbedModel, len(inputs), usage, start, err)
	return vecs, err
}

func (s *Server) aiExtractCertNumber(ctx context.Context, data []byte, contentType string) (string, error) {
	start := time.Now()
	num, usage, err := s.Gemini.ExtractCertNumber(ctx, data, contentType)
	s.recordAIUsage(ctx, aiFeatureBRCert, "chat", s.Gemini.ChatModel, 1, usage, start, err)
	return num, err
}

// embedWorkerStatus is the embedding worker's last-sweep snapshot. In
// memory only — it resets on deploy, which is fine for a "is it running"
// indicator; the durable progress number comes from the places table.
type embedWorkerStatus struct {
	mu          sync.Mutex
	lastRunAt   time.Time
	lastError   string
	lastEmbedded int
}

func (st *embedWorkerStatus) set(embedded int, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.lastRunAt = time.Now()
	st.lastEmbedded = embedded
	st.lastError = ""
	if err != nil {
		st.lastError = err.Error()
	}
}

func (st *embedWorkerStatus) snapshot() map[string]any {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := map[string]any{"lastError": st.lastError, "lastEmbedded": st.lastEmbedded}
	if !st.lastRunAt.IsZero() {
		out["lastRunAt"] = st.lastRunAt
	}
	return out
}
