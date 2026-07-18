-- Cross-admin history of destructive actions (vendor status changes, feed
-- moderation, subscription upgrades, vendor-grant admin overrides). Actor
-- email/name are denormalized at write time so the log stays readable even
-- if the admin account is later deleted.
CREATE TABLE admin_audit_log (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    TEXT NOT NULL,
    actor_email TEXT NOT NULL DEFAULT '',
    actor_name  TEXT NOT NULL DEFAULT '',
    action      TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id   TEXT NOT NULL DEFAULT '',
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX admin_audit_log_id_idx ON admin_audit_log (id DESC);
CREATE INDEX admin_audit_log_target_idx ON admin_audit_log (target_type, target_id);
CREATE INDEX admin_audit_log_actor_idx ON admin_audit_log (actor_id);
