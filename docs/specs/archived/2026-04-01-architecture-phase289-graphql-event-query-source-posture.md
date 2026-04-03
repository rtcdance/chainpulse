# Phase 289 - GraphQL Event Query Source Posture

## Status
Status: Approved

## Summary

Extend the GraphQL query-source-surfacing pilot from the `eventsByName` list
path to the single-event `event` read path so the pilot now covers both a list
and a single-item event query shape.

## Problem

After phase 287, GraphQL query-source surfacing existed only on:

- `eventsByName`

That was enough to prove a pilot, but still left the GraphQL single-event path
without the same compact source signal. This made the pilot narrower than it
needed to be and left `event` inconsistent with the rest of the emerging
source-surfacing story.

## Decision

Expose `querySourcePosture` on the GraphQL single-event `event` resolver using
the same compact source semantics as the current pilot:

- `graphql-cache-hit`
- `graphql-event-store`

Keep the change additive and small:

- no GraphQL envelope rewrite
- no new top-level query meta block
- only a compact source posture on the existing event result

## Scope

In scope:

- GraphQL single-event source surfacing
- focused resolver tests
- package-level API regression coverage

Out of scope:

- broad GraphQL query meta envelopes
- protocol-wide source parity claims
- new query execution paths

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase289-graphql-event-query-source-posture.md`

## Exit Criteria

- GraphQL single-event results can expose `querySourcePosture`.
- The `event` resolver distinguishes cache hits from live event-store reads.
- Focused GraphQL tests and package-level API tests remain green.
