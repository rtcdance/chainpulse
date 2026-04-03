# Phase 350 - Shared Monitoring Runtime Surface Baseline Assessment

## Status
Status: Approved

## Summary

Assess the shared monitoring runtime surface after alignment to aggregate
`GetRuntimeMetrics()` with protocol-scoped compatibility.

## Problem

After phase 349, shared monitoring now provides:

- aggregate runtime metrics via no-arg `GetRuntimeMetrics()`
- protocol-scoped runtime metrics via `GetProtocolRuntimeMetrics(protocol)`
- compact coverage/runtime posture and reliability hint semantics

The repository needs an explicit baseline stop-line decision for this aligned
runtime surface.

## Decision

Classify the current shared monitoring runtime surface as:

- `stage-complete for the shared monitoring runtime surface baseline`

This means:

- aligned aggregate runtime surfacing is strong enough to pause by default
- protocol-scoped runtime visibility remains explicitly available

It does **not** mean:

- monitoring aggregation strategy has been redesigned
- alerting/slo semantics are finalized
- broader control-plane orchestration is complete

## Scope

In scope:

- shared monitoring runtime surface baseline assessment
- explicit stop-line after runtime API alignment
- architecture/index documentation updates

Out of scope:

- monitoring storage redesign
- cross-service telemetry federation redesign
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase350-shared-monitoring-runtime-surface-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestMonitoringAggregateRuntimeMetricsUnobserved|TestMonitoringAggregateRuntimeMetricsHealthy|TestMonitoringAggregateRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase350-shared-monitoring-runtime-surface-baseline-assessment.md` should pass while the spec remains in `Approved` state.

## Exit Criteria

- The docs explicitly describe shared monitoring runtime surfacing as a stable
  baseline with a stop-line.
- Future shared monitoring runtime surface expansion is treated as explicit reopen.
