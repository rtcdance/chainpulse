# Phase 364 - Shared Monitoring Legacy Protocol Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared monitoring legacy protocol metrics after posture alignment and
compatibility preservation.

## Problem

After phase 363, `GetMetrics(protocol)` now exposes:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

while retaining existing protocol counter fields.

The repository needs an explicit statement on whether this aligned legacy
protocol surface is sufficient to treat monitoring posture semantics as a
stable baseline stop-line.

## Decision

Classify the current shared monitoring legacy protocol metrics alignment as:

- `stage-complete for the shared monitoring legacy protocol metrics baseline`

This means:

- legacy monitoring posture fields are aligned enough to pause by default
- compatibility with existing protocol counter consumers remains intact

It does **not** mean:

- monitoring strategy has been redesigned
- alerting policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared monitoring legacy protocol metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- metrics storage redesign
- alerting policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase364-shared-monitoring-legacy-protocol-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestMonitoringMetricsIncludesPostureFields|TestMonitoringRuntimeMetricsUnobserved|TestMonitoringRuntimeMetricsHealthy|TestMonitoringRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase364-shared-monitoring-legacy-protocol-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase364-shared-monitoring-legacy-protocol-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared monitoring legacy protocol metrics
  alignment as a stable baseline with a stop-line.
- Future legacy monitoring posture field expansion is treated as explicit reopen.
