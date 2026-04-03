# Phase 329 - Core API Router Runtime Metrics

## Status
Status: Approved

## Summary

Extend the core API router from raw route counting to a compact runtime metrics
surface with coverage posture, runtime posture, and a reliability hint.

## Problem

The core API router already exposes route count and implicitly holds middleware
coverage, but callers still have to inspect routes and middleware separately to
decide whether the router is empty, lightly guarded, or ready.

## Decision

Add `GetRuntimeMetrics()` to `APIRouter` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no routing redesign
- no handler contract change
- only core runtime metrics surfacing

## Scope

In scope:

- core router runtime metrics
- compact route/middleware posture
- focused router tests

Out of scope:

- routing redesign
- per-route diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/core -run 'TestAPIRouterRuntimeMetricsUnobserved|TestAPIRouterRuntimeMetricsWatch|TestAPIRouterRuntimeMetricsReady'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase329-core-api-router-runtime-metrics.md`

## Exit Criteria

- `APIRouter` exposes a compact runtime metrics surface beyond raw route counts.
- Focused router tests confirm unobserved, watch, and ready posture classification.
