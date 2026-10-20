-- Extra powers a super-admin can grant to an individual admin, on top of the
-- normal admin role. Super-admins (SUPER_ADMIN_EMAILS) hold every permission
-- implicitly. Currently only 'vendor_logins': changing a vendor account's
-- login email and password when a business is handed over to its real owner.
ALTER TABLE users ADD COLUMN admin_permissions TEXT[] NOT NULL DEFAULT '{}';
