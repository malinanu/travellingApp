-- ISO 3166-1 alpha-2 country code, used to pick the right public-holiday
-- calendar for the automatic closure-prompt feature (see
-- internal/holidays). Every existing vendor really is Sri Lanka, so
-- backfilling 'LK' is correct, not just a placeholder default.
ALTER TABLE vendors ADD COLUMN country TEXT NOT NULL DEFAULT 'LK';
