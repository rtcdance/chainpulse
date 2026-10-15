-- Additive state axes. Legacy finalized is deliberately NOT finality evidence.
ALTER TABLE blockchain_events ADD COLUMN event_state JSONB NOT NULL DEFAULT
'{"canonicality":"canonical","processing":"pending","finality":{"state":"unknown","confidence":"unknown","source":"unknown"}}';
UPDATE blockchain_events SET event_state = jsonb_build_object(
 'canonicality', CASE WHEN status='reorged' THEN 'orphaned' ELSE 'canonical' END,
 'processing', CASE WHEN status IN ('confirmed','finalized') THEN 'processed' WHEN status='failed' THEN 'failed' ELSE 'pending' END,
 'finality', jsonb_build_object('state','unknown','confidence','unknown','source','unknown'));
ALTER TABLE blockchain_events ADD CONSTRAINT event_state_valid CHECK ((
 jsonb_typeof(event_state)='object'
 AND event_state->>'canonicality' IN ('canonical','orphaned')
 AND event_state->>'processing' IN ('pending','processed','failed','raw_only')
 AND jsonb_typeof(event_state->'finality')='object'
 AND event_state#>>'{finality,source}' IS NOT NULL
 AND (
  (event_state#>>'{finality,state}'='unknown' AND event_state#>>'{finality,confidence}' IN ('unknown','degraded'))
  OR (
   event_state->>'canonicality'='canonical'
   AND ((event_state#>>'{finality,state}'='safe' AND event_state#>>'{finality,confidence}'='safe' AND event_state#>>'{finality,source}'='rpc:safe')
     OR (event_state#>>'{finality,state}'='finalized' AND event_state#>>'{finality,confidence}'='finalized' AND event_state#>>'{finality,source}'='rpc:finalized'))
   AND jsonb_typeof(event_state#>'{finality,observed_at}')='string'
   AND (event_state#>>'{finality,observed_at}')::timestamptz IS NOT NULL
   AND event_state#>>'{finality,head_hash}' ~ '^0x[0-9a-fA-F]{64}$'
   AND jsonb_typeof(event_state#>'{finality,head_number}')='number'
   AND (event_state#>>'{finality,head_number}')::numeric >= block_number
  )
 )
) IS TRUE);

-- Compatibility window for old binaries which only write status. Reorg must
-- never erase processing success or turn historical status into RPC evidence.
CREATE FUNCTION chainpulse_sync_legacy_event_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='INSERT' OR NEW.status IS DISTINCT FROM OLD.status THEN
  NEW.event_state=jsonb_set(NEW.event_state,'{canonicality}',to_jsonb(CASE WHEN NEW.status='reorged' THEN 'orphaned' ELSE 'canonical' END));
  IF NEW.status IN ('confirmed','finalized') THEN
   NEW.event_state=jsonb_set(NEW.event_state,'{processing}','"processed"');
  ELSIF NEW.status='failed' THEN
   NEW.event_state=jsonb_set(NEW.event_state,'{processing}','"failed"');
  END IF;
  IF NEW.status='reorged' THEN
   NEW.event_state=jsonb_set(NEW.event_state,'{finality}','{"state":"unknown","confidence":"unknown","source":"reorg"}');
  END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER sync_legacy_event_state BEFORE INSERT OR UPDATE OF status ON blockchain_events
FOR EACH ROW EXECUTE FUNCTION chainpulse_sync_legacy_event_state();
