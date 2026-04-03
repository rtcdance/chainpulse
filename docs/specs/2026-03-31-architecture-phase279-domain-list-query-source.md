# Phase 279 - Domain List Query Source

## Status
Status: Approved

## Summary

Promote `GET /events` from a retrieval-only list path to a first-step
domain/query-backed list path with real query source exposure.

## Problem

The event/query data plane had already grown a strong response `meta` contract,
but the main list path still remained retrieval-only.

That meant the contract could describe more query semantics than the list path
could actually surface.

## Decision

Update `pkg/plugins/api/event_query_handler.go` so `HandleGetAllEvents`:

1. attempts the optional domain query service first
2. preserves retrieval fallback
3. exposes the real domain query source when available

This allows `GET /events` to surface sources such as:

- `cache`
- `mongodb`
- `postgresql`

without rewriting the whole query stack.

## Scope

In scope:

- domain-query-first list path for `GET /events`
- retrieval fallback preservation
- handler-level tests
- gateway runtime route coverage

Out of scope:

- migration of every list path to domain query
- storage/query execution rewrites
- stronger consistency guarantees than the current contracts expose

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase279-domain-list-query-source.md`

## Exit Criteria

- `GET /events` can use the domain query service when available.
- The response `meta` exposes the real domain query source for the list path.
- Retrieval fallback remains available and covered by tests.
