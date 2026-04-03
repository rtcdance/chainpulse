# Phase 287 - GraphQL Query Source Posture

## Status
Status: Approved

## Summary

Extend the event/query source-surfacing line into the GraphQL protocol surface
by exposing a compact `querySourcePosture` on GraphQL event results for the
`eventsByName` read path.

## Problem

After phase 286, the HTTP event/query data plane had a strong
query-service-backed baseline, but that source-surfacing work was still mostly
confined to the HTTP event routes.

GraphQL already had a real event read path in `eventsByName`, including cache
and event-store behavior, but callers still had no compact way to tell whether
the current result came from:

- a cache hit
- or the live event store path

## Decision

Expose a compact `querySourcePosture` field on GraphQL event results and use it
for the `eventsByName` resolver with two initial source semantics:

- `graphql-cache-hit`
- `graphql-event-store`

Keep the change additive:

- no GraphQL envelope rewrite
- no schema-wide result reshaping
- only a small source-surfacing extension on an existing event result type

## Scope

In scope:

- GraphQL event type enrichment
- `eventsByName` resolver source surfacing
- focused GraphQL resolver tests

Out of scope:

- GraphQL-wide query meta envelopes
- new GraphQL endpoints
- cross-protocol parity beyond this small source-surfacing slice

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase287-graphql-query-source-posture.md`

## Exit Criteria

- GraphQL event results can expose a compact `querySourcePosture`.
- `eventsByName` distinguishes cache hits from live event-store reads.
- Focused GraphQL tests and package-level API tests remain green.
