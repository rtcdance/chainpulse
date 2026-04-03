# Phase 328 - Core Plugin Registry Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core plugin-registry runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 327, the core plugin registry now exposes a compact runtime
surface with:

- raw plugin-registry metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core plugin-registry slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core plugin-registry work as:

- `stage-complete for the core plugin-registry runtime baseline`

This means:

- the current core plugin-registry runtime surface is strong enough to pause by
  default
- the baseline already exposes compact runtime semantics for plugin-state
  coverage and lifecycle readiness

It does **not** mean:

- plugin lifecycle orchestration has been redesigned
- per-plugin diagnostics are complete
- broader control-plane semantics have been finalized

## Scope

In scope:

- core plugin-registry runtime baseline assessment
- explicit stop-line for the current core plugin-registry runtime surface
- architecture/index documentation updates

Out of scope:

- lifecycle orchestration redesign
- per-plugin contract changes
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase328-core-plugin-registry-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the core plugin-registry runtime surface as a
  stable baseline with a stop-line.
- Future core plugin-registry expansion is treated as an explicit reopen
  rather than default continuation.
