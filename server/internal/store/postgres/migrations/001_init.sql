CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    country       TEXT NOT NULL DEFAULT '',
    travel_style  TEXT NOT NULL DEFAULT '',
    interests     JSONB NOT NULL DEFAULT '[]',
    vendor_id     TEXT,
    extra         JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vendors (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    category     TEXT NOT NULL DEFAULT '',
    town         TEXT NOT NULL DEFAULT '',
    region_id    TEXT,
    owner        TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    priority     TEXT NOT NULL DEFAULT 'normal',
    docs         BOOLEAN NOT NULL DEFAULT FALSE,
    photos       INTEGER NOT NULL DEFAULT 0,
    wifi_score   DOUBLE PRECISION,
    flags        JSONB NOT NULL DEFAULT '[]',
    note         TEXT NOT NULL DEFAULT '',
    tier         TEXT NOT NULL DEFAULT 'Free',
    profile      JSONB NOT NULL DEFAULT '{}',
    menu         JSONB NOT NULL DEFAULT '{}',
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE offers (
    id        TEXT PRIMARY KEY,
    vendor_id TEXT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    title     TEXT NOT NULL,
    type      TEXT NOT NULL DEFAULT '',
    value     TEXT NOT NULL DEFAULT '',
    status    TEXT NOT NULL DEFAULT 'draft',
    start_at  TEXT NOT NULL DEFAULT '',
    end_at    TEXT NOT NULL DEFAULT '',
    lim       INTEGER NOT NULL DEFAULT 0,
    claimed   INTEGER NOT NULL DEFAULT 0,
    redeemed  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE places (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    vendor_id TEXT,
    data      JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE claims (
    id           TEXT PRIMARY KEY,
    code         TEXT NOT NULL UNIQUE,
    place_id     TEXT NOT NULL DEFAULT '',
    offer_label  TEXT NOT NULL DEFAULT '',
    offer_value  TEXT NOT NULL DEFAULT '',
    vendor_id    TEXT,
    traveller_id TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'claimed',
    claimed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    redeemed_at  TIMESTAMPTZ
);
CREATE INDEX claims_vendor_idx ON claims (vendor_id, redeemed_at DESC);
CREATE INDEX claims_traveller_idx ON claims (traveller_id);

CREATE TABLE trips (
    owner_id   TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '',
    code       TEXT NOT NULL DEFAULT '',
    start_date TEXT NOT NULL DEFAULT '',
    collab     JSONB NOT NULL DEFAULT '[]',
    days       JSONB NOT NULL DEFAULT '[]',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE groups (
    owner_id   TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL DEFAULT '',
    code       TEXT NOT NULL DEFAULT '',
    member_ids JSONB NOT NULL DEFAULT '[]'
);

CREATE TABLE feed_items (
    id         BIGSERIAL PRIMARY KEY,
    vendor     TEXT NOT NULL DEFAULT '',
    town       TEXT NOT NULL DEFAULT '',
    type       TEXT NOT NULL DEFAULT 'Offer',
    text       TEXT NOT NULL DEFAULT '',
    status     TEXT NOT NULL DEFAULT 'ok',
    reason     TEXT NOT NULL DEFAULT '',
    resolved   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX feed_items_created_idx ON feed_items (created_at DESC);

CREATE TABLE regions (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    x         INTEGER NOT NULL DEFAULT 0,
    y         INTEGER NOT NULL DEFAULT 0,
    travelers INTEGER NOT NULL DEFAULT 0,
    vendors   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE subscriptions (
    id        TEXT PRIMARY KEY,
    vendor_id TEXT,
    vendor    TEXT NOT NULL DEFAULT '',
    town      TEXT NOT NULL DEFAULT '',
    tier      TEXT NOT NULL DEFAULT 'Free',
    mrr       INTEGER NOT NULL DEFAULT 0,
    renews_at TIMESTAMPTZ,
    state     TEXT NOT NULL DEFAULT 'free',
    since     TEXT NOT NULL DEFAULT ''
);

CREATE TABLE audio_guides (
    id        TEXT PRIMARY KEY,
    site      TEXT NOT NULL DEFAULT '',
    region    TEXT NOT NULL DEFAULT '',
    lat       DOUBLE PRECISION NOT NULL DEFAULT 0,
    lng       DOUBLE PRECISION NOT NULL DEFAULT 0,
    lang      TEXT NOT NULL DEFAULT 'English',
    published BOOLEAN NOT NULL DEFAULT FALSE,
    plays     INTEGER NOT NULL DEFAULT 0,
    depths    JSONB NOT NULL DEFAULT '[]',
    duration  JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE vibe_votes (
    place_id     TEXT NOT NULL,
    vibe_id      TEXT NOT NULL,
    traveller_id TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (place_id, vibe_id, traveller_id)
);

CREATE TABLE checkins (
    id           BIGSERIAL PRIMARY KEY,
    place_id     TEXT NOT NULL,
    traveller_id TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE vendor_notifications (
    id        TEXT PRIMARY KEY,
    vendor_id TEXT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    icon      TEXT NOT NULL DEFAULT '',
    tone      TEXT NOT NULL DEFAULT '',
    title     TEXT NOT NULL DEFAULT '',
    sub       TEXT NOT NULL DEFAULT '',
    t         TEXT NOT NULL DEFAULT '',
    unread    BOOLEAN NOT NULL DEFAULT TRUE
);

-- Rich vendor analytics blobs (dashboard, growth, reports) keyed by vendor.
CREATE TABLE vendor_blobs (
    vendor_id TEXT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    kind      TEXT NOT NULL,
    data      JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (vendor_id, kind)
);
