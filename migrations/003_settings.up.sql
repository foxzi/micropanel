CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO settings (key, value) VALUES ('server_name', '');
INSERT INTO settings (key, value) VALUES ('server_notes', '');
