-- Widen the event natural key to include block_hash (invariant I02, finding F06).
--
-- The physical event identity is (chain_id, block_hash, transaction_hash,
-- log_index). The previous constraint used block_number instead of block_hash,
-- so the same transaction re-included at the same height on a different branch
-- (a reorg that replaces a block) would collide with the block it replaced.
--
-- The database is expected to be empty at this point (this is a pre-release
-- schema correction), so the switch is done directly rather than via
-- expand -> backfill -> verify -> switch. If applied to a populated database,
-- backfill/verification would be required first.
ALTER TABLE blockchain_events
    DROP CONSTRAINT IF EXISTS uq_blockchain_events_natural_key;

ALTER TABLE blockchain_events
    ADD CONSTRAINT uq_blockchain_events_natural_key
    UNIQUE (chain_id, block_hash, transaction_hash, log_index);
