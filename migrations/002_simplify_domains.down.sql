-- Restore is_primary column in domains
CREATE TABLE domains_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    hostname TEXT NOT NULL UNIQUE,
    is_primary INTEGER NOT NULL DEFAULT 0,
    ssl_enabled INTEGER NOT NULL DEFAULT 0,
    ssl_expires_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE CASCADE
);

INSERT INTO domains_old (id, site_id, hostname, is_primary, created_at)
SELECT id, site_id, hostname, 0, created_at FROM domains;

DROP TABLE domains;
ALTER TABLE domains_old RENAME TO domains;

CREATE INDEX IF NOT EXISTS idx_domains_site ON domains(site_id);

-- Remove SSL and www_alias fields from sites (recreate table)
CREATE TABLE sites_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    owner_id INTEGER NOT NULL,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO sites_old (id, name, owner_id, is_enabled, created_at, updated_at)
SELECT id, name, owner_id, is_enabled, created_at, updated_at FROM sites;

DROP TABLE sites;
ALTER TABLE sites_old RENAME TO sites;

CREATE INDEX IF NOT EXISTS idx_sites_owner ON sites(owner_id);
