-- SQLite does not support DROP COLUMN before 3.35.0, so recreate the table
CREATE TABLE sites_backup AS SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at FROM sites;
DROP TABLE sites;
CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    ssl_enabled INTEGER NOT NULL DEFAULT 0,
    ssl_expires_at DATETIME,
    www_alias INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);
INSERT INTO sites SELECT id, name, owner_id, is_enabled, ssl_enabled, ssl_expires_at, www_alias, created_at, updated_at FROM sites_backup;
DROP TABLE sites_backup;
