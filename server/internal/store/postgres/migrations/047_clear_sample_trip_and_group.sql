-- Older app builds started every new user on the same made-up sample group
-- ("Weligama Crew", code WLG-4K2, members mia/theo/ines) and sample trip
-- ("Weligama Weekend", code WLG-7QF). Whoever edited them saved those copies
-- here. The group code keys the live-location socket (/ws/group), so every
-- account holding WLG-4K2 shared one channel with strangers.

-- Groups: a fresh private code for the sample code and any code shared by
-- more than one group, the made-up members gone, the default name renamed.
UPDATE groups
SET code = 'GRP-' || upper(substr(md5(owner_id || clock_timestamp()::text || random()::text), 1, 12))
WHERE code = 'WLG-4K2'
   OR code IN (SELECT code FROM groups WHERE code <> '' GROUP BY code HAVING count(*) > 1);

UPDATE groups
SET member_ids = member_ids - 'mia' - 'theo' - 'ines' - 'kai' - 'sofia' - 'noah'
WHERE member_ids ?| ARRAY['mia', 'theo', 'ines', 'kai', 'sofia', 'noah'];

UPDATE groups SET name = 'Spotto Crew' WHERE name = 'Weligama Crew';

-- One group per code from now on (PUT /me/group swaps out a taken code).
CREATE UNIQUE INDEX groups_code_key ON groups (code) WHERE code <> '';

-- The sample trip: renamed, without its made-up collaborators and its
-- sample stops (s1-s7, which pointed at places that don't exist). Stops the
-- user added themselves stay. The app cleans its saved copy the same way.
UPDATE trips
SET name = CASE WHEN name = 'Weligama Weekend' THEN 'Spotto Trip' ELSE name END,
    collab = collab - 'theo' - 'mia',
    days = COALESCE((
        SELECT jsonb_agg(
            CASE WHEN jsonb_typeof(d -> 'stops') = 'array' THEN
                jsonb_set(d, '{stops}', COALESCE((
                    SELECT jsonb_agg(s ORDER BY n)
                    FROM jsonb_array_elements(d -> 'stops') WITH ORDINALITY AS st(s, n)
                    WHERE s ? 'custom'
                       OR NOT (s ->> 'id' = ANY (ARRAY['s1', 's2', 's3', 's4', 's5', 's6', 's7']))
                ), '[]'::jsonb))
            ELSE d END
            ORDER BY i)
        FROM jsonb_array_elements(days) WITH ORDINALITY AS dy(d, i)
    ), '[]'::jsonb)
WHERE code = 'WLG-7QF' AND jsonb_typeof(days) = 'array';
