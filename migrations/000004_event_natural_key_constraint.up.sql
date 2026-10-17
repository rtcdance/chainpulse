-- Add unique constraint on event natural key to prevent duplicate indexing
-- of the same on-chain event (identified by chain + block + tx + log_index).
--
-- NOTE: this constraint targets `blockchain_events`, the table created by
-- 000001_init_schema.up.sql. The previous version targeted `events_metadata`,
-- which no migration ever creates, so a fresh migrate aborted here with
-- `relation "events_metadata" does not exist` (see T02 status doc).
--
-- The natural key intentionally does NOT yet include block_hash: widening it to
-- (chain_id, block_hash, transaction_hash, log_index) is T04's job (finding F06).
ALTER TABLE blockchain_events
    DROP CONSTRAINT IF EXISTS uq_blockchain_events_natural_key;

ALTER TABLE blockchain_events
    ADD CONSTRAINT uq_blockchain_events_natural_key
    UNIQUE (chain_id, block_number, transaction_hash, log_index);
