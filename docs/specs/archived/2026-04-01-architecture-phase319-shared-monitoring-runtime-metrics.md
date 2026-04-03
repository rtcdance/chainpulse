# Phase 319 - Shared Monitoring Runtime Metrics

## Status
Status: Approved

## Summary

Extend shared protocol monitoring from raw counters to a compact runtime
metrics surface with coverage posture, runtime posture, and a reliability
hint.

## Problem

Shared monitoring already exposes raw request, success, failure, and latency
metrics, but callers still have to interpret those numbers manually to decide
whether a protocol path is healthy, degraded, or still unobserved.

## Decision

Add `GetRuntimeMetrics(protocol)` to `Monitoring` and expose:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no protocol instrumentation redesign
- no request pipeline contract change
- only shared runtime metrics surfacing

## Scope

In scope:

- shared monitoring runtime metrics
- compact coverage/runtime posture
- focused monitoring tests

Out of scope:

- instrumentation redesign
- per-request trace expansion
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestMonitoringRuntimeMetricsUnobserved|TestMonitoringRuntimeMetricsHealthy|TestMonitoringRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase319-shared-monitoring-runtime-metrics.md`

## Exit Criteria

- `Monitoring` exposes a compact runtime metrics surface beyond raw counters.
- Focused monitoring tests confirm unobserved, healthy, and degraded posture classification.
