-- Add payload column to dlq_events for storing full event JSON for retry
ALTER TABLE dlq_events ADD COLUMN IF NOT EXISTS payload TEXT;

CREATE INDEX IF NOT EXISTS idx_dlq_events_status_created ON dlq_events(status, created_at);