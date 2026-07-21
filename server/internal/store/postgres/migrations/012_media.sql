-- Media uploads (Cloudflare R2). One row per uploaded object; the object
-- itself lives in R2 and is served from a public URL. owner_type/owner_id
-- tie an object to a vendor, place, review, user, etc.; role distinguishes
-- cover vs gallery vs promo-video vs doc vs avatar vs review vs contrib.
CREATE TABLE IF NOT EXISTS media (
    id           TEXT PRIMARY KEY,
    owner_type   TEXT NOT NULL,
    owner_id     TEXT NOT NULL,
    kind         TEXT NOT NULL,               -- image | video | doc
    role         TEXT NOT NULL,               -- cover | gallery | promo | doc | avatar | review | contrib | menu_item
    object_key   TEXT NOT NULL,               -- R2 object key
    url          TEXT NOT NULL,               -- public URL
    content_type TEXT NOT NULL,
    size         BIGINT NOT NULL DEFAULT 0,
    width        INT NOT NULL DEFAULT 0,
    height       INT NOT NULL DEFAULT 0,
    uploaded_by  TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'ready',
    sort         INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS media_owner_idx ON media (owner_type, owner_id);

-- Traveller/vendor profile picture, served from R2 (empty = use initials).
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '';
