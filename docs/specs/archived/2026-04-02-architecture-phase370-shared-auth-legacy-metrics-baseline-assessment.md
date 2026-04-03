# Phase 370 - Shared Auth Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared auth legacy metrics after posture alignment and compatibility
preservation.

## Problem

After phase 369, `GetMetrics()` now exposes:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

while retaining existing token count fields.

The repository needs an explicit statement on whether this aligned legacy auth
surface is sufficient to treat authentication posture semantics as a stable
baseline stop-line.

## Decision

Classify the current shared auth legacy metrics alignment as:

- `stage-complete for the shared auth legacy metrics baseline`

This means:

- legacy auth posture fields are aligned enough to pause by default
- compatibility with existing token count consumers remains intact

It does **not** mean:

- token lifecycle strategy has been redesigned
- permission policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared auth legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- token rotation redesign
- auth policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase370-shared-auth-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestAuthenticationMetricsIncludesPostureFields|TestAuthenticationRuntimeMetricsUnconfigured|TestAuthenticationRuntimeMetricsReady|TestAuthenticationRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase370-shared-auth-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase370-shared-auth-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared auth legacy metrics alignment as a stable
  baseline with a stop-line.
- Future legacy auth posture field expansion is treated as explicit reopen.
