-- Travel content for travellers: audio guides and audiobooks, blog posts,
-- travel guides, itineraries and short tips, written in the Command Center
-- and read in the app and on the website.
--
-- Replaces the old audio_guides table, which only ever held generated
-- sample rows (no audio files, invented play counts).
DROP TABLE IF EXISTS audio_guides;

-- An admin or super-admin can let any account write content. Publishers
-- sign in to the Command Center and only see their own posts, which an
-- admin reviews before they go live.
ALTER TABLE users ADD COLUMN publisher BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE content_items (
    id            TEXT PRIMARY KEY,
    type          TEXT NOT NULL CHECK (type IN ('audio', 'blog', 'guide', 'itinerary', 'tip')),
    slug          TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL DEFAULT '',
    summary       TEXT NOT NULL DEFAULT '',
    -- Markdown; never rendered as raw HTML.
    body          TEXT NOT NULL DEFAULT '',
    -- "/media/raw/<key>" paths, signed when served (see signMediaPath).
    cover_path    TEXT NOT NULL DEFAULT '',
    audio_path    TEXT NOT NULL DEFAULT '',
    audio_seconds INT NOT NULL DEFAULT 0,
    chapters      JSONB NOT NULL DEFAULT '[]', -- [{title, startSec}]
    place_ids     JSONB NOT NULL DEFAULT '[]', -- a guide's linked places
    days          JSONB NOT NULL DEFAULT '[]', -- itinerary: [{title, stops: [{placeId, name, lat, lng, time, note}]}]
    tags          TEXT[] NOT NULL DEFAULT '{}',
    region_id     TEXT REFERENCES regions(id) ON DELETE SET NULL,
    language      TEXT NOT NULL DEFAULT 'en',
    author_id     TEXT REFERENCES users(id) ON DELETE SET NULL,
    status        TEXT NOT NULL DEFAULT 'draft'
                  CHECK (status IN ('draft', 'in_review', 'published', 'rejected', 'archived')),
    review_note   TEXT NOT NULL DEFAULT '',
    reviewed_by   TEXT REFERENCES users(id) ON DELETE SET NULL,
    submitted_at  TIMESTAMPTZ,
    published_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    views         INT NOT NULL DEFAULT 0,
    plays         INT NOT NULL DEFAULT 0
);

CREATE INDEX content_items_feed_idx ON content_items (status, type, published_at DESC);
CREATE INDEX content_items_author_idx ON content_items (author_id);
