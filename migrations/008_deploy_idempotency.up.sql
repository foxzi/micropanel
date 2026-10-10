-- Idempotency key for API deploys: a retry with the key of a successful
-- deploy returns that deploy instead of unpacking the archive again.
ALTER TABLE deploys ADD COLUMN idempotency_key TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_deploys_site_idempotency_key
    ON deploys(site_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
