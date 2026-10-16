-- A saved record of each traveller finishing a day's quest. Until now
-- "complete" was only worked out on the fly from today's check-ins, so
-- nobody could look back and see who finished which quest; quest_claims
-- only covers rewards that were actually scanned.
CREATE TABLE quest_completions (
    traveller_id TEXT NOT NULL,
    date TEXT NOT NULL, -- UTC quest day, 'YYYY-MM-DD' (same as daily_quests.date)
    vendor_id TEXT REFERENCES vendors(id) ON DELETE SET NULL, -- the day's sponsor; NULL on the fallback Spotto quest
    goal INT NOT NULL,
    reward_description TEXT NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (traveller_id, date)
);
CREATE INDEX quest_completions_vendor_date_idx ON quest_completions (vendor_id, date);
CREATE INDEX quest_completions_date_idx ON quest_completions (date, completed_at DESC);

-- Daily progress counts a traveller's check-ins since the start of the UTC day.
CREATE INDEX IF NOT EXISTS checkins_traveller_created_idx ON checkins (traveller_id, created_at);

-- Backfill from past check-ins, so the history isn't empty on day one. A day
-- counts as completed at the first check-in of the goal-th distinct place,
-- using the same goal the server would have used that day (the scheduled
-- quest's, or 5 when nothing was scheduled).
WITH firsts AS (
    SELECT traveller_id,
           to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day,
           place_id,
           MIN(created_at) AS at
    FROM checkins
    GROUP BY 1, 2, 3
), ranked AS (
    SELECT *, row_number() OVER (PARTITION BY traveller_id, day ORDER BY at) AS n
    FROM firsts
), scheduled AS (
    SELECT d.date, d.vendor_id, d.goal, vs.reward_description
    FROM daily_quests d
    JOIN vendor_sponsorships vs ON vs.vendor_id = d.vendor_id
)
INSERT INTO quest_completions (traveller_id, date, vendor_id, goal, reward_description, completed_at)
SELECT r.traveller_id, r.day, s.vendor_id, COALESCE(s.goal, 5), COALESCE(s.reward_description, 'Mystery Reward'), r.at
FROM ranked r
LEFT JOIN scheduled s ON s.date = r.day
WHERE r.n = COALESCE(s.goal, 5)
ON CONFLICT DO NOTHING;
