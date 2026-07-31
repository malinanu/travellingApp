-- Persists live Google Places Details lookups (phone/website/opening
-- hours/wheelchair accessibility/price level) across process restarts, same
-- rationale as weather_cache (013): importsrilanka runs (and the whole API
-- restarts) on every deploy, which would otherwise wipe
-- internal/placedetails.Cache's in-memory cache and re-spend paid Places
-- API calls right after every deploy. Keyed by the place's Google Place ID
-- (data.sourceId), not our internal place id, since that's the identifier
-- the Places Details endpoint takes.
CREATE TABLE place_details_cache (
    place_id TEXT PRIMARY KEY,
    details JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);
