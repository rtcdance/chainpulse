# Phase 387 - Puller Runtime Control Plane

## Status
Status: Approved

## Summary

Reopen execution-service control-plane work with a minimal writable runtime
control surface for `puller` that can pause and resume the polling loop.

## Problem

`puller` now has a strong read-only operator surface:

- `/health*`
- `/metrics`
- `/runtime/summary`

but it still has no minimal writable control action. The next meaningful step
is a real, low-risk action surface that uses the existing polling loop
lifecycle instead of inventing a broader mutable control system.

## Decision

Add a focused puller runtime control plane with:

- `GET /runtime/control`
- `POST /runtime/control/pause`
- `POST /runtime/control/resume`

Back the routes with a concurrency-safe in-memory loop controller that causes
the existing polling loop to skip poll ticks while paused.

Keep the slice intentionally small:

- control only the in-process polling loop
- no distributed coordination
- no persistence across restarts
- no new control plane for other services in the same phase

## Scope

In scope:

- add a puller loop controller with pause/resume state
- make the polling loop honor paused state
- expose runtime control routes for current state, pause, and resume
- add focused HTTP and loop-control tests
- document the new control-plane slice

Out of scope:

- durable control state
- multi-instance coordination
- broader execution-service control parity

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/puller/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase387-puller-runtime-control-plane.md`

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/puller/...` passed
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/puller/...` passed
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase387-puller-runtime-control-plane.md` passed
- `scripts/dev-micro-loop.sh --mode fast --base HEAD` remains blocked by the existing missing `gofumpt` prerequisite unless the environment changes

## Exit Criteria

- `puller` exposes `GET /runtime/control`.
- `puller` exposes `POST /runtime/control/pause` and `POST /runtime/control/resume`.
- A paused control state causes the polling loop to skip poll execution.
