ALTER TABLE vendor_sponsorships ADD COLUMN status TEXT NOT NULL DEFAULT 'pending';

CREATE TABLE daily_quests (
    date TEXT PRIMARY KEY, -- 'YYYY-MM-DD'
    vendor_id TEXT NOT NULL REFERENCES vendors(id),
    goal INT NOT NULL DEFAULT 5,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
