-- Roll application readers back before removing this additive projection.
-- Legacy status, physical IDs, raw payloads and orphan rows are preserved.
DROP TRIGGER IF EXISTS sync_legacy_event_state ON blockchain_events;
DROP FUNCTION IF EXISTS chainpulse_sync_legacy_event_state();
ALTER TABLE blockchain_events DROP CONSTRAINT IF EXISTS event_state_valid;
ALTER TABLE blockchain_events DROP COLUMN IF EXISTS event_state;
