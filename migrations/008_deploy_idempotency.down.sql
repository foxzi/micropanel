DROP INDEX IF EXISTS idx_deploys_site_idempotency_key;
ALTER TABLE deploys DROP COLUMN idempotency_key;
