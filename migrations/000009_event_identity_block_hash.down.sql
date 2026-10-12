-- Revert the natural key to the pre-F06 form (block_number instead of block_hash).
ALTER TABLE blockchain_events
    DROP CONSTRAINT IF EXISTS uq_blockchain_events_natural_key;

ALTER TABLE blockchain_events
    ADD CONSTRAINT uq_blockchain_events_natural_key
    UNIQUE (chain_id, block_number, transaction_hash, log_index);
