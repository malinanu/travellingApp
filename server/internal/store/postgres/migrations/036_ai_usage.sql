-- Which embedding model produced each place's vector. Vectors from
-- different models aren't comparable, so the embedding worker re-embeds any
-- row whose model differs from the configured GEMINI_EMBED_MODEL. Existing
-- vectors came from text-embedding-004 (now retired) and are cleared so they
-- are rebuilt with the current model.
ALTER TABLE places ADD COLUMN embedding_model TEXT;
UPDATE places SET embedding = NULL WHERE embedding IS NOT NULL;

-- One row per upstream Gemini call from any feature (Local Guide chat,
-- query/search embeddings, the embedding backfill worker, BR-certificate
-- OCR, inspiration extraction). guide_requests stays as the per-chat-turn
-- record; this is the per-API-call record that powers the admin dashboard's
-- token usage, quota and cross-feature error views.
CREATE TABLE ai_usage_events (
    id             BIGSERIAL PRIMARY KEY,
    feature        TEXT NOT NULL,
    kind           TEXT NOT NULL, -- 'chat' | 'embed' (which daily quota it counts against)
    model          TEXT NOT NULL,
    status         TEXT NOT NULL, -- 'ok' | 'rate_limited' | 'error'
    error_message  TEXT,
    input_tokens   INTEGER NOT NULL DEFAULT 0,
    output_tokens  INTEGER NOT NULL DEFAULT 0,
    tokens_estimated BOOLEAN NOT NULL DEFAULT false,
    items          INTEGER NOT NULL DEFAULT 1,
    latency_ms     INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ai_usage_events_created_idx ON ai_usage_events (created_at DESC);

-- Admin-entered daily limits the dashboard measures "left to use" against —
-- the Gemini API has no endpoint that reports remaining quota. Defaults are
-- conservative free-tier figures; the admin edits them to match their
-- project's actual tier in AI Studio.
INSERT INTO platform_settings (key, value) VALUES
    ('ai_quota', '{"chatRequestsPerDay": 250, "chatTokensPerDay": 1000000, "embedRequestsPerDay": 1000}')
ON CONFLICT (key) DO NOTHING;
