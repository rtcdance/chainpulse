# Phase 322 - Shared Error Handler Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current shared error-handler runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 321, the shared error handler now exposes a compact runtime surface
with:

- raw error-handler metrics
- circuit posture
- retry posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current shared error-handler slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current shared error-handler work as:

- `stage-complete for the shared error-handler runtime baseline`

This means:

- the current shared error-handler runtime surface is strong enough to pause by
  default
- the baseline already exposes compact runtime semantics for circuit readiness
  and retry posture

It does **not** mean:

- the circuit breaker has been redesigned
- per-error diagnostics are complete
- broader control-plane semantics have been finalized

## Scope

In scope:

- shared error-handler runtime baseline assessment
- explicit stop-line for the current shared error-handler runtime surface
- architecture/index documentation updates

Out of scope:

- retry-policy redesign
- per-error contract changes
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase322-shared-error-handler-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the shared error-handler runtime surface as a
  stable baseline with a stop-line.
- Future shared error-handler expansion is treated as an explicit reopen rather
  than default continuation.
