-- Add status column to blockchain_events for reorg soft-delete tracking.
-- Existing rows default to 'pending' (matches the NOT NULL DEFAULT in the DDL).
ALTER TABLE blockchain_events ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'pending';

-- Index for efficient reorg queries (mark events in a block range as reorged).
CREATE INDEX IF NOT EXISTS idx_event_status ON blockchain_events(status);
