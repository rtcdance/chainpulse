# Phase 316 - Shared Request Batcher Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current shared request-batcher runtime surface after it reached
compact facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 315, the shared request batcher now exposes a compact runtime
surface with:

- raw batcher counters
- capacity posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current shared request-batcher slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current shared request-batcher work as:

- `stage-complete for the shared request-batcher runtime baseline`

This means:

- the current shared request-batcher runtime surface is strong enough to pause
  by default
- the baseline already exposes compact runtime semantics for fill level and
  batcher health

It does **not** mean:

- processor behavior has been redesigned
- per-request diagnostics are complete
- broader control-plane semantics have been finalized

## Scope

In scope:

- shared request-batcher runtime baseline assessment
- explicit stop-line for the current shared request-batcher runtime surface
- architecture/index documentation updates

Out of scope:

- batching processor redesign
- per-request contract changes
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase316-shared-request-batcher-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the shared request-batcher runtime surface as a
  stable baseline with a stop-line.
- Future shared request-batcher expansion is treated as an explicit reopen
  rather than default continuation.
