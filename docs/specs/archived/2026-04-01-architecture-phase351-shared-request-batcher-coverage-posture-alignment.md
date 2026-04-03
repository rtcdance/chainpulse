# Phase 351 - Shared Request Batcher Coverage Posture Alignment

## Status
Status: Implemented

## Summary

Align shared request-batcher runtime metrics to expose explicit
`coverage_posture` while preserving existing `capacity_posture` compatibility.

## Problem

`RequestBatcher.GetRuntimeMetrics()` currently uses `capacity_posture` as the
primary coverage signal, which diverges from the broader runtime contract that
exposes `coverage_posture` across components.

## Decision

Extend `GetRuntimeMetrics()` to include:

- `coverage_posture` (aligned with existing batch-capacity posture semantics)
- existing `capacity_posture` retained for backward compatibility

Keep the change intentionally small:

- no batching algorithm redesign
- no processor contract change
- only runtime metrics field alignment

## Scope

In scope:

- request-batcher runtime coverage field alignment
- compatibility preservation for existing capacity posture consumers
- focused runtime metrics tests

Out of scope:

- batch processing strategy redesign
- throughput policy changes
- broader control-plane semantics

## Validation

- `go test ./pkg/plugins/api/shared -run 'TestRequestBatcherRuntimeMetricsHealthy|TestRequestBatcherRuntimeMetricsUnobserved|TestRequestBatcherRuntimeMetricsDegraded'`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase351-shared-request-batcher-coverage-posture-alignment.md`

## Verification Summary

- `go test ./pkg/plugins/api/shared -run 'TestRequestBatcherRuntimeMetricsHealthy|TestRequestBatcherRuntimeMetricsUnobserved|TestRequestBatcherRuntimeMetricsDegraded'` passed with `GOCACHE=/tmp/chainpulse-go-build-cache` under the local Go 1.24 toolchain.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase351-shared-request-batcher-coverage-posture-alignment.md` passed.

## Exit Criteria

- `RequestBatcher` runtime metrics expose `coverage_posture` and retain
  `capacity_posture` compatibility.
- Focused tests confirm posture values for healthy/unobserved/degraded states.
