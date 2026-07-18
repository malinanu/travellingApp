-- Time-limited PIN grants letting an owner give a manager temporary edit
-- access to Business Profile, which managers otherwise have zero standing
-- access to. See VendorProfileGrant in internal/domain for field semantics.
CREATE TABLE vendor_profile_grants (
    id                TEXT PRIMARY KEY,
    vendor_id         TEXT NOT NULL,
    grantee_id        TEXT NOT NULL,
    created_by        TEXT NOT NULL,
    pin_hash          TEXT NOT NULL,
    duration_mins     INT  NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    pin_expires_at    TIMESTAMPTZ NOT NULL,
    redeemed_at       TIMESTAMPTZ,
    access_expires_at TIMESTAMPTZ,
    revoked_at        TIMESTAMPTZ,
    revoked_by        TEXT
);
CREATE INDEX vendor_profile_grants_vendor_idx ON vendor_profile_grants (vendor_id);
CREATE INDEX vendor_profile_grants_grantee_idx ON vendor_profile_grants (grantee_id);
