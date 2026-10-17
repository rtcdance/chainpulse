-- Normal application rollback retains this additive schema and its database
-- guards. Removing verified proof requires an explicit backup/restore procedure
-- while all writers are stopped; never erase a finalized boundary silently.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM chain_finality_observations WHERE tag='finalized' AND proof_at IS NOT NULL) THEN
  RAISE EXCEPTION 'retain schema17 for application rollback; verified finality proof requires backup/restore before schema removal';
 END IF;
END $$;
DROP TRIGGER guard_finality_events ON blockchain_events;
DROP TRIGGER guard_finality_blocks ON blocks;
DROP TRIGGER guard_finality_checkpoint ON indexing_state;
DROP FUNCTION chainpulse_guard_finality_events();
DROP FUNCTION chainpulse_guard_finality_blocks();
DROP FUNCTION chainpulse_guard_finality_checkpoint();
DROP FUNCTION chainpulse_verified_finalized_height(TEXT);
DROP TABLE chain_finality_observations;
