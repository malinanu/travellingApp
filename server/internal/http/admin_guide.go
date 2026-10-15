package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
)

// handleAdminGuideStats powers the Local Guide dashboard's stat-card grid
// and tool-usage breakdown â€” usage/reliability over the trailing 7 days
// plus the embedding backfill's overall progress.
func (s *Server) handleAdminGuideStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.GuideStats(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	usage, err := s.Store.GuideToolUsage(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	limit, err := s.Store.GetGuideRateLimitRPM(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	aiUsage, err := s.Store.AIUsageByFeature(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	daily, err := s.Store.AIUsageDaily(r.Context(), 14)
	if err != nil {
		storeErr(w, err)
		return
	}
	quota, err := s.Store.GetAIQuota(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	used, err := s.Store.AIQuotaUsageToday(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	embeddings, err := s.Store.EmbeddingStatsFor(r.Context(), s.Gemini.EmbedModel)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"stats":        stats,
		"toolUsage":    usage,
		"rateLimitRPM": limit,
		"configured":   s.GeminiAPIKey != "",
		"models": map[string]string{
			"chat":  s.Gemini.ChatModel,
			"embed": s.Gemini.EmbedModel,
		},
		"aiUsage":      aiUsage,
		"daily":        daily,
		"quota":        quota,
		"quotaUsed":    used,
		"embeddings":   embeddings,
		"embedWorker":  s.embedStatus.snapshot(),
	})
}

// handleAdminGuideUsers lists the most recently active Local Guide users
// for the dashboard's browsable list, along with their analytics.
func (s *Server) handleAdminGuideUsers(w http.ResponseWriter, r *http.Request) {
	const limit = 50
	users, err := s.Store.RecentGuideUsers(r.Context(), limit)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// handleAdminGuideErrors returns the most recent failed Gemini calls from
// any feature (chat, embeddings, BR-cert OCR, inspiration) with the real
// upstream error text — the traveller only ever sees a generic message, so
// this is the diagnostic surface for actually finding out why (auth, quota,
// a retired model id, etc.) without shell/DB access.
func (s *Server) handleAdminGuideErrors(w http.ResponseWriter, r *http.Request) {
	const limit = 30
	errs, err := s.Store.RecentAIErrors(r.Context(), limit)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, errs)
}

// handleAdminGuideConversationDetail returns one conversation's full
// transcript â€” an admin can inspect any conversation, unlike the
// traveller-facing endpoint which enforces ownership.
func (s *Server) handleAdminGuideConversationDetail(w http.ResponseWriter, r *http.Request) {
	messages, err := s.Store.GuideMessagesByConversation(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

// handleAdminGuideUserMessages returns all guide messages for a given traveller
func (s *Server) handleAdminGuideUserMessages(w http.ResponseWriter, r *http.Request) {
	messages, err := s.Store.GuideMessagesByTraveller(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, messages)
}

func (s *Server) handleAdminPutGuideRateLimit(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RateLimitRPM int `json:"rateLimitRPM"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.RateLimitRPM < 1 {
		writeErr(w, http.StatusBadRequest, "rate limit must be >= 1")
		return
	}
	if err := s.Store.SetGuideRateLimitRPM(r.Context(), body.RateLimitRPM); err != nil {
		storeErr(w, err)
		return
	}
	s.guideLimit.setMax(body.RateLimitRPM)
	writeJSON(w, http.StatusOK, map[string]int{"rateLimitRPM": body.RateLimitRPM})
}

func (s *Server) handleAdminPutAIQuota(w http.ResponseWriter, r *http.Request) {
	var body domain.AIQuota
	if !readJSON(w, r, &body) {
		return
	}
	if body.ChatRequestsPerDay < 1 || body.ChatTokensPerDay < 1 || body.EmbedRequestsPerDay < 1 {
		writeErr(w, http.StatusBadRequest, "every quota limit must be at least 1")
		return
	}
	if err := s.Store.SetAIQuota(r.Context(), body); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, body)
}

// handleAdminGuideHealth makes one tiny live chat call and one embed call
// against the configured models and reports each result with Google's own
// error text — the quickest way to confirm on staging that the key and
// model ids actually work, without sending a real Local Guide message.
func (s *Server) handleAdminGuideHealth(w http.ResponseWriter, r *http.Request) {
	if s.GeminiAPIKey == "" {
		writeErr(w, http.StatusNotImplemented, "GEMINI_API_KEY is not set on the server")
		return
	}
	check := func(model string, fn func() error) map[string]any {
		start := time.Now()
		err := fn()
		out := map[string]any{
			"model":     model,
			"ok":        err == nil,
			"latencyMs": time.Since(start).Milliseconds(),
		}
		if err != nil {
			out["error"] = err.Error()
		}
		return out
	}
	chat := check(s.Gemini.ChatModel, func() error {
		msg, _, err := s.aiComplete(r.Context(), aiFeatureHealthCheck, []gemini.Message{
			{Role: "user", Content: "Reply with the single word: ok"},
		}, nil)
		if err == nil && strings.TrimSpace(msg.Content) == "" {
			return errors.New("model returned an empty reply")
		}
		return err
	})
	embed := check(s.Gemini.EmbedModel, func() error {
		vecs, err := s.aiEmbed(r.Context(), aiFeatureHealthCheck, []string{"quiet beach cafe"})
		if err == nil && (len(vecs) != 1 || len(vecs[0]) == 0) {
			return errors.New("embedding response was empty")
		}
		return err
	})
	writeJSON(w, http.StatusOK, map[string]any{"chat": chat, "embed": embed})
}

// handleAdminGuideReembed clears every stored place embedding so the
// embedding worker rebuilds them all with the current model, then kicks off
// a sweep immediately rather than waiting for the next tick.
func (s *Server) handleAdminGuideReembed(w http.ResponseWriter, r *http.Request) {
	n, err := s.Store.ClearAllEmbeddings(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	go s.runEmbeddingSweep(context.WithoutCancel(r.Context()))
	writeJSON(w, http.StatusOK, map[string]any{"cleared": n})
}
