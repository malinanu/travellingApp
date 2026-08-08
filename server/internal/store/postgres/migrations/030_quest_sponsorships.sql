CREATE TABLE vendor_sponsorships (
    vendor_id TEXT PRIMARY KEY,
    reward_description TEXT NOT NULL,
    reward_limit INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE quest_claims (
    id TEXT PRIMARY KEY,
    traveller_id TEXT NOT NULL,
    vendor_id TEXT NOT NULL,
    date TEXT NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(traveller_id, date)
);
