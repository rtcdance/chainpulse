-- Add correlation_id column to blockchain_events for cross-chain event correlation.
-- Events that represent the same logical operation across different chains
-- (e.g. a bridge transfer on Ethereum + Polygon) share the same correlation_id.
-- NULL means the event is not correlated across chains.

ALTER TABLE blockchain_events ADD COLUMN IF NOT EXISTS correlation_id VARCHAR(128);

CREATE INDEX IF NOT EXISTS idx_events_correlation_id ON blockchain_events(correlation_id);