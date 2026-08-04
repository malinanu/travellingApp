-- Stores the Google Place ID on a vendor row so that (1) enrichWithPlaceDetails
-- can look up live phone/website/accessibility for vendor-registered places
-- (not just importsrilanka-seeded ones), and (2) duplicate registrations for the
-- same Google listing can be detected. Nullable because many vendors register
-- manually without using Google's "Find your business" search.
ALTER TABLE vendors ADD COLUMN google_place_id TEXT;

-- Prevent two vendors from claiming the same Google listing. Null values
-- (manual registrations) are allowed to coexist — only actual place IDs
-- must be unique.
CREATE UNIQUE INDEX vendors_google_place_id_idx ON vendors (google_place_id) WHERE google_place_id IS NOT NULL;
