# Phase 293 - GraphQL Events By Block Source Posture

## Status
Status: Approved

## Summary

Extend the GraphQL query-source-surfacing mini-baseline to the `eventsByBlock`
read path so the baseline now spans another event-list shape in addition to the
single-event, name-filtered, and address-filtered paths.

## Problem

After phase 292, the GraphQL source-surfacing mini-baseline already covered:

- `event`
- `eventsByName`
- `eventsByAddress`

But `eventsByBlock` still lacked the same compact source signal, leaving an
obvious event-list read path outside the current mini-baseline.

## Decision

Expose `querySourcePosture` on GraphQL `eventsByBlock` results using the same
compact source semantics already used by the mini-baseline:

- `graphql-event-store`

Keep the change intentionally small:

- no GraphQL envelope rewrite
- no new top-level meta block
- only source posture on existing event results

## Scope

In scope:

- GraphQL `eventsByBlock` source surfacing
- focused GraphQL resolver tests
- package-level API regression coverage

Out of scope:

- GraphQL cache semantics for block queries
- broad GraphQL parity claims
- new query execution paths

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase293-graphql-events-by-block-source-posture.md`

## Exit Criteria

- GraphQL `eventsByBlock` results can expose `querySourcePosture`.
- Focused GraphQL tests and package-level API tests remain green.
