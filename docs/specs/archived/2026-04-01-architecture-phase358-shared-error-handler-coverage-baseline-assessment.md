# Phase 358 - Shared Error Handler Coverage Baseline Assessment

## Status
Status: Approved

## Summary

Assess the shared error-handler runtime surface after coverage posture
alignment and compatibility preservation.

## Problem

After phase 357, shared error-handler runtime metrics now expose:

- `coverage_posture`
- `circuit_posture` (compatibility)
- `retry_posture`
- `reliability_hint`

The repository needs an explicit statement on whether this aligned runtime
surface is sufficient to treat error-handler posture semantics as a stable
baseline stop-line.

## Decision

Classify the current shared error-handler coverage alignment as:

- `stage-complete for the shared error-handler coverage baseline`

This means:

- error-handler runtime posture fields are aligned enough to pause by default
- compatibility with existing circuit posture consumers remains intact

It does **not** mean:

- circuit-breaker strategy has been redesigned
- retry policy is finalized
- broader control-plane semantics are complete

## Scope

In scope:

- shared error-handler coverage baseline assessment
- explicit stop-line after posture field alignment
- architecture/index documentation updates

Out of scope:

- error-handling workflow redesign
- retry/circuit policy semantics changes
- broader service orchestration guarantees

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase358-shared-error-handler-coverage-baseline-assessment.md`
- `go test ./pkg/plugins/api/shared -run 'TestErrorHandlerRuntimeMetricsReady|TestErrorHandlerRuntimeMetricsOpenCircuit|TestErrorHandlerRuntimeMetricsProbing'`

## Verification Summary

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase358-shared-error-handler-coverage-baseline-assessment.md` should pass while the spec remains in `Approved` state.
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase358-shared-error-handler-coverage-baseline-assessment.md` passed.

## Exit Criteria

- The docs explicitly describe shared error-handler coverage alignment as a
  stable baseline with a stop-line.
- Future error-handler posture field expansion is treated as explicit reopen.
