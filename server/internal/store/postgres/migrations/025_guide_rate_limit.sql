INSERT INTO platform_settings (key, value) VALUES ('guide_rate_limit_rpm', '15') ON CONFLICT (key) DO NOTHING;
