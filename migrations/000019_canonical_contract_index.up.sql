-- Serves the contract-filtered canonical query (QueryDomainEvents with
-- contractAddress, which binds lower(contract_address)) without a per-scan
-- lower() expression recompute. Measured 2026-10-16 on the T14 workload
-- fixture (build/acceptance/20261014-working/t14-e14, 123,675 canonical rows
-- on chain 31337):
--
--   count(*) WHERE chain_id=$1 AND lower(contract_address)=$2
--         AND status<>'reorged' AND event_state->>'canonicality'='canonical'
--     before: Seq Scan + Filter (lower(contract_address) = ...), 400 ms
--     after:  Index Only Scan, 118 ms
--
-- Same partial predicate as idx_events_canonical_chain_page (WITHOUT INDEX is
-- the canonical row shape: status + canonicality), so it matches exactly the
-- rows the contract filter may return.
CREATE INDEX IF NOT EXISTS idx_events_canonical_chain_contract
    ON blockchain_events (chain_id, lower(contract_address), block_number DESC)
    WHERE status<>'reorged' AND event_state->>'canonicality'='canonical';
