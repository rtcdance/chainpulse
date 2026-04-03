# Phase 297 - GraphQL Schema Builder Source Parity

## Status
Status: Approved

## Summary

Restore GraphQL query-source-surfacing parity between the resolver path and the
schema-builder path for `eventsByName`.

## Problem

After the recent GraphQL source-surfacing expansion, the direct
`EventResolver.ResolveEventsByName(...)` path already attached
`querySourcePosture=graphql-event-store`, but the schema-builder
`resolveEventsByName(...)` path still returned event payloads without that
field.

That left a real protocol-surface inconsistency inside the same GraphQL event
family.

## Decision

Update the schema-builder `resolveEventsByName(...)` implementation to attach
the same compact source posture as the resolver path:

- `graphql-event-store`

Lock the parity with a focused schema-builder test.

## Scope

In scope:

- schema-builder parity for GraphQL `eventsByName`
- focused GraphQL test coverage for the schema-builder path
- package-level API regression coverage

Out of scope:

- broader GraphQL source taxonomy changes
- new GraphQL response envelope fields
- additional resolver expansion

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase297-graphql-schema-builder-source-parity.md`

## Exit Criteria

- GraphQL `eventsByName` exposes `querySourcePosture` consistently through both
  resolver and schema-builder paths.
- Focused GraphQL tests and package-level API tests remain green.
