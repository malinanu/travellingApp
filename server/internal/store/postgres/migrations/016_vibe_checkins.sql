-- Replaces the old one-time-ever binary vibe_votes with a richer, repeatable
-- per-traveller check-in: each submission is a full snapshot (all three
-- dimensions together) with its own timestamp, so a traveller can re-rate a
-- place on a later visit instead of being stuck with their first vote
-- forever. vibe_votes/its old handlers are being dropped in the same
-- change, not kept alongside this.
CREATE TABLE vibe_checkins (
    id           BIGSERIAL PRIMARY KEY,
    place_id     TEXT NOT NULL,
    traveller_id TEXT NOT NULL,
    productive   SMALLINT NOT NULL CHECK (productive BETWEEN 0 AND 10),
    loud         SMALLINT NOT NULL CHECK (loud BETWEEN 0 AND 10),
    crowded      SMALLINT NOT NULL CHECK (crowded BETWEEN 0 AND 10),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Serves both VibeAggregate (latest row per traveller for a place) and
-- MyLatestVibeCheckin (latest row for one traveller at a place).
CREATE INDEX vibe_checkins_place_traveller_idx ON vibe_checkins (place_id, traveller_id, created_at DESC);

DROP TABLE vibe_votes;
