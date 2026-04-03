# Phase 347 - Shared Health Runtime Metrics Alignment

## Status
Status: Implemented

## Summary

Align shared health runtime surfacing with the common `GetRuntimeMetrics()`
shape while preserving compatibility with existing `GetRuntimeSummary()`
callers.

## Problem

Shared health already exposes runtime posture and reliability hint through
`GetRuntimeSummary()`, but this slice is inconsistent with the broader runtime
metrics contract that uses `GetRuntimeMetrics()` and includes explicit
`coverage_posture`.

## Decision

Add `GetRuntimeMetrics()` to `HealthCheck` with:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep `GetRuntimeSummary()` as a compatibility wrapper.

Keep the change intentionally small:

- no health-state transition redesign
- no component check contract change
- only runtime metrics interface alignment

## Scope

In scope:

- shared health runtime metrics interface alignment
- explicit health coverage posture classification
- focused health runtime metrics tests

Out of scope:

- health storage redesign
- cross-service health aggregation redesign
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestHealthCheckRuntimeMetricsUnconfigured|TestHealthCheckRuntimeMetricsPartial|TestHealthCheckRuntimeMetricsComplete|TestHealthCheckRuntimeSummaryHealthy|TestHealthCheckRuntimeSummaryDegraded|TestHealthCheckRuntimeSummaryUnhealthy|TestHealthCheckRuntimeSummaryUnobserved'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase347-shared-health-runtime-metrics-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestHealthCheckRuntimeMetricsUnconfigured|TestHealthCheckRuntimeMetricsPartial|TestHealthCheckRuntimeMetricsComplete|TestHealthCheckRuntimeSummaryHealthy|TestHealthCheckRuntimeSummaryDegraded|TestHealthCheckRuntimeSummaryUnhealthy|TestHealthCheckRuntimeSummaryUnobserved'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase347-shared-health-runtime-metrics-alignment.md` passed.

## Exit Criteria

- `HealthCheck` exposes `GetRuntimeMetrics()` with compact health coverage/runtime semantics.
- Existing `GetRuntimeSummary()` callers remain compatible.
- Focused tests confirm unconfigured, partial, and complete coverage posture.
