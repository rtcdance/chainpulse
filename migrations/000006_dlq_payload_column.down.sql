DROP INDEX IF EXISTS idx_dlq_events_status_created;
ALTER TABLE dlq_events DROP COLUMN IF EXISTS payload;