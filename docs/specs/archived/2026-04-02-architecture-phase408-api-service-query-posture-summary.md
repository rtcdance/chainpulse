# Phase 408 - API Service Query Posture Summary

## Status
Status: Approved

## Summary

Strengthen the `api-service` `/runtime/summary` query section so it surfaces a
compact query-runtime posture that better matches the `ARCHITECTURE_v1.md`
query-service blueprint.

## Problem

Phase 407 added a read-only `/runtime/summary` route for `api-service`, but the
query section still mainly exposed:

- query health status
- query health message
- a generic query health hint

That is helpful, but still lighter than the query-service blueprint in
`ARCHITECTURE_v1.md`, which explicitly emphasizes:

- cache posture
- circuit-breaker posture
- consistency posture
- graceful degradation semantics

## Decision

Add a compact runtime summary contract in `pkg/services/query` and let
`api-service` consume it when available.

The first compact query-runtime posture layer should expose:

- query posture
- cache posture
- circuit-breaker posture
- consistency posture
- reliability hint

Keep the implementation honest:

- use real query-service health/cache facts where available
- mark circuit-breaker and consistency as `not-wired` rather than pretending
  deeper runtime wiring already exists

## Scope

In scope:

- query runtime summary contract in `pkg/services/query`
- api-service runtime summary integration for the new posture fields
- focused query and api-service tests
- documentation refresh

Out of scope:

- circuit-breaker execution redesign
- consistency-checker background orchestration
- new query endpoints or control routes

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/services/query ./pkg/plugins/api ./cmd/microservices/api-service/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase408-api-service-query-posture-summary.md`

## Exit Criteria

- `QueryService` can expose a compact runtime summary contract.
- `api-service` `/runtime/summary` includes query posture fields that better
  match the v1 query-service blueprint.
- Focused tests lock both the query summary contract and the api-service
  integration.
