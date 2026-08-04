-- Add apple_id column for Apple OAuth authentication
ALTER TABLE users ADD COLUMN IF NOT EXISTS apple_id TEXT UNIQUE;
