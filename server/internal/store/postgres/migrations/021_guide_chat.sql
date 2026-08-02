-- Local Guide chat history — see internal/http/guide.go. Only user-visible
-- turns are stored (not intermediate tool-call/tool-result exchanges); each
-- new user message re-derives whatever tool calls it needs from the prior
-- plain-text turns, so there's nothing else to replay.
CREATE TABLE guide_conversations (
    id           TEXT PRIMARY KEY,
    traveller_id TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX guide_conversations_traveller_idx ON guide_conversations (traveller_id, updated_at DESC);

CREATE TABLE guide_messages (
    id              BIGSERIAL PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES guide_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL, -- 'user' | 'assistant'
    content         TEXT NOT NULL,
    place_ids       JSONB NOT NULL DEFAULT '[]', -- places referenced/recommended in this turn
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX guide_messages_conversation_idx ON guide_messages (conversation_id, created_at ASC);
