# Phase 299 - GraphQL Schema Builder Cache Source Parity

## Status
Status: Approved

## Summary

Extend GraphQL schema-builder event reads to use the same cache-aware query
source surfacing already used by the resolver-based event paths.

## Problem

After phase 298, the resolver-based GraphQL event family could already expose:

- `graphql-cache-hit`
- `graphql-event-store`

for several cached event reads.

But the schema-builder path still had no cache dependency at all, which meant
the same event-family queries could only surface live event-store posture when
they were executed through the schema-builder implementation.

## Decision

Add a cache dependency to `SchemaBuilder` and use it to restore cache-aware
source parity for the schema-builder event family where the resolver path
already supports cache-aware surfacing:

- `event`
- `events`
- `eventsByAddress`
- `eventsByName`

Keep `eventsByBlock` unchanged because the resolver path still treats it as a
live-only read.

## Scope

In scope:

- `SchemaBuilder` cache dependency
- cache-aware source surfacing for schema-builder event reads
- focused schema-builder cache-hit coverage
- package-level API regression coverage

Out of scope:

- broader GraphQL cache invalidation redesign
- new GraphQL source categories
- cache support for `eventsByBlock`

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase299-graphql-schema-builder-cache-source-parity.md`

## Exit Criteria

- Schema-builder event reads can surface `graphql-cache-hit` where the resolver
  path already does.
- Resolver and schema-builder event-family source contracts remain aligned.
- Focused GraphQL tests and package-level API tests remain green.
