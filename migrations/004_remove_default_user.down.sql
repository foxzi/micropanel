-- Restore default admin user (password: admin)
INSERT OR IGNORE INTO users (email, password_hash, role, is_active)
VALUES ('admin@localhost', '$2a$10$bjUGCUkEk/UHFfo/xT1/IusJ1heW2yLQ7v5Y4CNZ/UDJ9oJLcK9JW', 'admin', 1);
