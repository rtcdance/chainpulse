DROP INDEX IF EXISTS idx_events_correlation_id;

ALTER TABLE blockchain_events DROP COLUMN IF EXISTS correlation_id;