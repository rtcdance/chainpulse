# Phase 359 - Shared Health Summary Posture Alignment

## Status
Status: Approved

## Summary

Align `HealthCheck.GetHealthSummary()` with the common posture-oriented runtime
surface while preserving existing summary count compatibility.

## Problem

Shared health currently exposes compact posture fields through
`GetRuntimeMetrics()`, but `GetHealthSummary()` still returns only raw counts
and `overall_status`. Summary-first callers do not get the same compact
posture and reliability signals.

## Decision

Extend `GetHealthSummary()` to include:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep existing summary fields unchanged:

- `overall_status`
- `total_components`
- `healthy_count`
- `degraded_count`
- `unhealthy_count`

Keep the change intentionally small:

- no health-state transition redesign
- no component check contract change
- only summary surface alignment

## Scope

In scope:

- shared health summary posture field alignment
- compatibility preservation for existing summary-count consumers
- focused summary/runtime posture tests

Out of scope:

- health storage redesign
- cross-service health aggregation redesign
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestHealthCheckRuntimeSummaryHealthy|TestHealthCheckRuntimeSummaryDegraded|TestHealthCheckRuntimeSummaryUnhealthy|TestHealthCheckRuntimeSummaryUnobserved|TestHealthCheckHealthSummaryIncludesPostureFields'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase359-shared-health-summary-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestHealthCheckRuntimeMetricsUnconfigured|TestHealthCheckRuntimeMetricsPartial|TestHealthCheckRuntimeMetricsComplete|TestHealthCheckRuntimeSummaryHealthy|TestHealthCheckRuntimeSummaryDegraded|TestHealthCheckRuntimeSummaryUnhealthy|TestHealthCheckRuntimeSummaryUnobserved|TestHealthCheckHealthSummaryIncludesPostureFields'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase359-shared-health-summary-posture-alignment.md` passed.

## Exit Criteria

- `GetHealthSummary()` exposes posture/hint fields aligned with runtime metrics.
- Existing summary-count fields remain compatible.
- Focused tests confirm aligned posture values for healthy/degraded/unhealthy/unobserved states.
