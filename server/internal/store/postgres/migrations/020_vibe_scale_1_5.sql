-- Rescales Current Vibe check-ins from the old 0-10-in-steps-of-2 scale
-- (the Flutter tap UI only ever sent 0,2,4,6,8,10 — 5 segments worth 2
-- points each) to a proper 1-5 scale (0 stays valid as the "not yet
-- rated" sentinel for a dimension not part of a given tap). Old values
-- were always even, so integer division by 2 is an exact, lossless
-- conversion (0->0, 2->1, 4->2, 6->3, 8->4, 10->5) — safe to run before
-- tightening the constraint below.
UPDATE vibe_checkins SET productive = productive / 2, loud = loud / 2, crowded = crowded / 2;

ALTER TABLE vibe_checkins DROP CONSTRAINT IF EXISTS vibe_checkins_productive_check;
ALTER TABLE vibe_checkins ADD CONSTRAINT vibe_checkins_productive_check CHECK (productive BETWEEN 0 AND 5);
ALTER TABLE vibe_checkins DROP CONSTRAINT IF EXISTS vibe_checkins_loud_check;
ALTER TABLE vibe_checkins ADD CONSTRAINT vibe_checkins_loud_check CHECK (loud BETWEEN 0 AND 5);
ALTER TABLE vibe_checkins DROP CONSTRAINT IF EXISTS vibe_checkins_crowded_check;
ALTER TABLE vibe_checkins ADD CONSTRAINT vibe_checkins_crowded_check CHECK (crowded BETWEEN 0 AND 5);
