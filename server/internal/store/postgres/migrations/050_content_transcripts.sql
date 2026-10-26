-- Transcripts of audio guides, written by the transcriber service
-- (transcriber/ — faster-whisper) after a publisher uploads the audio.
-- The service claims queued rows through /internal/transcripts/claim and
-- posts the result back; nothing else reads or writes this table.
CREATE TABLE content_transcripts (
    content_id   TEXT PRIMARY KEY REFERENCES content_items(id) ON DELETE CASCADE,
    -- The audio upload this transcript is for; a result for an older upload
    -- (replaced while it was being transcribed) is ignored.
    audio_path   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'queued'
                 CHECK (status IN ('queued', 'working', 'done', 'failed')),
    attempts     INT NOT NULL DEFAULT 0,
    error        TEXT NOT NULL DEFAULT '',
    -- working: when the claim lapses (the transcriber crashed); queued: not
    -- before (a retry's back-off). NULL = now.
    lease_until  TIMESTAMPTZ,
    language     TEXT NOT NULL DEFAULT '',
    duration_sec DOUBLE PRECISION NOT NULL DEFAULT 0,
    segments     JSONB NOT NULL DEFAULT '[]', -- [{start, end, text}], seconds
    text         TEXT NOT NULL DEFAULT '',    -- the segments joined, for search
    edited       BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX content_transcripts_queue_idx ON content_transcripts (status, lease_until);
