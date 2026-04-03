# Phase 307 - HTTP Runtime Metrics

## Status
Status: Approved

## Summary

Add a compact runtime metrics surface to the HTTP plugin so it can expose route
count, transport posture, runtime posture, and a reliability hint.

## Problem

The HTTP plugin already exposes raw facts such as running state and TLS
metrics, but it has no single compact runtime surface that describes whether it
is stopped, unrouted, or actively serving registered routes.

## Decision

Add `GetRuntimeMetrics()` to the HTTP plugin and expose:

- `running`
- `route_count`
- `transport_posture`
- `route_posture`
- `runtime_posture`
- `reliability_hint`

Also add `APIRouter.RouteCount()` so the plugin can report registered route
coverage without reaching into router internals.

## Scope

In scope:

- router route-count helper
- HTTP compact runtime metrics surface
- focused HTTP plugin tests

Out of scope:

- HTTP response envelope changes
- HTTP request-level source semantics
- broader HTTP control-plane redesign

## Validation

- `go test ./pkg/plugins/api/core -run 'TestAPIRouterRouteCount'`
- `go test ./pkg/plugins/api/http -run 'TestHTTPPluginGetRuntimeMetricsPlaintextStopped|TestHTTPPluginGetRuntimeMetricsTLSServing'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase307-http-runtime-metrics.md`

## Exit Criteria

- The HTTP plugin exposes a compact runtime metrics surface.
- The router can report registered route count through a stable helper.
