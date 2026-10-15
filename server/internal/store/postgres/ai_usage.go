package postgres

// Gemini usage accounting for the admin Local Guide dashboard — every
// upstream AI call lands in ai_usage_events (see migrations/036_ai_usage.sql).

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// pacificDayStart is the start of the current Pacific-time day — Google
// resets Gemini's daily quotas at midnight Pacific, so "today" on the
// dashboard uses the same boundary.
const pacificDayStart = `(date_trunc('day', now() AT TIME ZONE 'America/Los_Angeles') AT TIME ZONE 'America/Los_Angeles')`

func (s *Store) InsertAIUsage(ctx context.Context, e domain.AIUsageEvent) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO ai_usage_events
			(feature, kind, model, status, error_message, input_tokens, output_tokens, tokens_estimated, items, latency_ms)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,$10)`,
		e.Feature, e.Kind, e.Model, e.Status, e.ErrorMessage,
		e.InputTokens, e.OutputTokens, e.TokensEstimated, e.Items, e.LatencyMs)
	return err
}

// AIUsageByFeature aggregates the trailing 30 days per feature+model, with
// today/7d/30d columns side by side via FILTER.
func (s *Store) AIUsageByFeature(ctx context.Context) ([]domain.AIUsageRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT feature, kind, model,
			count(*) FILTER (WHERE created_at >= `+pacificDayStart+`),
			count(*) FILTER (WHERE created_at >= now() - interval '7 days'),
			count(*),
			count(*) FILTER (WHERE created_at >= now() - interval '7 days' AND status = 'error'),
			count(*) FILTER (WHERE created_at >= now() - interval '7 days' AND status = 'rate_limited'),
			COALESCE(sum(input_tokens)  FILTER (WHERE created_at >= `+pacificDayStart+`), 0),
			COALESCE(sum(output_tokens) FILTER (WHERE created_at >= `+pacificDayStart+`), 0),
			COALESCE(sum(input_tokens)  FILTER (WHERE created_at >= now() - interval '7 days'), 0),
			COALESCE(sum(output_tokens) FILTER (WHERE created_at >= now() - interval '7 days'), 0),
			COALESCE(sum(input_tokens), 0),
			COALESCE(sum(output_tokens), 0),
			bool_or(tokens_estimated)
		FROM ai_usage_events
		WHERE created_at >= now() - interval '30 days'
		GROUP BY feature, kind, model
		ORDER BY count(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AIUsageRow{}
	for rows.Next() {
		var r domain.AIUsageRow
		if err := rows.Scan(&r.Feature, &r.Kind, &r.Model,
			&r.RequestsToday, &r.Requests7d, &r.Requests30d, &r.Errors7d, &r.RateLimited7d,
			&r.InputTokensToday, &r.OutputTokensToday, &r.InputTokens7d, &r.OutputTokens7d,
			&r.InputTokens30d, &r.OutputTokens30d, &r.TokensEstimated); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AIUsageDaily returns one point per Pacific-time day for the last `days`
// days, zero-filled so the chart has no gaps.
func (s *Store) AIUsageDaily(ctx context.Context, days int) ([]domain.AIDailyPoint, error) {
	rows, err := s.Pool.Query(ctx, `
		WITH d AS (
			SELECT generate_series(
				(now() AT TIME ZONE 'America/Los_Angeles')::date - ($1::int - 1),
				(now() AT TIME ZONE 'America/Los_Angeles')::date,
				interval '1 day')::date AS day
		)
		SELECT to_char(d.day, 'YYYY-MM-DD'),
			count(e.id),
			COALESCE(sum(e.input_tokens), 0),
			COALESCE(sum(e.output_tokens), 0)
		FROM d
		LEFT JOIN ai_usage_events e
			ON (e.created_at AT TIME ZONE 'America/Los_Angeles')::date = d.day
		GROUP BY d.day
		ORDER BY d.day`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AIDailyPoint{}
	for rows.Next() {
		var p domain.AIDailyPoint
		if err := rows.Scan(&p.Day, &p.Requests, &p.InputTokens, &p.OutputTokens); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AIQuotaUsageToday sums today's (Pacific) consumption per quota bucket.
// Rate-limited calls are excluded — Google didn't serve them.
func (s *Store) AIQuotaUsageToday(ctx context.Context) (domain.AIQuotaUsage, error) {
	var u domain.AIQuotaUsage
	err := s.Pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE kind = 'chat'),
			COALESCE(sum(input_tokens + output_tokens) FILTER (WHERE kind = 'chat'), 0),
			COALESCE(sum(items) FILTER (WHERE kind = 'embed'), 0)
		FROM ai_usage_events
		WHERE created_at >= `+pacificDayStart+` AND status <> 'rate_limited'`).
		Scan(&u.ChatRequestsToday, &u.ChatTokensToday, &u.EmbedRequestsToday)
	return u, err
}

// RecentAIErrors lists the most recent failed Gemini calls across features.
func (s *Store) RecentAIErrors(ctx context.Context, limit int) ([]domain.AIErrorEntry, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT feature, model, status, COALESCE(error_message, ''), created_at
		FROM ai_usage_events
		WHERE status <> 'ok'
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AIErrorEntry{}
	for rows.Next() {
		var e domain.AIErrorEntry
		if err := rows.Scan(&e.Feature, &e.Model, &e.Status, &e.ErrorMessage, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// defaultAIQuota mirrors the seed row in migrations/036_ai_usage.sql.
var defaultAIQuota = domain.AIQuota{ChatRequestsPerDay: 250, ChatTokensPerDay: 1_000_000, EmbedRequestsPerDay: 1000}

func (s *Store) GetAIQuota(ctx context.Context) (domain.AIQuota, error) {
	var raw []byte
	err := s.Pool.QueryRow(ctx, `SELECT value FROM platform_settings WHERE key = 'ai_quota'`).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return defaultAIQuota, nil
	}
	if err != nil {
		return defaultAIQuota, err
	}
	q := defaultAIQuota
	if err := json.Unmarshal(raw, &q); err != nil {
		return defaultAIQuota, err
	}
	return q, nil
}

func (s *Store) SetAIQuota(ctx context.Context, q domain.AIQuota) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO platform_settings (key, value) VALUES ('ai_quota', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, jsonb(q))
	return err
}

// EmbeddingStatsFor reports backfill progress against the given (current)
// embedding model.
func (s *Store) EmbeddingStatsFor(ctx context.Context, model string) (domain.EmbeddingStats, error) {
	st := domain.EmbeddingStats{Model: model}
	err := s.Pool.QueryRow(ctx, `
		SELECT count(*),
			count(*) FILTER (WHERE embedding IS NOT NULL AND embedding_model = $1)
		FROM places`, model).Scan(&st.Total, &st.Embedded)
	st.Pending = st.Total - st.Embedded
	return st, err
}

// ClearAllEmbeddings forces the embedding worker to rebuild every place's
// vector on its next sweeps.
func (s *Store) ClearAllEmbeddings(ctx context.Context) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `UPDATE places SET embedding = NULL, embedding_model = NULL WHERE embedding IS NOT NULL`)
	return tag.RowsAffected(), err
}
