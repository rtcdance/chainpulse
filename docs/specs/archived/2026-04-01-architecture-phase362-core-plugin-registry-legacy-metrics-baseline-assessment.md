# Phase 362 - Core Plugin Registry Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess core plugin-registry legacy metrics after posture alignment and
compatibility preservation.

## Problem

After phase 361, `GetRegistryMetrics()` now exposes:

- `coverage_posture`
- `runtime_posture`
- `reliability_hint`

while retaining existing registry counter fields.

The repository needs an explicit statement on whether this aligned legacy
registry surface is sufficient to treat plugin-registry posture semantics as a
stable baseline stop-line.

## Decision

Classify the current core plugin-registry legacy metrics alignment as:

- `stage-complete for the core plugin-registry legacy metrics baseline`

This means:

- legacy registry posture fields are aligned enough to pause by default
- compatibility with existing registry counter consumers remains intact

It does **not** mean:

- plugin lifecycle strategy has been redesigned
- plugin rollout policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- core plugin-registry legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- lifecycle orchestration redesign
- plugin health policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase362-core-plugin-registry-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/core -run 'TestPluginRegistryMetricsIncludesPostureFields|TestPluginRegistryRuntimeMetricsUnobserved|TestPluginRegistryRuntimeMetricsReady|TestPluginRegistryRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase362-core-plugin-registry-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase362-core-plugin-registry-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe core plugin-registry legacy metrics alignment as
  a stable baseline with a stop-line.
- Future legacy registry posture field expansion is treated as explicit reopen.
