-- Persists Tomorrow.io responses across process restarts. importsrilanka
-- runs (and the whole API restarts) on every deploy, which would otherwise
-- wipe internal/weather/cache.go's in-memory cache and re-spend calls
-- against the free tier's tight limit (25/hour, 500/day) right after every
-- deploy. Keyed by the same coarse ~55km cell as the in-memory cache.
CREATE TABLE weather_cache (
    cell_key TEXT PRIMARY KEY,
    snapshot JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
