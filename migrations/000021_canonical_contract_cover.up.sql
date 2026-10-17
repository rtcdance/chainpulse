-- Performance-only, additive index. Requires a maintenance window: ordinary
-- CREATE INDEX blocks writes while building. Reserve disk/growth headroom;
-- the measured 309775-row fixture used 65MiB. No event or revision mutation.
-- Include the original expression input so PostgreSQL can use index-only
-- count scans, and retain the full page order to avoid a contract-wide sort.
BEGIN;
SET LOCAL lock_timeout = '3s';
SET LOCAL statement_timeout = '5min';
CREATE INDEX idx_events_canonical_contract_cover
    ON blockchain_events (chain_id, lower(contract_address), block_number DESC, log_index DESC, id DESC)
    INCLUDE (contract_address)
    WHERE status<>'reorged' AND event_state->>'canonicality'='canonical';
COMMIT;
