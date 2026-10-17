-- Serves the hot /events canonical query path (QueryDomainEvents) on
-- blockchain_events. EXPLAIN (ANALYZE, BUFFERS) at 500k rows (125k/chain,
-- 4 chains) measured on 2026-10-15 (build/acceptance/20261014-working/
-- t11-cache-explain.log):
--
--   count(*) WHERE chain_id=$1 AND status<>'reorged'
--     before: Parallel Bitmap Heap Scan, 362 ms
--     after:  Index Only Scan (Heap Fetches: 0), 23 ms
--   page WHERE chain_id=$1 AND status<>'reorged'
--          AND lower(contract_address)=$2 ORDER BY block_number DESC,... LIMIT 50
--     before: full backward scan, 461-529 ms (500k rows filtered)
--     after:  Index Scan, 0.11-0.32 ms
--
-- Partial on status<>'reorged' to match the default QueryDomainEvents
-- predicate; the expression column lower(contract_address) matches
-- domainPredicate's contractAddress binding.
CREATE INDEX IF NOT EXISTS idx_events_live_chain_contract
    ON blockchain_events (chain_id, lower(contract_address), block_number DESC)
    WHERE status<>'reorged';
