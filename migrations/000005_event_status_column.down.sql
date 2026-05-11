DROP INDEX IF EXISTS idx_event_status;
ALTER TABLE blockchain_events DROP COLUMN IF EXISTS status;
