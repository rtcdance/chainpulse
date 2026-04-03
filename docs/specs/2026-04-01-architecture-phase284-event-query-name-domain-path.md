# Phase 284 - Event Query Name Domain Path

## Status
Status: Approved

## Summary

Extend the event/query data plane so `GET /events/name/{eventName}` can use the
domain query path before falling back to retrieval, proving that the new
query-service-backed contract is not limited to the root or chain-filtered list
paths.

## Problem

After phase 283, the event/query contract had already expanded to:

- `GET /events`
- `GET /events/{id}`
- `GET /events/chain/{chainId}`

But the event-name list route still remained retrieval-only, leaving one more
real non-root read path outside the domain-query-first contract surface.

## Decision

Add a domain-query-first path to:

- `GET /events/name/{eventName}`

Use a MongoDB-style domain query request with an explicit `eventName` filter
and surface the result through the existing event query contract using:

- `queryPath=domain-name`
- domain query source posture
- existing consistency/reliability semantics

Fall back to retrieval when the domain query path fails or does not return
results.

## Scope

In scope:

- domain-query-first event-name list path
- query meta exposure for the name route
- handler-level coverage
- gateway runtime route coverage

Out of scope:

- contract-address path migration
- query execution rewrites
- storage-level consistency guarantees

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase284-event-query-name-domain-path.md`

## Exit Criteria

- `GET /events/name/{eventName}` can use the domain query path before
  retrieval.
- The event-name route exposes query-service-backed `meta` semantics through
  `domain-name`.
- Handler and gateway route tests cover the new domain-query-first event-name
  path.
