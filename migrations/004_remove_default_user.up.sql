-- Remove default admin user (if exists)
-- Users should create admin via CLI: micropanel user create -e admin@example.com -p password -r admin
DELETE FROM users WHERE email = 'admin@localhost';
