-- Real booking system: a vendor defines BookableServices (what's being
-- booked), travellers create Bookings against them. No slot-materialization
-- table — availability is checked on demand against bookable_services'
-- weekly availability rule + the vendor's existing Hours/HolidayOverrides
-- (vendorProfileDoc) + capacity minus overlapping confirmed bookings,
-- mirroring vendor.go's isOpenNow on-demand-check philosophy. See
-- server/internal/domain/booking.go.

CREATE TABLE bookable_services (
    id              TEXT PRIMARY KEY,
    vendor_id       TEXT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    type            TEXT NOT NULL, -- session | table | stay
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    duration_mins   INTEGER NOT NULL DEFAULT 0,
    capacity        INTEGER NOT NULL DEFAULT 1,
    party_size_min  INTEGER NOT NULL DEFAULT 1,
    party_size_max  INTEGER NOT NULL DEFAULT 1,
    guests_per_room INTEGER NOT NULL DEFAULT 1,
    price_label     TEXT NOT NULL DEFAULT '',
    availability    JSONB, -- null for type='stay'
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX bookable_services_vendor_idx ON bookable_services (vendor_id);

CREATE TABLE bookings (
    id                TEXT PRIMARY KEY,
    code              TEXT NOT NULL UNIQUE,
    vendor_id         TEXT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    service_id        TEXT NOT NULL REFERENCES bookable_services(id) ON DELETE CASCADE,
    type              TEXT NOT NULL,
    place_id          TEXT NOT NULL DEFAULT '',
    traveller_id      TEXT NOT NULL DEFAULT '',
    start_at          TIMESTAMPTZ NOT NULL,
    end_at            TIMESTAMPTZ NOT NULL,
    party_size        INTEGER NOT NULL DEFAULT 1,
    status            TEXT NOT NULL DEFAULT 'confirmed',
    note              TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    cancelled_at      TIMESTAMPTZ,
    cancelled_by      TEXT NOT NULL DEFAULT '',
    checked_in_at     TIMESTAMPTZ,
    reminder_sent_at  TIMESTAMPTZ
);
CREATE INDEX bookings_vendor_idx ON bookings (vendor_id, start_at DESC);
CREATE INDEX bookings_traveller_idx ON bookings (traveller_id, start_at DESC);
-- Drives the overlap/capacity check (Store.CreateBooking / OverlappingBookingsCount).
CREATE INDEX bookings_service_window_idx ON bookings (service_id, status, start_at, end_at);
-- Drives the reminder scheduler's sweep.
CREATE INDEX bookings_reminder_idx ON bookings (status, start_at) WHERE reminder_sent_at IS NULL;
