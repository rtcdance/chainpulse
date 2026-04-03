# Phase 360 - Shared Health Summary Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared health summary surfacing after posture alignment and compatibility
preservation.

## Problem

After phase 359, `GetHealthSummary()` now exposes:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

while retaining existing summary count fields.

The repository needs an explicit statement on whether this aligned health
summary surface is sufficient to treat summary posture semantics as a stable
baseline stop-line.

## Decision

Classify the current shared health summary alignment as:

- `stage-complete for the shared health summary baseline`

This means:

- summary-level posture fields are aligned enough to pause by default
- compatibility with existing summary count consumers remains intact

It does **not** mean:

- health check strategy has been redesigned
- cross-service health policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared health summary baseline assessment
- explicit stop-line after summary posture alignment
- architecture/index documentation updates

Out of scope:

- health model redesign
- scheduler/probe policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase360-shared-health-summary-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestHealthCheckRuntimeSummaryHealthy|TestHealthCheckRuntimeSummaryDegraded|TestHealthCheckRuntimeSummaryUnhealthy|TestHealthCheckRuntimeSummaryUnobserved|TestHealthCheckHealthSummaryIncludesPostureFields'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase360-shared-health-summary-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase360-shared-health-summary-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared health summary alignment as a stable
  baseline with a stop-line.
- Future summary posture field expansion is treated as explicit reopen.
