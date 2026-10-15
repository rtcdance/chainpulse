-- Durable per-chain reorg generation (T06).
--
-- A reorg bumps the chain's generation; the value fences derived caches
-- (cache.CacheScope.Generation) and stamps reorg_jobs/reorg_outbox rows, so a
-- reader can tell state written before a reorg from state written after it.
-- One row per chain; created on first reorg, absent means generation 0.
CREATE TABLE IF NOT EXISTS reorg_generations (
    chain_id   VARCHAR(64) PRIMARY KEY,
    generation BIGINT      NOT NULL DEFAULT 0,
    updated_at TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);
