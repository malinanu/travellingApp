-- Server-side login sessions. Each signed-in device holds a refresh token
-- (only its SHA-256 is stored) that it trades for new short-lived access
-- tokens, so it stays signed in until logout, revocation, or going unused
-- for longer than its idle limit.
CREATE TABLE user_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_hash TEXT NOT NULL UNIQUE,
    -- The hash this session's refresh token replaced, and when. A request
    -- with it soon after is a harmless race (two tabs refreshing at once);
    -- later, it means an old token was copied, and the session is revoked.
    previous_hash TEXT,
    rotated_at TIMESTAMPTZ,
    client TEXT NOT NULL, -- app | admin | vendor
    device_label TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX user_sessions_user_idx ON user_sessions (user_id);
CREATE INDEX user_sessions_previous_hash_idx ON user_sessions (previous_hash);
