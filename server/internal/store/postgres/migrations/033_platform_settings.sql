CREATE TABLE platform_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL
);

INSERT INTO platform_settings (key, value) VALUES ('rate_limit_rpm', '200');
