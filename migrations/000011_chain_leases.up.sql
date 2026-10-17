-- Chain write leases (T05 owner fencing).
--
-- A chain must be written by at most one worker at a time. Kafka consumer-group
-- rebalances move partition ownership between workers, and the previous owner
-- may still be mid-batch when it loses the partition. Without a fencing token
-- it would keep committing, producing two writers for one chain.
--
-- Each acquisition bumps `generation`, so a lease row carries both "who owns it
-- now" (owner_token) and "how many times ownership has changed" (generation).
-- A commit that presents a token which is no longer the current owner is
-- rejected. `expires_at` lets a crashed worker's lease be taken over instead of
-- deadlocking the chain forever.
CREATE TABLE IF NOT EXISTS chain_leases (
    chain_id    VARCHAR(64) PRIMARY KEY,
    owner_token VARCHAR(128) NOT NULL,
    generation  BIGINT NOT NULL DEFAULT 1,
    acquired_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chain_leases_expires_at ON chain_leases(expires_at);
