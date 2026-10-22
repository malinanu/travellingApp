-- importsrilanka gave every place's owner account (import_<placeId>@spotto.dev)
-- the same password, spotto123. The place ids are in the public repo, so every
-- one of those vendor logins could be guessed. '!locked' is not a bcrypt hash,
-- so no password can ever match it. The token_version bump and session revoke
-- sign out anyone who already got in. Real owners get access through vendor
-- profile grants, never through these accounts.
UPDATE users SET password_hash = '!locked', token_version = token_version + 1
WHERE email LIKE 'import\_%@spotto.dev';

UPDATE user_sessions SET revoked_at = now()
WHERE revoked_at IS NULL
  AND user_id IN (SELECT id FROM users WHERE email LIKE 'import\_%@spotto.dev');
