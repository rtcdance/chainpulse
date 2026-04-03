# Phase 309 - Shared TLS Runtime Metrics

## Status
Status: Approved

## Summary

Extend the shared TLS manager from raw reload/error counters to a compact
runtime metrics surface with certificate posture, reload posture, and a
reliability hint.

## Problem

The shared TLS manager already exposes raw reload/error metrics, but callers
still have to interpret those counters manually to determine whether TLS is
ready, degraded, or due for reload.

## Decision

Add `GetRuntimeMetrics()` to `TLSManager` and expose:

- `enabled`
- `reload_ttl`
- `certificate_posture`
- `reload_posture`
- `reliability_hint`

Keep the change intentionally small:

- no certificate file watching redesign
- no plugin-specific coupling
- only shared runtime metrics-level surfacing

## Scope

In scope:

- shared TLS runtime metrics surface
- compact certificate/reload posture
- focused TLS manager tests

Out of scope:

- plugin-specific runtime wiring
- TLS auto-reload orchestration redesign
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase309-shared-tls-runtime-metrics.md`

## Exit Criteria

- `TLSManager` exposes a compact runtime metrics surface beyond raw counters.
- Focused TLS manager tests confirm ready and reload-due posture classification.
