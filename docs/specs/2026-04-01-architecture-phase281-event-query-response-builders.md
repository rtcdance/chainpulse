# Phase 281 - Event Query Response Builders

## Status
Status: Approved

## Summary

Extract stable shared response builders for single-event and paginated event
query responses so externally visible query `meta` semantics are assembled
through one path instead of being duplicated across handler routes.

## Problem

After phase 280, the event/query data plane had a stronger externally visible
contract, but response assembly was still repeated in multiple handler paths:

- `GET /events`
- `GET /events/{id}`
- `GET /events/chain/{chainId}`
- `GET /events/contract/{address}`
- `GET /events/name/{eventName}`

That duplication made the contract more fragile than it needed to be. The
individual `meta` fields were already shared, but the response envelope itself
was still stitched together ad hoc at each route.

## Decision

Add shared event query response builders for:

- single-event responses
- paginated list responses

Then route existing handler paths through those builders so event/query data
plane responses share one stable assembly path for:

- `data`
- `pagination`
- `meta`
- `timestamp`

## Scope

In scope:

- shared response envelope builders for event query handlers
- handler migration to the shared builders
- focused tests for the builders

Out of scope:

- query execution rewrites
- new response fields
- cross-package API abstraction beyond the current handler package

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase281-event-query-response-builders.md`

## Exit Criteria

- Event query handlers assemble single-item and paginated responses through
  shared builders.
- Existing handler behavior remains green in package tests.
- The event/query response envelope becomes less route-local and more stable as
  an externally visible contract surface.
