# Phase 374 - Shared Error Handler Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared error-handler legacy metrics after posture alignment and
compatibility preservation.

## Problem

After phase 373, `GetMetrics()` now exposes:

- `coverage_posture`
- `circuit_posture` (compatibility)
- `retry_posture`
- `reliability_hint`

while retaining existing error-handler metric fields.

The repository needs an explicit statement on whether this aligned legacy
error-handler surface is sufficient to treat error-handler posture semantics as
a stable baseline stop-line.

## Decision

Classify the current shared error-handler legacy metrics alignment as:

- `stage-complete for the shared error-handler legacy metrics baseline`

This means:

- legacy error-handler posture fields are aligned enough to pause by default
- compatibility with existing error-handler metric consumers remains intact

It does **not** mean:

- circuit-breaker strategy has been redesigned
- retry policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared error-handler legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- breaker redesign
- retry policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase374-shared-error-handler-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerMetricsIncludesPostureFields|TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase374-shared-error-handler-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase374-shared-error-handler-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared error-handler legacy metrics alignment as
  a stable baseline with a stop-line.
- Future legacy error-handler posture field expansion is treated as explicit reopen.
