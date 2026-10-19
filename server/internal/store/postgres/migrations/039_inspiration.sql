-- Inspiration Board: travellers save reels/posts, the server watches the
-- video and lists the places in it, and travellers save those places.

-- Worldwide places found through Google Places Text Search. Kept so any
-- later lookup of the same place (by anyone) is served from here instead of
-- paying Google again.
CREATE TABLE geo_places (
    id TEXT PRIMARY KEY,
    google_place_id TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    lat DOUBLE PRECISION NOT NULL,
    lng DOUBLE PRECISION NOT NULL,
    address TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    country_code TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT 'see', -- eat | drink | stay | see | do | shop
    rating DOUBLE PRECISION,
    rating_count INT NOT NULL DEFAULT 0,
    photo_name TEXT NOT NULL DEFAULT '',
    maps_url TEXT NOT NULL DEFAULT '',
    types JSONB NOT NULL DEFAULT '[]',
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Text-search query → result, including misses (geo_place_id NULL), so the
-- same "name, area, city, country" string is never sent to Google twice.
CREATE TABLE geo_place_queries (
    query_key TEXT PRIMARY KEY,
    geo_place_id TEXT REFERENCES geo_places(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One extraction per canonical link, shared by every traveller who saves it.
CREATE TABLE link_extractions (
    canonical_url TEXT PRIMARY KEY,
    platform TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending | processing | done | partial | failed
    error TEXT NOT NULL DEFAULT '',
    caption TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    thumbnail_url TEXT NOT NULL DEFAULT '',
    results JSONB NOT NULL DEFAULT '[]',
    model TEXT NOT NULL DEFAULT '',
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX link_extractions_pending_idx ON link_extractions (next_attempt_at) WHERE status = 'pending';

CREATE TABLE inspiration_links (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    canonical_url TEXT NOT NULL REFERENCES link_extractions(canonical_url),
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, canonical_url)
);
CREATE INDEX inspiration_links_canonical_idx ON inspiration_links (canonical_url);

-- "Save for later". place_id points at the Spotto catalog but has no FK:
-- cmd/importsrilanka wipes and reloads places on every boot.
CREATE TABLE saved_places (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    place_id TEXT,
    geo_place_id TEXT REFERENCES geo_places(id) ON DELETE CASCADE,
    source_link_id TEXT REFERENCES inspiration_links(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((place_id IS NULL) <> (geo_place_id IS NULL))
);
CREATE UNIQUE INDEX saved_places_user_place_idx ON saved_places (user_id, place_id) WHERE place_id IS NOT NULL;
CREATE UNIQUE INDEX saved_places_user_geo_idx ON saved_places (user_id, geo_place_id) WHERE geo_place_id IS NOT NULL;
