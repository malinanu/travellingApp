-- Several payment gateways side by side (PayHere in LKR, PayPal in USD).
-- payments.amount_minor/currency stay "what the gateway charges"; the
-- booking's own LKR amount and the exchange rate used are kept next to it.
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS checkout_method      TEXT   NOT NULL DEFAULT 'POST',
    ADD COLUMN IF NOT EXISTS gateway_checkout_ref TEXT   NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS booking_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS booking_currency     TEXT   NOT NULL DEFAULT '',
    -- LKR per unit of the charge currency, in millionths; 0 = same currency.
    ADD COLUMN IF NOT EXISTS fx_rate_micros       BIGINT NOT NULL DEFAULT 0;

UPDATE payments SET booking_amount_minor = amount_minor, booking_currency = currency
WHERE booking_currency = '';

CREATE INDEX IF NOT EXISTS payments_checkout_ref_idx
    ON payments (gateway, gateway_checkout_ref) WHERE gateway_checkout_ref <> '';

-- What a refund pays back through the gateway, in the gateway's currency.
ALTER TABLE refunds
    ADD COLUMN IF NOT EXISTS gateway_amount_minor BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gateway_currency     TEXT   NOT NULL DEFAULT '';

UPDATE refunds SET gateway_amount_minor = amount_minor, gateway_currency = currency
WHERE gateway_currency = '';

-- Exchange rates the finance team sets for gateways that can't charge LKR.
CREATE TABLE IF NOT EXISTS fx_rates (
    currency            TEXT PRIMARY KEY CHECK (currency ~ '^[A-Z]{3}$'),
    lkr_per_unit_micros BIGINT NOT NULL CHECK (lkr_per_unit_micros > 0),
    set_by              TEXT NOT NULL DEFAULT '',
    set_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
