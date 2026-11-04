-- User privacy toggles: group visibility, public vibe votes inclusion, and personalised marketing offers.
ALTER TABLE users ADD COLUMN IF NOT EXISTS visible_to_group BOOL NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS count_vibe_votes BOOL NOT NULL DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS personalised_offers BOOL NOT NULL DEFAULT false;
