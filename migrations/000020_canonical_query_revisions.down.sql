-- Disable/revert the cache-reading application before removing its authority.
DROP TRIGGER IF EXISTS truncate_canonical_query_revision ON blockchain_events;
DROP TRIGGER IF EXISTS bump_canonical_query_revision ON blockchain_events;
DROP FUNCTION IF EXISTS chainpulse_bump_query_revision();
DROP TABLE IF EXISTS canonical_query_revisions;
