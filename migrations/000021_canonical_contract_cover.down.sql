-- Old applications and schema20 cache authority remain valid after rollback.
BEGIN;
SET LOCAL lock_timeout = '3s';
SET LOCAL statement_timeout = '5min';
DROP INDEX idx_events_canonical_contract_cover;
COMMIT;
