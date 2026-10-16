-- A cache revision describes every event mutation, including legacy SQL
-- writers that do not advance a checkpoint or a reorg generation.
CREATE TABLE canonical_query_revisions (
    chain_id TEXT PRIMARY KEY CHECK (length(chain_id) > 0),
    revision BIGINT NOT NULL CHECK (revision > 0),
    epoch TEXT NOT NULL DEFAULT md5(random()::text || clock_timestamp()::text)
        CHECK (epoch ~ '^[0-9a-f]{32}$')
);

CREATE FUNCTION chainpulse_bump_query_revision() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
DECLARE
    affected TEXT[];
    chain_key TEXT;
BEGIN
    IF TG_OP = 'TRUNCATE' THEN
        UPDATE public.canonical_query_revisions SET revision = revision + 1;
        RETURN NULL;
    ELSIF TG_OP = 'INSERT' THEN
        affected := ARRAY[NEW.chain_id];
    ELSIF TG_OP = 'DELETE' THEN
        affected := ARRAY[OLD.chain_id];
    ELSE
        affected := ARRAY[OLD.chain_id, NEW.chain_id];
    END IF;
    -- A chain move invalidates both scopes; stable ordering avoids opposite
    -- moves acquiring these revision locks in the opposite order.
    FOR chain_key IN SELECT DISTINCT value FROM unnest(affected) AS value ORDER BY value
    LOOP
        INSERT INTO public.canonical_query_revisions(chain_id, revision)
        VALUES (chain_key, 1)
        ON CONFLICT (chain_id) DO UPDATE
            SET revision = canonical_query_revisions.revision + 1;
    END LOOP;
    RETURN NULL;
END;
$$;

CREATE TRIGGER bump_canonical_query_revision
AFTER INSERT OR UPDATE OR DELETE ON blockchain_events
FOR EACH ROW EXECUTE FUNCTION chainpulse_bump_query_revision();

CREATE TRIGGER truncate_canonical_query_revision
AFTER TRUNCATE ON blockchain_events
FOR EACH STATEMENT EXECUTE FUNCTION chainpulse_bump_query_revision();

INSERT INTO canonical_query_revisions(chain_id, revision)
SELECT DISTINCT chain_id, 1 FROM blockchain_events;
