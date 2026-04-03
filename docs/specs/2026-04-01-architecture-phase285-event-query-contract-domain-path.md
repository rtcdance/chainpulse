# Phase 285 - Event Query Contract Domain Path

## Status
Status: Approved

## Summary

Extend the event/query data plane so `GET /events/contract/{address}` can use
the domain query path before falling back to retrieval, completing a symmetric
domain-query-first baseline across the three filter-list event read paths.

## Problem

After phases 283 and 284, the event/query contract had already expanded to:

- `GET /events/chain/{chainId}`
- `GET /events/name/{eventName}`

But the contract-address list route still remained retrieval-only, leaving the
filter-list family asymmetrical.

## Decision

Add a domain-query-first path to:

- `GET /events/contract/{address}`

Use a MongoDB-style domain query request with an explicit `contractAddress`
filter and surface the result through the existing event query contract using:

- `queryPath=domain-contract`
- domain query source posture
- existing consistency/reliability semantics

Fall back to retrieval when the domain query path fails or does not return
results.

## Scope

In scope:

- domain-query-first contract-address list path
- query meta exposure for the contract route
- handler-level coverage
- gateway runtime route coverage

Out of scope:

- new endpoint families
- query execution rewrites
- storage-level consistency guarantees

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase285-event-query-contract-domain-path.md`

## Exit Criteria

- `GET /events/contract/{address}` can use the domain query path before
  retrieval.
- The contract-address route exposes query-service-backed `meta` semantics
  through `domain-contract`.
- The chain/name/contract filter-list routes now form a symmetric
  domain-query-first baseline.
