-- Live offer stats: a real views counter, and updated_at so the server can
-- pick a deterministic "headline" offer to sync into a vendor's Place
-- (offer IDs are random hex, not chronological).
ALTER TABLE offers ADD COLUMN views INTEGER NOT NULL DEFAULT 0;
ALTER TABLE offers ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Links a claim back to the specific vendor Offer it was claimed from.
-- Nullable: existing claims predate this column, and claims may still
-- arrive from an older client build that doesn't send offerId.
ALTER TABLE claims ADD COLUMN offer_id TEXT;
CREATE INDEX claims_offer_id_idx ON claims (offer_id);

-- Enforces the one-place-per-vendor assumption the offer->place sync
-- relies on (already true in practice for seeded data).
CREATE UNIQUE INDEX places_vendor_id_idx ON places (vendor_id) WHERE vendor_id IS NOT NULL;

-- offers.claimed / offers.redeemed remain on the table (not dropped, to
-- keep this migration additive/non-destructive) but are no longer written
-- meaningfully by application code and no longer read into domain.Offer —
-- claimed/redeemed are now computed live from the claims table instead.
