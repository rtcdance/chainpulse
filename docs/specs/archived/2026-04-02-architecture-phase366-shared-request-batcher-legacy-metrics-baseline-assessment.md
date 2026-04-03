# Phase 366 - Shared Request Batcher Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared request-batcher legacy metrics after posture alignment and
compatibility preservation.

## Problem

After phase 365, `GetMetrics()` now exposes:

- `coverage_posture`
- `capacity_posture` (compatibility)
- `runtime_posture`
- `reliability_hint`

while retaining existing batch counter fields.

The repository needs an explicit statement on whether this aligned legacy batch
surface is sufficient to treat request-batcher posture semantics as a stable
baseline stop-line.

## Decision

Classify the current shared request-batcher legacy metrics alignment as:

- `stage-complete for the shared request-batcher legacy metrics baseline`

This means:

- legacy batcher posture fields are aligned enough to pause by default
- compatibility with existing batch counter consumers remains intact

It does **not** mean:

- batching strategy has been redesigned
- throughput policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared request-batcher legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- queue lifecycle redesign
- processor policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase366-shared-request-batcher-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestRequestBatcherMetricsIncludesPostureFields|TestRequestBatcherRuntimeMetricsHealthy|TestRequestBatcherRuntimeMetricsUnobserved|TestRequestBatcherRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase366-shared-request-batcher-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase366-shared-request-batcher-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared request-batcher legacy metrics alignment
  as a stable baseline with a stop-line.
- Future legacy batcher posture field expansion is treated as explicit reopen.
