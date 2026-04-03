# Phase 315 - Shared Request Batcher Runtime Metrics

## Status
Status: Approved

## Summary

Extend the shared request batcher from raw counters to a compact runtime
metrics surface with capacity posture, runtime posture, and a reliability hint.

## Problem

The shared request batcher already exposes raw request/batch/error counters, but
callers still have to interpret those numbers manually to determine whether the
batcher is healthy, degraded, or still unobserved, and whether current batches
are lightly filled or well utilized.

## Decision

Add `GetRuntimeMetrics()` to `RequestBatcher` and expose:

- `capacity_posture`
- `runtime_posture`
- `reliability_hint`

Keep the change intentionally small:

- no submit/process contract change
- no batching loop redesign
- only shared runtime metrics surfacing

## Scope

In scope:

- shared request-batcher runtime metrics
- compact capacity/runtime posture
- focused batcher tests

Out of scope:

- processor redesign
- per-request diagnostics
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestRequestBatcherRuntimeMetricsHealthy|TestRequestBatcherRuntimeMetricsUnobserved|TestRequestBatcherRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase315-shared-request-batcher-runtime-metrics.md`

## Exit Criteria

- `RequestBatcher` exposes a compact runtime metrics surface beyond raw counters.
- Focused batcher tests confirm healthy, unobserved, and degraded posture classification.
