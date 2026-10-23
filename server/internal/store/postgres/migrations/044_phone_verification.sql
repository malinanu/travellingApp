-- Vendor-portal team members (owner, managers, staff) verify their own
-- phone number with a one-time code sent by WhatsApp or SMS. Numbers stay
-- non-unique; a verified number on a business's team is what lets its
-- Business Profile contact number skip the "used by another business"
-- warning (see handleVendorPhoneCheck).
ALTER TABLE users ADD COLUMN phone_verified_at TIMESTAMPTZ;

-- At most one live code per user; a new request replaces the old one.
CREATE TABLE phone_verifications (
    user_id    TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    phone      TEXT NOT NULL,
    channel    TEXT NOT NULL,
    code_hash  TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    attempts   INT NOT NULL DEFAULT 0,
    sent_at    TIMESTAMPTZ NOT NULL
);
