# Phase 283 - Event Query Chain Domain Path

## Status
Status: Approved

## Summary

Extend the event/query data plane so `GET /events/chain/{chainId}` can use the
domain query path before falling back to retrieval, bringing a real non-root
read path onto the same query-service-backed contract surface as `GET /events`.

## Problem

After phase 282, the event/query contract had stronger shared assembly
boundaries, but query-service-backed execution still lived mainly on:

- `GET /events`
- domain-first `GET /events/{id}`

The chain-filtered list path still remained retrieval-only even though it was a
natural read surface for the same query-service-backed contract evolution.

## Decision

Add a domain-query-first path to:

- `GET /events/chain/{chainId}`

Use a MongoDB-style domain query request with an explicit `chainId` filter and
surface the result through the existing event query contract using:

- `queryPath=domain-chain`
- domain query source posture
- existing consistency/reliability semantics

Fall back to retrieval when the domain query path fails or does not return
results.

## Scope

In scope:

- domain-query-first chain list path
- query meta exposure for the chain route
- handler-level coverage
- gateway runtime route coverage

Out of scope:

- contract/name path migration
- query execution rewrites
- storage-level consistency guarantees

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase283-event-query-chain-domain-path.md`

## Exit Criteria

- `GET /events/chain/{chainId}` can use the domain query path before retrieval.
- The chain route exposes query-service-backed `meta` semantics through
  `domain-chain`.
- Handler and gateway route tests cover the new domain-query-first chain path.
