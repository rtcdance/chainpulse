# Phase 273 - Event Query Response Meta

## Status
Status: Approved

## Summary

Add a stable `meta` block to event query responses so callers can see where a
response came from and how complete its attached metadata is.

## Problem

The current event query surface can return useful event payloads, but it does
not clearly tell the caller:

- whether the response came from the domain-first path or the retrieval path
- whether event metadata was fully present, partially present, or absent

That makes the query data plane harder to reason about even when the returned
data is correct.

## Decision

Extend `pkg/plugins/api/event_query_handler.go` so `QueryResponse` includes a
small `meta` block with:

- `source`
- `metadataCompleteness`
- `resultCount`

Use that shape across both:

- single-event responses
- list responses

## Scope

In scope:

- `QueryResponse` contract enrichment
- handler-side meta assembly
- domain-first and retrieval fallback coverage
- runtime route coverage through gateway integration

Out of scope:

- changes to storage/query execution internals
- new list/search endpoints
- full query-service migration for list paths

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase273-event-query-response-meta.md`

## Exit Criteria

- Event query responses expose a stable query meta block.
- The meta block distinguishes domain-first and retrieval-backed responses.
- The meta block exposes metadata completeness for list and single-event paths.
