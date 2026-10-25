-- Where app users are, for the admin geo page's "Travellers" heat map. When
-- the app loads nearby places it sends the device position; the server keeps
-- only a rounded cell (degrees x 100, about 1.1 km) once per traveller per
-- cell per day, never the exact point. Travellers turn it off with the app's
-- "Usage analytics" switch (users.usage_analytics), which also deletes their
-- rows. Rows older than 180 days are pruned (see StartPresencePruner).
ALTER TABLE users ADD COLUMN usage_analytics BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE traveller_presence (
    user_id  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day      DATE NOT NULL,
    cell_lat INT NOT NULL,
    cell_lng INT NOT NULL,
    PRIMARY KEY (user_id, day, cell_lat, cell_lng)
);
CREATE INDEX traveller_presence_day_idx ON traveller_presence (day);
