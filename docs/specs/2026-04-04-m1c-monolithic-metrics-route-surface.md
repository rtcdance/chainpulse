Title: M1c Monolithic Metrics Route Surface
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1a` and `M1b` established the monolithic runtime and resilience baselines,
but `M1c` needs the monolithic API gateway surface to expose the minimum
operator-facing observability contract described by `ARCHITECTURE_v1.md`.
The monolithic startup output already advertises `/metrics`, but the gateway
runtime-route composition has not yet turned that into an explicit, test-backed
runtime contract alongside `/health*`, `/runtime/summary`, and `/runtime/control`.

## Scope

This slice will:

1. Add an optional gateway runtime metrics provider.
2. Expose `GET /metrics` through gateway runtime-route composition when that
   provider is configured.
3. Wire monolithic mode to export its existing metrics collector through that
   route.
4. Surface `metrics_route_enabled` in monolithic runtime summary.
5. Add focused route and summary tests.

## Non-Goals

This slice will not:

1. Change metric schemas.
2. Introduce Prometheus text exposition.
3. Add alerting, dashboards, or tracing.
4. Expand protocol coverage beyond the existing monolithic gateway surface.

## Selected Approach

Reuse the existing runtime-route composition model already used for health,
runtime summary, and runtime control. Add one more optional provider for
runtime metrics, return the existing metrics collector export as JSON, and make
the gateway summary explicitly state whether the metrics route is enabled.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-monolithic-metrics-route-surface.md`

## Decision

Approved for implementation as the first `M1c` observability/API-gateway slice.

## Implementation Notes

Implemented in:

- `pkg/plugins/api/gateway.go`
- `pkg/plugins/api/gateway_router_integration.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/runtime_metrics.go`
- `cmd/monolithic/chainpulse/runtime_metrics_test.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The gateway runtime-route composition now supports an optional runtime metrics
provider and registers `GET /metrics` when that provider is configured.

Monolithic mode now exports its existing metrics collector through that runtime
route, and the monolithic runtime summary gateway section now surfaces
`metrics_route_enabled` alongside the existing runtime route facts.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/plugins/api/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1c-monolithic-metrics-route-surface.md`
