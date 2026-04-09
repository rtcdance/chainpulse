Title: M1c Monolithic Runtime Route Inventory
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1c` already exposed monolithic `/runtime/summary`, `/runtime/control`, and `/metrics`,
but the gateway summary still mostly reported boolean feature flags. Operators could tell
that runtime routes existed, but not how much of the runtime surface was actually registered
in the gateway or whether the current route inventory had reached the minimum monolithic
observability baseline described by `ARCHITECTURE_v1.md`.

## Scope

This slice will:

1. Add a gateway runtime-route inventory snapshot derived from the registered router state.
2. Surface `registered_route_count`, `runtime_route_count`, and `runtime_surface_count`
   through the monolithic gateway summary.
3. Surface whether summary/control/metrics routes are currently enabled.
4. Add a compact `runtime_surface_posture` and `runtime_surface_hint` for the monolithic
   gateway runtime surface.
5. Add focused tests for the inventory and summary output.

## Non-Goals

This slice will not:

1. Add new runtime endpoints.
2. Change the semantics of `/health*`, `/runtime/summary`, `/runtime/control`, or `/metrics`.
3. Introduce writable control in monolithic mode.
4. Expand beyond the current monolithic gateway/operator baseline.

## Selected Approach

Reuse the initialized gateway router integration as the source of truth. Count registered
runtime routes directly from the router, derive a small route inventory snapshot, and expose
that inventory through the monolithic runtime summary. Keep the existing gateway posture
semantics intact and add a separate runtime-surface posture so `M1c` can tighten observability
without rewriting earlier milestone classifications.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-monolithic-runtime-route-inventory.md`

## Decision

Approved for implementation as the second `M1c` observability/API-gateway slice.

## Implementation Notes

Implemented in:

- `pkg/plugins/api/gateway.go`
- `pkg/plugins/api/gateway_router_integration.go`
- `pkg/plugins/api/gateway_router_integration_test.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The gateway now reports an actual runtime-route inventory from the router, and the monolithic
runtime summary gateway section now surfaces route counts plus a compact runtime-surface
posture/hint that tells operators whether the current runtime inventory has reached the
minimum `M1c` observability baseline.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-monolithic-runtime-route-inventory.md`
