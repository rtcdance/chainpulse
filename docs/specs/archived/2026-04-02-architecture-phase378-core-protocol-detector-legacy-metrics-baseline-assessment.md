# Phase 378 - Core Protocol Detector Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess core protocol-detector legacy metrics after posture alignment and
compatibility preservation.

## Problem

After phase 377, `GetMetrics()` now exposes:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

while retaining existing protocol metrics fields.

The repository needs an explicit statement on whether this aligned legacy
protocol-detector surface is sufficient to treat detector posture semantics as
a stable baseline stop-line.

## Decision

Classify the current core protocol-detector legacy metrics alignment as:

- `stage-complete for the core protocol-detector legacy metrics baseline`

This means:

- legacy detector posture fields are aligned enough to pause by default
- compatibility with existing protocol metrics consumers remains intact

It does **not** mean:

- protocol-detection strategy has been redesigned
- routing policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- core protocol-detector legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- protocol-detection redesign
- routing policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase378-core-protocol-detector-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestProtocolDetectorMetricsIncludesPostureFields|TestProtocolDetectorRuntimeMetricsUnobserved|TestProtocolDetectorRuntimeMetricsWatch|TestProtocolDetectorRuntimeMetricsReady'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase378-core-protocol-detector-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.

## Exit Criteria

- The docs explicitly describe core protocol-detector legacy metrics alignment
  as a stable baseline with a stop-line.
- Future legacy detector posture field expansion is treated as explicit reopen.
