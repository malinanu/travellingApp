-- Opt-in live location sharing for the "nearby travellers" feature, plus a
-- generic consent flag surfaced on the app's Privacy page.
ALTER TABLE users ADD COLUMN share_location BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN lat DOUBLE PRECISION;
ALTER TABLE users ADD COLUMN lng DOUBLE PRECISION;
ALTER TABLE users ADD COLUMN location_updated_at TIMESTAMPTZ;
