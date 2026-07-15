-- Vendor team accounts (owner + invited staff), a phone field for the
-- account settings page, and notification preference toggles.
ALTER TABLE users ADD COLUMN vendor_role TEXT NOT NULL DEFAULT 'owner';
ALTER TABLE users ADD COLUMN phone TEXT NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN notify_redemptions BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN notify_reviews BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN notify_payouts BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN notify_growth_tips BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE vendor_invites (
    id               TEXT PRIMARY KEY,
    vendor_id        TEXT NOT NULL,
    code             TEXT NOT NULL UNIQUE,
    created_by       TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at       TIMESTAMPTZ NOT NULL,
    accepted_at      TIMESTAMPTZ,
    accepted_user_id TEXT,
    revoked          BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX vendor_invites_vendor_idx ON vendor_invites (vendor_id);
