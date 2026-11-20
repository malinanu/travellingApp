-- Paid bookings, commission, refunds, ledger and payouts.
--
-- All money is integer minor units (cents) of currency, LKR for now.
-- Ledger sign convention: a positive amount is money Spotto owes the vendor,
-- a negative amount is money the vendor owes Spotto.

-- What a service costs, and its cancellation terms.
ALTER TABLE bookable_services
    ADD COLUMN IF NOT EXISTS price_minor BIGINT NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'LKR',
    ADD COLUMN IF NOT EXISTS pricing_mode TEXT NOT NULL DEFAULT 'free'
        CHECK (pricing_mode IN ('free', 'per_person', 'per_booking', 'per_night')),
    ADD COLUMN IF NOT EXISTS free_cancel_hours INT NOT NULL DEFAULT 24 CHECK (free_cancel_hours >= 0),
    ADD COLUMN IF NOT EXISTS late_cancel_refund_pct INT NOT NULL DEFAULT 0
        CHECK (late_cancel_refund_pct BETWEEN 0 AND 100);

-- The money and cancellation terms of a booking, fixed when it is made so
-- later price, policy or commission changes never rewrite past bookings.
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS currency TEXT NOT NULL DEFAULT 'LKR',
    ADD COLUMN IF NOT EXISTS commission_mode TEXT NOT NULL DEFAULT 'none'
        CHECK (commission_mode IN ('none', 'percent', 'per_guest_fee')),
    ADD COLUMN IF NOT EXISTS commission_rate_bps INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS fee_per_guest_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS commission_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS vendor_net_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refunded_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS payment_status TEXT NOT NULL DEFAULT 'not_required'
        CHECK (payment_status IN ('not_required', 'pending', 'paid', 'refunded', 'partially_refunded', 'failed')),
    ADD COLUMN IF NOT EXISTS free_cancel_hours INT NOT NULL DEFAULT 24,
    ADD COLUMN IF NOT EXISTS late_cancel_refund_pct INT NOT NULL DEFAULT 0;

-- How Spotto's cut is worked out. The most specific active rule wins:
-- vendor, then category, then booking type, then default. A rule takes
-- effect at active_from, so changing a rate adds a row and keeps history.
-- mode=percent applies to paid bookings (rate_bps of the amount);
-- mode=per_guest_fee applies to free bookings and is charged to the
-- vendor per guest when the booking is checked in.
CREATE TABLE IF NOT EXISTS commission_rules (
    id                  TEXT PRIMARY KEY,
    scope               TEXT NOT NULL CHECK (scope IN ('default', 'booking_type', 'category', 'vendor')),
    scope_key           TEXT NOT NULL DEFAULT '',
    mode                TEXT NOT NULL CHECK (mode IN ('percent', 'per_guest_fee')),
    rate_bps            INT NOT NULL DEFAULT 0 CHECK (rate_bps BETWEEN 0 AND 10000),
    fee_per_guest_minor BIGINT NOT NULL DEFAULT 0 CHECK (fee_per_guest_minor >= 0),
    active_from         TIMESTAMPTZ NOT NULL DEFAULT now(),
    note                TEXT NOT NULL DEFAULT '',
    created_by          TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scope, scope_key, mode, active_from)
);
CREATE INDEX IF NOT EXISTS commission_rules_lookup_idx ON commission_rules (mode, scope, scope_key, active_from DESC);

-- Placeholder rates until the business values are set in the admin:
-- 10% on paid bookings, LKR 100 per seated guest on free table bookings.
INSERT INTO commission_rules (id, scope, scope_key, mode, rate_bps, active_from, note)
VALUES ('cr_default_percent', 'default', '', 'percent', 1000, '2000-01-01', 'Placeholder default commission')
ON CONFLICT DO NOTHING;
INSERT INTO commission_rules (id, scope, scope_key, mode, fee_per_guest_minor, active_from, note)
VALUES ('cr_table_per_guest', 'booking_type', 'table', 'per_guest_fee', 10000, '2000-01-01', 'Placeholder per-guest fee for free tables')
ON CONFLICT DO NOTHING;

-- One gateway checkout for one booking.
CREATE TABLE IF NOT EXISTS payments (
    id                 TEXT PRIMARY KEY,
    booking_id         TEXT NOT NULL REFERENCES bookings(id) ON DELETE RESTRICT,
    gateway            TEXT NOT NULL,
    gateway_order_id   TEXT NOT NULL UNIQUE,
    gateway_payment_id TEXT NOT NULL DEFAULT '',
    amount_minor       BIGINT NOT NULL,
    currency           TEXT NOT NULL,
    status             TEXT NOT NULL DEFAULT 'created'
        CHECK (status IN ('created', 'succeeded', 'failed', 'cancelled', 'chargeback')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS payments_booking_idx ON payments (booking_id);

-- Every gateway notification, recorded once: event_key is unique, so a
-- repeated notification is recognised and ignored.
CREATE TABLE IF NOT EXISTS payment_events (
    id          TEXT PRIMARY KEY,
    payment_id  TEXT REFERENCES payments(id) ON DELETE RESTRICT,
    gateway     TEXT NOT NULL,
    event_key   TEXT NOT NULL UNIQUE,
    status      TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS refunds (
    id                TEXT PRIMARY KEY,
    booking_id        TEXT NOT NULL REFERENCES bookings(id) ON DELETE RESTRICT,
    payment_id        TEXT REFERENCES payments(id) ON DELETE RESTRICT,
    amount_minor      BIGINT NOT NULL CHECK (amount_minor > 0),
    currency          TEXT NOT NULL,
    reason            TEXT NOT NULL DEFAULT '',
    -- manual: the gateway can't refund automatically; finance pays it back
    -- by hand and marks it done.
    status            TEXT NOT NULL DEFAULT 'requested'
        CHECK (status IN ('requested', 'succeeded', 'failed', 'manual', 'manual_done')),
    gateway_refund_id TEXT NOT NULL DEFAULT '',
    requested_by      TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS refunds_status_idx ON refunds (status, created_at);

CREATE TABLE IF NOT EXISTS payouts (
    id             TEXT PRIMARY KEY,
    vendor_id      TEXT NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    amount_minor   BIGINT NOT NULL CHECK (amount_minor > 0),
    currency       TEXT NOT NULL,
    period_end     TIMESTAMPTZ NOT NULL,
    status         TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'approved', 'paid', 'failed', 'cancelled')),
    approved_by    TEXT NOT NULL DEFAULT '',
    approved_at    TIMESTAMPTZ,
    paid_at        TIMESTAMPTZ,
    bank_reference TEXT NOT NULL DEFAULT '',
    note           TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS payouts_vendor_idx ON payouts (vendor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS payouts_status_idx ON payouts (status, created_at);

-- What Spotto and each vendor owe each other. An entry with no payout_id is
-- unsettled; a payout settles the entries attached to it.
CREATE TABLE IF NOT EXISTS ledger_entries (
    id           TEXT PRIMARY KEY,
    vendor_id    TEXT NOT NULL REFERENCES vendors(id) ON DELETE RESTRICT,
    booking_id   TEXT REFERENCES bookings(id) ON DELETE RESTRICT,
    payout_id    TEXT REFERENCES payouts(id) ON DELETE RESTRICT,
    kind         TEXT NOT NULL CHECK (kind IN ('booking_net', 'per_guest_fee', 'refund_reversal', 'adjustment')),
    amount_minor BIGINT NOT NULL,
    currency     TEXT NOT NULL,
    -- When the entry becomes payable: the booking's end for booking money,
    -- so nothing is paid out before the booking has taken place.
    payable_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    memo         TEXT NOT NULL DEFAULT '',
    created_by   TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- One entry of each booking-driven kind per booking: a repeated webhook or
-- a second QR scan can't double-count.
CREATE UNIQUE INDEX IF NOT EXISTS ledger_entries_booking_kind_idx
    ON ledger_entries (booking_id, kind) WHERE booking_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ledger_entries_unsettled_idx
    ON ledger_entries (vendor_id, payable_at) WHERE payout_id IS NULL;

-- Where a vendor's payouts go. The account number is encrypted with
-- PAYOUT_ENCRYPTION_KEY (AES-256-GCM) and only its last four digits are
-- shown outside the finance team.
CREATE TABLE IF NOT EXISTS vendor_payout_accounts (
    vendor_id          TEXT PRIMARY KEY REFERENCES vendors(id) ON DELETE CASCADE,
    account_holder     TEXT NOT NULL,
    bank_name          TEXT NOT NULL,
    branch             TEXT NOT NULL DEFAULT '',
    account_number_enc TEXT NOT NULL,
    account_last4      TEXT NOT NULL,
    verified_at        TIMESTAMPTZ,
    verified_by        TEXT NOT NULL DEFAULT '',
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
