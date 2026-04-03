# Phase 365 - Shared Request Batcher Legacy Metrics Posture Alignment

## Status
Status: Approved

## Summary

Align `RequestBatcher.GetMetrics()` with the compact posture-oriented runtime
surface while preserving existing batch counter compatibility.

## Problem

Shared request batcher already exposes posture fields through
`GetRuntimeMetrics()`, but legacy callers of `GetMetrics()` still receive only
raw batch counters. Those callers do not get the same compact posture and
reliability signals.

## Decision

Extend `GetMetrics()` to include:

- `coverage_posture`
- `capacity_posture` (compatibility)
- `runtime_posture`
- `reliability_hint`

Keep existing batch metrics fields unchanged:

- `batcher_name`
- `total_requests`
- `total_batches`
- `avg_batch_size`
- `errors`
- `avg_duration_ms`
- `total_duration`

Keep the change intentionally small:

- no batching algorithm redesign
- no processor contract change
- only legacy batch metrics alignment

## Scope

In scope:

- shared request-batcher legacy metrics posture alignment
- compatibility preservation for existing batch counter consumers
- focused batcher metrics tests

Out of scope:

- throughput policy redesign
- queue lifecycle changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestRequestBatcherMetricsIncludesPostureFields|TestRequestBatcherRuntimeMetricsHealthy|TestRequestBatcherRuntimeMetricsUnobserved|TestRequestBatcherRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase365-shared-request-batcher-legacy-metrics-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestRequestBatcherMetricsIncludesPostureFields|TestRequestBatcherRuntimeMetricsHealthy|TestRequestBatcherRuntimeMetricsUnobserved|TestRequestBatcherRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase365-shared-request-batcher-legacy-metrics-posture-alignment.md` passed.

## Exit Criteria

- `GetMetrics()` exposes posture/hint fields aligned with runtime metrics.
- Existing batch counter fields remain compatible.
- Focused tests confirm aligned posture values for healthy/unobserved/degraded states.
