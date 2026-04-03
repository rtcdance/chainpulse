# Phase 291 - GraphQL Events By Address Source Posture

## Status
Status: Approved

## Summary

Extend the GraphQL query-source-surfacing pilot to the `eventsByAddress` read
path so the pilot now covers a second list-style event query in addition to the
single-event path.

## Problem

After phase 289, GraphQL source surfacing already covered:

- `event`
- `eventsByName`

But `eventsByAddress` still lacked the same compact source signal, leaving the
pilot weaker on another common event-read shape.

## Decision

Expose `querySourcePosture` on GraphQL `eventsByAddress` results using the same
compact pilot semantics:

- `graphql-cache-hit`
- `graphql-event-store`

Keep the change additive and small:

- no GraphQL envelope rewrite
- no new top-level GraphQL meta block
- only source posture on existing event results

## Scope

In scope:

- GraphQL `eventsByAddress` source surfacing
- focused GraphQL resolver tests
- package-level API regression coverage

Out of scope:

- broad GraphQL query meta envelopes
- full GraphQL parity claims
- new query execution paths

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase291-graphql-events-by-address-source-posture.md`

## Exit Criteria

- GraphQL `eventsByAddress` results can expose `querySourcePosture`.
- The resolver distinguishes cache hits from live event-store reads.
- Focused GraphQL tests and package-level API tests remain green.
