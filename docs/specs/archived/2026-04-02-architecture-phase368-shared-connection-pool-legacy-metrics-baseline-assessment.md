# Phase 368 - Shared Connection Pool Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared connection-pool legacy metrics after posture alignment and
compatibility preservation.

## Problem

After phase 367, `GetMetrics()` now exposes:

- `coverage_posture`
- `capacity_posture` (compatibility)
- `runtime_posture`
- `reliability_hint`

while retaining existing pool counter fields.

The repository needs an explicit statement on whether this aligned legacy pool
surface is sufficient to treat connection-pool posture semantics as a stable
baseline stop-line.

## Decision

Classify the current shared connection-pool legacy metrics alignment as:

- `stage-complete for the shared connection-pool legacy metrics baseline`

This means:

- legacy pool posture fields are aligned enough to pause by default
- compatibility with existing pool counter consumers remains intact

It does **not** mean:

- pool lifecycle strategy has been redesigned
- capacity policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared connection-pool legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- connection cleanup redesign
- factory policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase368-shared-connection-pool-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestConnectionPoolMetricsIncludesPostureFields|TestConnectionPoolRuntimeMetricsHealthy|TestConnectionPoolRuntimeMetricsUnobserved|TestConnectionPoolRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase368-shared-connection-pool-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase368-shared-connection-pool-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared connection-pool legacy metrics alignment
  as a stable baseline with a stop-line.
- Future legacy pool posture field expansion is treated as explicit reopen.
