# Phase 313 - Shared Connection Pool Runtime Metrics

## Status
Status: Approved

## Summary

Extend the shared connection pool from raw counters to a compact runtime
metrics surface with capacity posture, runtime posture, and a reliability hint.

## Problem

The shared connection pool already exposes raw creation/reuse/error counters,
but callers still have to interpret those numbers manually to decide whether
the pool is healthy, pressured, degraded, or still unobserved.

## Decision

Add `GetRuntimeMetrics()` to `ConnectionPool` and expose:

- `capacity_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no acquire/release contract change
- no cleanup redesign
- only shared runtime metrics surfacing

## Scope

In scope:

- shared connection pool runtime metrics
- compact capacity/runtime posture
- focused connection pool tests

Out of scope:

- pool scheduling redesign
- per-connection diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestConnectionPoolRuntimeMetricsHealthy|TestConnectionPoolRuntimeMetricsUnobserved|TestConnectionPoolRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase313-shared-connection-pool-runtime-metrics.md`

## Exit Criteria

- `ConnectionPool` exposes a compact runtime metrics surface beyond raw counters.
- Focused pool tests confirm healthy, unobserved, and degraded posture classification.
