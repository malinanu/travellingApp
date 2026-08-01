-- Server-generated thumbnail for image uploads (handleMediaConfirm resizes
-- a small derivative to R2 alongside the original) — empty for anything
-- uploaded before this change, or where thumbnail generation failed, so
-- clients always fall back to the full-size url in that case.
ALTER TABLE media ADD COLUMN IF NOT EXISTS thumb_key TEXT NOT NULL DEFAULT '';
