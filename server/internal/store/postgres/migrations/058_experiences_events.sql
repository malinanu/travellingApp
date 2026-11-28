-- Local hosts, experiences and dated events.

-- A vendor is either a venue (a place people walk into) or a host (a local
-- running experiences with a meeting point, no storefront).
ALTER TABLE vendors
    ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'venue' CHECK (kind IN ('venue', 'host'));

-- A category can allow more than one kind of booking (a bar takes table
-- reservations and sells tickets to its gig nights). booking_type stays as
-- the category's main type.
ALTER TABLE vendor_categories
    ADD COLUMN IF NOT EXISTS allowed_booking_types TEXT[] NOT NULL DEFAULT '{}';
UPDATE vendor_categories SET allowed_booking_types = ARRAY[booking_type]
WHERE booking_type IS NOT NULL AND booking_type <> '' AND allowed_booking_types = '{}';
-- Venues where people book a table can also run ticketed events.
UPDATE vendor_categories SET allowed_booking_types = ARRAY['table', 'event']
WHERE id IN ('restaurants', 'bars', 'nightlife', 'live-music') AND booking_type = 'table';

-- How a service is scheduled: weekly time windows (a daily 7am surf
-- lesson) or dated occurrences (a whale-watching trip on given dates, an
-- event night). Blackout dates ("YYYY-MM-DD", Sri Lankan days) close a
-- weekly service on specific days. meeting_point tells guests where an
-- experience starts: {"note": "...", "lat": .., "lng": ..}.
ALTER TABLE bookable_services
    ADD COLUMN IF NOT EXISTS scheduling TEXT NOT NULL DEFAULT 'weekly' CHECK (scheduling IN ('weekly', 'occurrences')),
    ADD COLUMN IF NOT EXISTS blackout_dates TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS meeting_point JSONB;

CREATE TABLE IF NOT EXISTS service_occurrences (
    id         TEXT PRIMARY KEY,
    service_id TEXT NOT NULL REFERENCES bookable_services(id) ON DELETE RESTRICT,
    start_at   TIMESTAMPTZ NOT NULL,
    end_at     TIMESTAMPTZ NOT NULL CHECK (end_at > start_at),
    -- NULL uses the service's capacity.
    capacity   INT CHECK (capacity IS NULL OR capacity > 0),
    status     TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS service_occurrences_service_idx ON service_occurrences (service_id, start_at);

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS occurrence_id TEXT REFERENCES service_occurrences(id) ON DELETE RESTRICT;
CREATE INDEX IF NOT EXISTS bookings_occurrence_idx ON bookings (occurrence_id) WHERE occurrence_id IS NOT NULL;

-- Older category slugs (from migration 032, still on some vendors) that are
-- experiences or events become bookable as such.
UPDATE vendor_categories
SET booking_type = 'experience', allowed_booking_types = ARRAY['experience', 'event']
WHERE id IN ('surf-school', 'dive-school', 'yoga-studio', 'cooking-class', 'pottery-workshop', 'batik-atelier',
             'spice-trail', 'safari-wildlife', 'boat-trip', 'city-tour', 'tour-guide', 'experience-host')
  AND (booking_type IS NULL OR booking_type = '');
UPDATE vendor_categories
SET booking_type = 'event', allowed_booking_types = ARRAY['event']
WHERE id IN ('festival-event', 'party-promoter', 'surf-competition', 'community-charity-event', 'music-night')
  AND (booking_type IS NULL OR booking_type = '');
