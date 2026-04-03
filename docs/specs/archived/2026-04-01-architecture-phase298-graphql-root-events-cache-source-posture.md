# Phase 298 - GraphQL Root Events Cache Source Posture

## Status
Status: Approved

## Summary

Extend GraphQL query-source surfacing on the root `events` connection to cover
cache-hit responses as well as live event-store responses.

## Problem

After phase 295, the GraphQL root `events` connection exposed
`querySourcePosture=graphql-event-store` for live reads, but cache-served root
list responses still had no way to surface that they came from cache.

That left the most generic GraphQL event list path behind the source semantics
already used by other cached GraphQL event reads.

## Decision

Add cache-aware source surfacing to `EventResolver.ResolveEvents(...)`:

- cache-hit responses surface `graphql-cache-hit`
- live responses continue to surface `graphql-event-store`

Use a small helper that can rewrite source posture on connection node payloads
without redesigning the GraphQL connection envelope.

## Scope

In scope:

- root `events` cache-hit source surfacing
- connection helper for updating event node source posture
- focused GraphQL cache-hit coverage
- package-level API regression coverage

Out of scope:

- schema-builder cache support
- broader GraphQL cache taxonomy changes
- new GraphQL response meta fields

## Validation

- `go test ./pkg/plugins/api/graphql/...`
- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase298-graphql-root-events-cache-source-posture.md`

## Exit Criteria

- GraphQL root `events` cache-hit responses expose `querySourcePosture`.
- Live root `events` responses continue exposing `graphql-event-store`.
- Focused GraphQL tests and package-level API tests remain green.
