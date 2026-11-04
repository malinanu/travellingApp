-- Allow multiple quest reward proposals per vendor with a unique sponsorship ID.
-- Existing rows receive an ID, and daily_quests gains an optional sponsorship_id reference.

ALTER TABLE vendor_sponsorships ADD COLUMN IF NOT EXISTS id TEXT;

-- Generate unique IDs for existing rows
UPDATE vendor_sponsorships
SET id = 'spon_' || substr(md5(random()::text || vendor_id || now()::text), 1, 16)
WHERE id IS NULL;

ALTER TABLE vendor_sponsorships ALTER COLUMN id SET NOT NULL;

-- Drop existing single-vendor primary key and set id as primary key if not already set
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint con
        JOIN pg_class rel ON rel.oid = con.conrelid
        JOIN pg_attribute a ON a.attrelid = rel.oid AND a.attnum = ANY(con.conkey)
        WHERE rel.relname = 'vendor_sponsorships'
          AND con.contype = 'p'
          AND a.attname != 'id'
    ) THEN
        ALTER TABLE vendor_sponsorships DROP CONSTRAINT vendor_sponsorships_pkey;
        ALTER TABLE vendor_sponsorships ADD PRIMARY KEY (id);
    ELSIF NOT EXISTS (
        SELECT 1
        FROM pg_constraint con
        JOIN pg_class rel ON rel.oid = con.conrelid
        WHERE rel.relname = 'vendor_sponsorships' AND con.contype = 'p'
    ) THEN
        ALTER TABLE vendor_sponsorships ADD PRIMARY KEY (id);
    END IF;
END $$;

-- Additional metadata columns
ALTER TABLE vendor_sponsorships ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
ALTER TABLE vendor_sponsorships ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS vendor_sponsorships_vendor_id_idx ON vendor_sponsorships (vendor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS vendor_sponsorships_status_idx ON vendor_sponsorships (status, created_at DESC);

-- Link daily_quests to the specific sponsorship proposal/reward
ALTER TABLE daily_quests ADD COLUMN IF NOT EXISTS sponsorship_id TEXT REFERENCES vendor_sponsorships(id) ON DELETE SET NULL;

-- Backfill existing daily_quests to link to their vendor's sponsorship
UPDATE daily_quests dq
SET sponsorship_id = vs.id
FROM vendor_sponsorships vs
WHERE vs.vendor_id = dq.vendor_id AND dq.sponsorship_id IS NULL;
