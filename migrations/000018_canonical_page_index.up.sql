-- Serves the hot /events canonical query path (QueryDomainEvents) and
-- CanonicalEventStats. Measured 2026-10-16 at 215k rows (107k/chain) on the
-- T14 workload fixture (build/acceptance/20261014-working/t14-e14):
--
--   page WHERE chain_id=$1 AND status<>'reorged'
--         AND event_state->>'canonicality'='canonical'
--         ORDER BY block_number DESC,log_index DESC,id DESC LIMIT 50 OFFSET 21700
--     before: Parallel Seq Scan + external merge sort, 203 ms (disk-backed sort)
--     after:  Index Only Scan (Heap Fetches: 0), 3.4 ms
--   count(*) WHERE chain_id=ANY($1) AND status<>'reorged'
--         AND event_state->>'canonicality'='canonical'
--     before: Parallel Seq Scan, 214 ms
--     after:  Index Only Scan, 52 ms
--   GROUP BY event_name over the same predicate
--     before: heap-backed HashAggregate fed by seq scan, 866 ms
--     after:  Index Only Scan, 104 ms
--
-- WITHOUT INDEX is the canonical row shape (status + canonicality), so the
-- partial predicate matches the default QueryDomainEvents filter and the
-- CanonicalEventStats counts exactly. event_name is a trailing INCLUDE-style
-- column so the stats GROUP BY stays index-only.
CREATE INDEX IF NOT EXISTS idx_events_canonical_chain_page
    ON blockchain_events (chain_id, block_number DESC, log_index DESC, id DESC, event_name)
    WHERE status<>'reorged' AND event_state->>'canonicality'='canonical';
