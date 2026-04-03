# Phase 382 - Puller Metrics Runtime Surface

## Status
Status: Approved

## Summary

Expose the `puller` runtime `/metrics` route so the service-plane matches the
startup contract already advertised by `main.go`.

## Problem

`puller` currently logs:

- `Metrics available at: http://localhost:<port>/metrics`

but its runtime HTTP mux only serves `/health*` routes. This creates a
service-plane mismatch where operators are told a metrics surface exists even
though the route is not actually wired.

## Decision

Add a focused runtime `/metrics` handler for `puller` that exports the existing
`core.MetricsCollector` payload as JSON.

Keep the slice intentionally small:

- reuse the existing in-process metrics collector
- add no new transport or Prometheus contract
- preserve the current `/health*` runtime surface unchanged

## Scope

In scope:

- wire `/metrics` into the `puller` runtime HTTP mux
- pass the metrics collector through the runtime server helper
- add focused route coverage that verifies the route returns collected metrics
- document the new runtime-surface slice

Out of scope:

- Prometheus exposition redesign
- broader metrics format standardization
- changes to other services

## Validation

- focused `go test` for `cmd/microservices/puller`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase382-puller-metrics-runtime-surface.md`

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/puller/...` passed
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase382-puller-metrics-runtime-surface.md` passed
- `scripts/dev-micro-loop.sh --mode fast --base HEAD` remains blocked by the existing missing `gofumpt` prerequisite unless the environment changes

## Exit Criteria

- `puller` runtime HTTP surface serves `/metrics`.
- The route returns the current collector payload.
- The service-plane no longer advertises a metrics endpoint it does not expose.
