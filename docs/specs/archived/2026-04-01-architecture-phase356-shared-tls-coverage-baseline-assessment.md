# Phase 356 - Shared TLS Coverage Baseline Assessment

## Status
Status: Approved

## Summary

Assess the shared TLS runtime surface after coverage posture alignment and
compatibility preservation.

## Problem

After phase 355, shared TLS runtime metrics now expose:

- `coverage_posture`
- `certificate_posture` (compatibility)
- `reload_posture`
- `reliability_hint`

The repository needs an explicit statement on whether this aligned runtime
surface is sufficient to treat TLS posture semantics as a stable baseline
stop-line.

## Decision

Classify the current shared TLS coverage alignment as:

- `stage-complete for the shared TLS coverage baseline`

This means:

- TLS runtime posture fields are aligned enough to pause by default
- compatibility with existing certificate posture consumers remains intact

It does **not** mean:

- TLS lifecycle strategy has been redesigned
- certificate management policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared TLS coverage baseline assessment
- explicit stop-line after posture field alignment
- architecture/index documentation updates

Out of scope:

- certificate issuance workflow redesign
- transport policy changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase356-shared-tls-coverage-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestTLSManagerRuntimeMetricsReady|TestTLSManagerRuntimeMetricsReloadDue|TestTLSManagerRuntimeMetricsUnobserved|TestTLSManagerRuntimeMetricsDegraded'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase356-shared-tls-coverage-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase356-shared-tls-coverage-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared TLS coverage alignment as a stable
  baseline with a stop-line.
- Future TLS posture field expansion is treated as explicit reopen.
