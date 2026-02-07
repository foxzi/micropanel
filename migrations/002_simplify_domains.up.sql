-- Add SSL and www_alias fields to sites table
-- Site name is now the primary hostname

ALTER TABLE sites ADD COLUMN ssl_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sites ADD COLUMN ssl_expires_at DATETIME;
ALTER TABLE sites ADD COLUMN www_alias INTEGER NOT NULL DEFAULT 1;

-- Remove is_primary from domains (all remaining domains are aliases)
-- SQLite doesn't support DROP COLUMN directly, so we recreate the table

CREATE TABLE domains_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    hostname TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
);

INSERT INTO domains_new (id, site_id, hostname, created_at)
SELECT id, site_id, hostname, created_at FROM domains WHERE is_primary = 0;

DROP TABLE domains;
ALTER TABLE domains_new RENAME TO domains;

CREATE INDEX IF NOT EXISTS idx_domains_site ON domains(site_id);
