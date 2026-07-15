-- token_version lets the server revoke all previously issued JWTs for a
-- user (e.g. when 2FA is disabled) without needing a full session/blacklist
-- store: bump the counter and every outstanding token's embedded version
-- stops matching on next request.
ALTER TABLE users ADD COLUMN token_version INTEGER NOT NULL DEFAULT 0;
