# Phase 279 - Event Query Source Posture

## Status
Status: Approved

## Summary

Expose a compact query source posture so event query responses can distinguish
real execution sources such as cache, MongoDB, PostgreSQL, retrieval fallback,
and domain/query-service paths.

## Problem

After phase 278, event query responses exposed stronger path and consistency
semantics, but callers still lacked a dedicated posture that answered:

- what concrete execution source served this result?

This was especially important for the list path, where `domainQuery.Query(...)`
was already in play but not clearly surfaced as a query-service-backed path.

## Decision

Extend event query response `meta` with:

- `querySourcePosture`

Use it to expose compact source semantics such as:

- `domain-service`
- `query-service-direct`
- `cache-hit`
- `mongodb-live`
- `postgres-fallback`
- `retrieval-service`
- `retrieval-fallback`

## Scope

In scope:

- event query response meta enrichment
- handler-level tests
- gateway runtime route coverage

Out of scope:

- query execution rewrites
- new query endpoints
- storage-level consistency guarantees

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase279-event-query-source-posture.md`

## Exit Criteria

- Event query responses expose a compact source posture.
- Domain-first, fallback, retrieval-backed, and domain-list query paths are all
  covered by tests.
- The list query path now explicitly surfaces its query-service-backed posture.
