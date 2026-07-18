-- Email + role on vendor invites (so a "manager" role can be assigned and
-- invites can be delivered by email instead of only a copy-link code),
-- Google OAuth account linking, and password-reset tokens.
ALTER TABLE vendor_invites ADD COLUMN email TEXT NOT NULL DEFAULT '';
ALTER TABLE vendor_invites ADD COLUMN role TEXT NOT NULL DEFAULT 'staff';

ALTER TABLE users ADD COLUMN google_id TEXT;
CREATE UNIQUE INDEX users_google_id_idx ON users (google_id) WHERE google_id IS NOT NULL;

-- token_hash, never the raw token, following the same never-store-the-
-- secret-itself pattern TOTP_ENCRYPTION_KEY uses for 2FA secrets.
CREATE TABLE password_reset_tokens (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX password_reset_tokens_user_idx ON password_reset_tokens (user_id);
CREATE INDEX password_reset_tokens_hash_idx ON password_reset_tokens (token_hash);
