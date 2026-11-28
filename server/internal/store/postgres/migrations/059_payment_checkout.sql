-- The gateway checkout a payment sends the traveller to (an HTML form:
-- URL plus fields), kept so the hosted checkout page can be reopened.
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS checkout_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS checkout_fields JSONB NOT NULL DEFAULT '{}';

-- Notifications that verified but couldn't be applied (amount mismatch,
-- unknown order) are kept for the finance team to look at.
ALTER TABLE payment_events
    ADD COLUMN IF NOT EXISTS outcome TEXT NOT NULL DEFAULT '';
