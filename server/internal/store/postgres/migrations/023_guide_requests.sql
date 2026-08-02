-- One row per POST /guide/chat call, success or failure — decoupled from
-- guide_messages (which only records successful turns) so a rate-limited
-- or errored request doesn't need a fake assistant message to be tracked.
-- Powers the admin Local Guide dashboard's reliability/usage stats.
CREATE TABLE guide_requests (
    id                  BIGSERIAL PRIMARY KEY,
    conversation_id     TEXT,
    traveller_id        TEXT NOT NULL,
    status              TEXT NOT NULL, -- 'ok' | 'rate_limited' | 'error'
    error_message       TEXT,
    latency_ms          INTEGER NOT NULL,
    tools_used          JSONB NOT NULL DEFAULT '[]',
    used_semantic_query BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX guide_requests_created_idx ON guide_requests (created_at DESC);
