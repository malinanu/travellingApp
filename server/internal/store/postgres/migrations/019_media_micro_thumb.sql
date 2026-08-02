-- Tiny inline base64 blur-placeholder derivative (see internal/media's
-- GenerateThumbnails) alongside the existing 320px thumb, plus a status
-- column driving the background thumbnail worker
-- (internal/http/thumbnail_worker.go) that replaces the old synchronous
-- generation in handleMediaConfirm.
ALTER TABLE media ADD COLUMN IF NOT EXISTS micro_thumb TEXT NOT NULL DEFAULT '';
ALTER TABLE media ADD COLUMN IF NOT EXISTS thumb_status TEXT NOT NULL DEFAULT '';

-- Backfill: every existing ready image needs the worker to (re)generate
-- both derivatives — old rows already have thumb_key but never got a
-- micro_thumb, and anything that never got thumbnailed gets picked up too.
UPDATE media SET thumb_status = 'pending' WHERE kind = 'image' AND status = 'ready';
