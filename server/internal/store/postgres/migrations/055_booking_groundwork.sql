-- Groundwork for paid bookings.
--
-- 1. capacity_unit: a service's capacity can now count guests (a tour of 12,
--    a class of 20) instead of concurrent bookings. Existing services keep
--    counting bookings.
-- 2. archived_at: deleting a service that already has bookings archives it
--    instead, so booking (and, from 057, payment and ledger) history keeps
--    its reference.
-- 3. hold_expires_at: see below.
-- 4. Bookings no longer disappear when their vendor or service is deleted:
--    the CASCADE foreign keys from 025 become RESTRICT. A vendor with no
--    bookings can still be deleted (its services still cascade); one with
--    booking history cannot be hard-deleted.

ALTER TABLE bookable_services
    ADD COLUMN IF NOT EXISTS capacity_unit TEXT NOT NULL DEFAULT 'bookings'
        CHECK (capacity_unit IN ('bookings', 'guests')),
    ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ;

-- hold_expires_at: a priced booking holds its seats while awaiting payment
-- only until this time (see the awaiting_payment status).
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS hold_expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS bookings_hold_idx ON bookings (hold_expires_at) WHERE status = 'awaiting_payment';

ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_vendor_id_fkey;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_service_id_fkey;
ALTER TABLE bookings
    ADD CONSTRAINT bookings_vendor_id_fkey FOREIGN KEY (vendor_id) REFERENCES vendors(id) ON DELETE RESTRICT,
    ADD CONSTRAINT bookings_service_id_fkey FOREIGN KEY (service_id) REFERENCES bookable_services(id) ON DELETE RESTRICT;
