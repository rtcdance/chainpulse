# Phase 323 - Shared Authentication Runtime Metrics

## Status
Status: Approved

## Summary

Extend shared authentication from raw token counts to a compact runtime metrics
surface with coverage posture, runtime posture, and a reliability hint.

## Problem

Shared authentication already exposes raw token counts, active token counts,
and expired token counts, but callers still have to interpret those numbers
manually to decide whether authentication is ready, aging, or degraded.

## Decision

Add `GetRuntimeMetrics()` to `Authentication` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no token-validation redesign
- no permission contract change
- only shared runtime metrics surfacing

## Scope

In scope:

- shared authentication runtime metrics
- compact token coverage/runtime posture
- focused authentication tests

Out of scope:

- token rotation redesign
- per-token diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestAuthenticationRuntimeMetricsUnconfigured|TestAuthenticationRuntimeMetricsReady|TestAuthenticationRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase323-shared-auth-runtime-metrics.md`

## Exit Criteria

- `Authentication` exposes a compact runtime metrics surface beyond raw token counts.
- Focused authentication tests confirm unconfigured, ready, and degraded posture classification.
