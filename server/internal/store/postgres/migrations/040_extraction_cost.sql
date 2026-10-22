-- What each reel extraction cost, so moving extraction to its own server
-- is decided from real numbers (see internal/reelextract).
ALTER TABLE link_extractions
    ADD COLUMN video_bytes BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN video_secs REAL NOT NULL DEFAULT 0,
    ADD COLUMN processing_ms INT NOT NULL DEFAULT 0;
