-- Traveller reviews (rating 1-5; 1-2 count as "bad" for vendor flagging),
-- real coordinates on vendors/regions (mirrors places.data's lat/lng),
-- and TOTP two-factor auth columns on users.

CREATE TABLE reviews (
    id           TEXT PRIMARY KEY,
    place_id     TEXT NOT NULL,
    vendor_id    TEXT,
    traveller_id TEXT NOT NULL,
    rating       INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    text         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (place_id, traveller_id)
);
CREATE INDEX reviews_vendor_idx ON reviews (vendor_id);
CREATE INDEX reviews_place_idx ON reviews (place_id, created_at DESC);

ALTER TABLE vendors ADD COLUMN lat DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE vendors ADD COLUMN lng DOUBLE PRECISION NOT NULL DEFAULT 0;
-- 'suspended' joins pending/approved/rejected/edits as a status value (no enum type in use).

ALTER TABLE regions ADD COLUMN lat DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE regions ADD COLUMN lng DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE users ADD COLUMN totp_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN totp_pending_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
