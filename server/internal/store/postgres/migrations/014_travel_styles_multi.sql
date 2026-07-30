-- travel_style was a single TEXT value ("nomad") — the app now lets a
-- traveller pick more than one style, matching how `interests` already
-- works. Convert in place: an existing single value becomes a one-item
-- array, empty string becomes an empty array, so no data is lost.
ALTER TABLE users ALTER COLUMN travel_style DROP DEFAULT;
ALTER TABLE users ALTER COLUMN travel_style TYPE JSONB USING (
    CASE WHEN travel_style = '' THEN '[]'::jsonb ELSE jsonb_build_array(travel_style) END
);
ALTER TABLE users ALTER COLUMN travel_style SET DEFAULT '[]'::jsonb;
