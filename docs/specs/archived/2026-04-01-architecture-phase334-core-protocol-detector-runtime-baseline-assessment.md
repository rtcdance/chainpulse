# Phase 334 - Core Protocol Detector Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core protocol-detector runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 333, the core protocol detector now exposes a compact runtime
surface with:

- raw protocol-detector metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core protocol-detector slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core protocol-detector work as:

- `stage-complete for the core protocol-detector runtime baseline`

This means:

- the current core protocol-detector runtime surface is strong enough to pause
  by default
- the baseline already exposes compact runtime semantics for protocol coverage
  and detector readiness

It does **not** mean:

- protocol detection has been redesigned
- per-request diagnostics are complete
- broader control-plane semantics have been finalized

## Scope

In scope:

- core protocol-detector runtime baseline assessment
- explicit stop-line for the current core protocol-detector runtime surface
- architecture/index documentation updates

Out of scope:

- protocol-detection redesign
- per-request contract changes
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase334-core-protocol-detector-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the core protocol-detector runtime surface as a
  stable baseline with a stop-line.
- Future core protocol-detector expansion is treated as an explicit reopen
  rather than default continuation.
