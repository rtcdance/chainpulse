# Phase 372 - Shared TLS Legacy Metrics Baseline Assessment

## Status
Status: Approved

## Summary

Assess shared TLS legacy metrics after posture alignment and compatibility
preservation.

## Problem

After phase 371, `GetMetrics()` now exposes:

- `coverage_posture`
- `certificate_posture` (compatibility)
- `reload_posture`
- `reliability_hint`

while retaining existing TLS metric fields.

The repository needs an explicit statement on whether this aligned legacy TLS
surface is sufficient to treat TLS posture semantics as a stable baseline
stop-line.

## Decision

Classify the current shared TLS legacy metrics alignment as:

- `stage-complete for the shared TLS legacy metrics baseline`

This means:

- legacy TLS posture fields are aligned enough to pause by default
- compatibility with existing TLS metric consumers remains intact

It does **not** mean:

- TLS lifecycle strategy has been redesigned
- certificate policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared TLS legacy metrics baseline assessment
- explicit stop-line after legacy metrics posture alignment
- architecture/index documentation updates

Out of scope:

- certificate provisioning redesign
- TLS policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase372-shared-tls-legacy-metrics-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerMetricsIncludesPostureFields|TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue|TestTLSManagerRuntimeMetricsUnobserved|TestTLSManagerRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase372-shared-tls-legacy-metrics-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase372-shared-tls-legacy-metrics-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared TLS legacy metrics alignment as a stable
  baseline with a stop-line.
- Future legacy TLS posture field expansion is treated as explicit reopen.
