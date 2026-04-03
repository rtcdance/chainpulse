# Phase 295 - GraphQL Root Events Source Posture

## Status
Status: Approved

## Summary

Extend the GraphQL query-source-surfacing mini-baseline to the root `events`
connection so the generic paginated list path exposes the same compact source
signal as the rest of the GraphQL event read family.

## Problem

After phase 294, the GraphQL source-surfacing mini-baseline already covered:

- `event`
- `eventsByName`
- `eventsByAddress`
- `eventsByBlock`

But the root paginated `events` connection still returned event nodes without
the same `querySourcePosture`, which left the most generic event list path
outside the current mini-baseline.

## Decision

Expose `querySourcePosture` on the node payloads returned by GraphQL root
`events` connections using the same compact source semantics already used by
the GraphQL mini-baseline:

- `graphql-event-store`

Keep the change intentionally small:

- no GraphQL envelope rewrite
- no new top-level meta block
- only source posture on existing event nodes

## Scope

In scope:

- GraphQL root `events` source surfacing
- resolver and schema-builder parity for the root list path
- focused GraphQL resolver tests
- package-level API regression coverage

Out of scope:

- GraphQL cache semantics for the root paginated path
- broader GraphQL response meta redesign
- full GraphQL query-source parity claims

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase295-graphql-root-events-source-posture.md`

## Exit Criteria

- GraphQL root `events` connection nodes expose `querySourcePosture`.
- Focused GraphQL tests and package-level API tests remain green.
