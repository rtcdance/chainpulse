# Phase 363 - Shared Monitoring Legacy Protocol Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `Monitoring.GetMetrics(protocol)` with the compact posture-oriented
runtime surface while preserving existing protocol counter compatibility.

## Problem

Shared monitoring already exposes posture fields through
`GetProtocolRuntimeMetrics(protocol)`, but legacy callers of
`GetMetrics(protocol)` still receive only raw protocol counters. Those callers
do not get the same compact posture and reliability signals.

## Decision

Extend `GetMetrics(protocol)` to include:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep existing protocol metrics fields unchanged:

- `protocol`
- `total_requests`
- `successful_requests`
- `failed_requests`
- `success_rate`
- `error_rate`
- `avg_duration_ms`
- `last_request_time`

Keep the change intentionally small:

- no monitoring storage redesign
- no request recording contract change
- only legacy protocol metrics alignment

## Scope

In scope:

- shared monitoring legacy protocol metrics posture alignment
- compatibility preservation for existing protocol counter consumers
- focused monitoring metrics tests

Out of scope:

- alerting policy redesign
- monitoring aggregation redesign
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestMonitoringMetricsIncludesPostureFields|TestMonitoringRuntimeMetricsUnobserved|TestMonitoringRuntimeMetricsHealthy|TestMonitoringRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase363-shared-monitoring-legacy-protocol-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestMonitoringMetricsIncludesPostureFields|TestMonitoringRuntimeMetricsUnobserved|TestMonitoringRuntimeMetricsHealthy|TestMonitoringRuntimeMetricsDegraded|TestMonitoringAggregateRuntimeMetricsUnobserved|TestMonitoringAggregateRuntimeMetricsHealthy|TestMonitoringAggregateRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase363-shared-monitoring-legacy-protocol-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics(protocol)` exposes posture/hint fields aligned with protocol runtime metrics.
- Existing protocol counter fields remain compatible.
- Focused tests confirm aligned posture values for unobserved, healthy, and degraded states.
