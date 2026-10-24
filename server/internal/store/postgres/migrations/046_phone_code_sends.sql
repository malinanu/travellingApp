-- One row per phone verification code sent by WhatsApp or SMS. Every send
-- costs money, so sends are capped per account, per number and across the
-- whole platform per day (see ReservePhoneCodeSend). Rows older than two
-- days are pruned as new ones arrive. No foreign key: deleting an account
-- must not reset the counts.
CREATE TABLE phone_code_sends (
    id        BIGSERIAL PRIMARY KEY,
    user_id   TEXT NOT NULL,
    phone_key TEXT NOT NULL,
    channel   TEXT NOT NULL,
    sent_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX phone_code_sends_sent_at_idx ON phone_code_sends (sent_at);
CREATE INDEX phone_code_sends_user_idx ON phone_code_sends (user_id, sent_at);
CREATE INDEX phone_code_sends_phone_idx ON phone_code_sends (phone_key, sent_at);
