# Phase 384 - Event Processor Runtime Summary Surface

## Status
Status: Approved

## Summary

Expose a read-only `event-processor` `/runtime/summary` route so operators can
read rollout posture, component posture, and metrics summary from one compact
runtime surface.

## Problem

`event-processor` now exposes:

- `/health*`
- `/metrics`

but operators still have to stitch together rollout posture and metrics shape
from multiple endpoints. The next higher-value slice is a compact read-only
runtime summary surface rather than jumping straight into mutable control
actions.

## Decision

Add a focused `/runtime/summary` route for `event-processor` that returns:

- service identity
- runtime mode and posture
- component state
- rollout summary details
- compact metrics summary

Keep the slice intentionally small:

- read-only only
- no new control actions
- no metrics transport redesign
- no broader multi-service abstraction yet

## Scope

In scope:

- add `/runtime/summary` to the `event-processor` runtime mux
- build a compact response from existing rollout state and current metrics
- add focused route coverage
- document the new operator-facing runtime surface

Out of scope:

- write/control endpoints
- Prometheus exposition redesign
- `puller` parity work in the same phase

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase384-event-processor-runtime-summary-surface.md`

## Exit Criteria

- `event-processor` exposes `/runtime/summary`.
- The route returns compact rollout and metrics summary data from the current
  runtime state.
- Operators no longer need to manually combine `/health/rollout` and `/metrics`
  just to read a first-pass runtime summary.
