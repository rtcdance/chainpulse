# Phase 376 - Shared Response Compressor Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared response-compressor legacy metrics after posture alignment and
compatibility preservation.

## Problem

After phase 375, `GetMetrics()` now exposes:

- `coverage_posture`
- `efficiency_posture`
- `reliability_hint`

while retaining existing compression metric fields.

The repository needs an explicit statement on whether this aligned legacy
compression surface is sufficient to treat response-compressor posture
semantics as a stable baseline stop-line.

## Decision

Classify the current shared response-compressor legacy metrics alignment as:

- `stage-complete for the shared response-compressor legacy metrics baseline`

This means:

- legacy compression posture fields are aligned enough to pause by default
- compatibility with existing compression metric consumers remains intact

It does **not** mean:

- compression strategy has been redesigned
- transport policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared response-compressor legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- compression-threshold redesign
- transport policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase376-shared-response-compressor-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestResponseCompressorMetricsIncludesPostureFields|TestResponseCompressorRuntimeMetricsUnobserved|TestResponseCompressorRuntimeMetricsBypassed|TestResponseCompressorRuntimeMetricsEfficient'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase376-shared-response-compressor-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase376-shared-response-compressor-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared response-compressor legacy metrics
  alignment as a stable baseline with a stop-line.
- Future legacy compression posture field expansion is treated as explicit reopen.
