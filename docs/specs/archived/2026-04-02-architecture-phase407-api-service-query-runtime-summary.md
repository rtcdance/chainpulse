# Phase 407 - API Service Query Runtime Summary

## Status
Status: Approved

## Summary

Expose a read-only `/runtime/summary` route for `api-service` so the current
query-service-backed microservice runtime can present one compact runtime view
instead of forcing operators to infer query state from `/health` and `/metrics`
separately.

## Problem

`ARCHITECTURE_v1.md` describes the query service as a first-class runtime role
with health, caching, degradation, and consistency concerns.

The current `api-service` bootstrap already wires:

- `QueryService`
- health routes
- metrics
- query handlers

But it still lacks one compact runtime summary surface comparable to the
execution-oriented microservices. That leaves a practical gap:

- query runtime is wired
- query/runtime posture exists in rollout reasoning
- operators still do not get one read-only runtime summary route

## Decision

Add an optional runtime-summary route to the API gateway runtime route
composition and use it in `api-service` to expose:

- service/runtime posture
- query runtime status and message
- rollout posture derived from the existing api-service rollout helpers
- compact metrics summary

Keep the slice read-only:

- no writable control routes
- no request/response contract redesign for query endpoints
- no new query execution semantics

## Scope

In scope:

- runtime-summary route composition support in the gateway integration
- api-service runtime summary provider and response shape
- focused route and integration tests
- documentation refresh

Out of scope:

- query control plane
- new query endpoints
- distributed query orchestration redesign

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/api-service/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase407-api-service-query-runtime-summary.md`

## Exit Criteria

- `api-service` exposes a read-only `/runtime/summary` route through the
  existing runtime route composition.
- The summary includes query runtime health and compact runtime/metrics posture.
- Focused tests lock the route composition and summary payload.
