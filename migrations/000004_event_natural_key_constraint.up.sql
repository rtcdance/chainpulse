-- Add unique constraint on event natural key to prevent duplicate indexing
-- of the same on-chain event (identified by chain + block + tx + log_index).
ALTER TABLE events_metadata
    ADD CONSTRAINT uq_events_metadata_natural_key
    UNIQUE (chain_id, block_number, transaction_hash, log_index);
