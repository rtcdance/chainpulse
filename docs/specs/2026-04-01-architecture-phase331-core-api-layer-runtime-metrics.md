# Phase 331 - Core API Layer Runtime Metrics

## Status
Status: Approved

## Summary

Extend the core API layer from raw router wiring to a compact runtime metrics
surface with coverage posture, runtime posture, and a reliability hint.

## Problem

The core API layer already composes router state and error mapping, but callers
still have to inspect route counts, middleware, and error-mapper presence
manually to decide whether the layer is empty, merely routed, or hardened.

## Decision

Add `GetRuntimeMetrics()` to `APILayer` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no routing redesign
- no error-mapper contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core API-layer runtime metrics
- compact route/error-mapper/middleware posture
- focused API-layer tests

Out of scope:

- routing redesign
- error-mapper redesign
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestAPILayerRuntimeMetricsUnobserved|TestAPILayerRuntimeMetricsWatch|TestAPILayerRuntimeMetricsHardened'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase331-core-api-layer-runtime-metrics.md`

## Exit Criteria

- `APILayer` exposes a compact runtime metrics surface beyond raw router wiring.
- Focused API-layer tests confirm unobserved, watch, and hardened posture classification.
