-- Tracks when each admin last dismissed the notification bell. Admin
-- notifications are computed live from current data (pending vendors,
-- flagged content, bad-review vendors) rather than stored as discrete
-- rows — see Store.AdminNotifications — so only this read-cursor needs
-- persisting, one row per admin.
CREATE TABLE admin_notification_reads (
    admin_id     TEXT PRIMARY KEY,
    last_read_at TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01'
);
