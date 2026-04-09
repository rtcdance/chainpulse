## Status
Status: Approved

## Summary

The Docker microservice acceptance stack can start multiple services that initialize the shared PostgreSQL metadata schema concurrently. PostgreSQL may still emit duplicate catalog-key errors during `CREATE TABLE IF NOT EXISTS` or `CREATE INDEX IF NOT EXISTS`, which currently crashes startup.

## Decision

- Treat known PostgreSQL duplicate catalog conflicts during metadata schema bootstrap as idempotent success.
- Apply the tolerance to both table creation and index creation in the PostgreSQL event metadata store.
- Add a focused unit test for the conflict matcher.

## Acceptance

- Concurrent metadata store bootstrap no longer crashes on known duplicate catalog conflicts.
- Targeted Go tests remain green.
