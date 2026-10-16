-- Additive per-chain RPC evidence. Display freshness and durable finalized
-- protection are separate: degraded polling retains the last verified proof.
CREATE TABLE chain_finality_observations (
 chain_id VARCHAR(64) NOT NULL CHECK (chain_id <> ''),
 tag VARCHAR(16) NOT NULL CHECK (tag IN ('safe','finalized')),
 status VARCHAR(16) NOT NULL CHECK (status IN ('known','degraded')),
 checked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 proof_at TIMESTAMPTZ,
 head_number BIGINT CHECK (head_number >= 0),
 head_hash VARCHAR(66),
 genesis_hash VARCHAR(66),
 generation BIGINT CHECK (generation >= 0),
 PRIMARY KEY(chain_id,tag),
 CONSTRAINT finality_proof_complete CHECK (
  (proof_at IS NULL AND head_number IS NULL AND head_hash IS NULL AND genesis_hash IS NULL AND generation IS NULL AND status='degraded')
  OR (proof_at IS NOT NULL AND head_number IS NOT NULL AND head_hash IS NOT NULL
   AND genesis_hash IS NOT NULL AND generation IS NOT NULL
   AND head_hash ~ '^0x[0-9a-f]{64}$' AND genesis_hash ~ '^0x[0-9a-f]{64}$')
 )
);

-- Keep verified finality effective during application rollback: older binaries
-- cannot mark protected effects orphaned or erase their header/checkpoint.
CREATE FUNCTION chainpulse_verified_finalized_height(p_chain TEXT) RETURNS BIGINT
LANGUAGE SQL STABLE AS $$
 SELECT head_number FROM chain_finality_observations
 WHERE chain_id=p_chain AND tag='finalized' AND proof_at IS NOT NULL
$$;

CREATE FUNCTION chainpulse_guard_finality_events() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE boundary BIGINT;
BEGIN
 boundary=chainpulse_verified_finalized_height(OLD.chain_id);
 IF boundary IS NOT NULL AND OLD.block_number<=boundary THEN
  IF TG_OP='DELETE' THEN RAISE EXCEPTION 'verified finalized boundary protects event deletion' USING ERRCODE='23514'; END IF;
  IF NEW.status IS NULL OR NEW.status='reorged' OR NEW.event_state->>'canonicality'='orphaned'
    OR NEW.chain_id IS DISTINCT FROM OLD.chain_id OR NEW.block_hash IS DISTINCT FROM OLD.block_hash
    OR NEW.block_number IS DISTINCT FROM OLD.block_number OR NEW.id IS DISTINCT FROM OLD.id
    OR NEW.transaction_hash IS DISTINCT FROM OLD.transaction_hash OR NEW.log_index IS DISTINCT FROM OLD.log_index
    OR NEW.contract_address IS DISTINCT FROM OLD.contract_address OR NEW.timestamp IS DISTINCT FROM OLD.timestamp
    OR NEW.event_data->'raw' IS DISTINCT FROM OLD.event_data->'raw' OR NEW.event_data->'topics' IS DISTINCT FROM OLD.event_data->'topics' THEN
   RAISE EXCEPTION 'verified finalized boundary protects event identity' USING ERRCODE='23514';
  END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER guard_finality_events BEFORE UPDATE OR DELETE ON blockchain_events
 FOR EACH ROW EXECUTE FUNCTION chainpulse_guard_finality_events();

CREATE FUNCTION chainpulse_guard_finality_blocks() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE boundary BIGINT;
BEGIN
 boundary=chainpulse_verified_finalized_height(OLD.chain_id);
 IF boundary IS NOT NULL AND OLD.number<=boundary THEN
  IF TG_OP='DELETE' THEN RAISE EXCEPTION 'verified finalized boundary protects header deletion' USING ERRCODE='23514'; END IF;
  IF NEW.chain_id IS DISTINCT FROM OLD.chain_id OR NEW.hash IS DISTINCT FROM OLD.hash OR NEW.number IS DISTINCT FROM OLD.number OR NEW.parent_hash IS DISTINCT FROM OLD.parent_hash OR NEW.timestamp IS DISTINCT FROM OLD.timestamp THEN
   RAISE EXCEPTION 'verified finalized boundary protects header identity' USING ERRCODE='23514';
  END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER guard_finality_blocks BEFORE UPDATE OR DELETE ON blocks
 FOR EACH ROW EXECUTE FUNCTION chainpulse_guard_finality_blocks();

CREATE FUNCTION chainpulse_guard_finality_checkpoint() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE proof chain_finality_observations%ROWTYPE;
BEGIN
 SELECT * INTO proof FROM chain_finality_observations WHERE chain_id=OLD.chain_id AND tag='finalized' AND proof_at IS NOT NULL;
 IF FOUND THEN
  IF TG_OP='DELETE' THEN RAISE EXCEPTION 'verified finalized boundary protects checkpoint deletion' USING ERRCODE='23514'; END IF;
  IF NEW.chain_id IS DISTINCT FROM OLD.chain_id OR NEW.last_indexed_block IS NULL OR NEW.last_indexed_hash IS NULL OR NEW.last_indexed_block<proof.head_number
    OR (NEW.last_indexed_block=proof.head_number AND NEW.last_indexed_hash IS DISTINCT FROM proof.head_hash) THEN
   RAISE EXCEPTION 'verified finalized boundary protects checkpoint rewind' USING ERRCODE='23514';
  END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER guard_finality_checkpoint BEFORE UPDATE OR DELETE ON indexing_state
 FOR EACH ROW EXECUTE FUNCTION chainpulse_guard_finality_checkpoint();
