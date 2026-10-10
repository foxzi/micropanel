-- State of the latest certificate issuance, so an API client can request a
-- certificate asynchronously and poll the site for the outcome.
ALTER TABLE sites ADD COLUMN ssl_status TEXT NOT NULL DEFAULT 'none'
    CHECK (ssl_status IN ('none', 'issuing', 'issued', 'failed'));
ALTER TABLE sites ADD COLUMN ssl_error TEXT NOT NULL DEFAULT '';

UPDATE sites SET ssl_status = 'issued' WHERE ssl_enabled = 1;
