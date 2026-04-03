# Phase 330 - Core API Router Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current core API-router runtime surface after it reached compact
facts, posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 329, the core API router now exposes a compact runtime surface
with:

- raw router metrics
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current core API-router slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current core API-router work as:

- `stage-complete for the core API-router runtime baseline`

This means:

- the current core API-router runtime surface is strong enough to pause by
  default
- the baseline already exposes compact runtime semantics for route coverage and
  router readiness

It does **not** mean:

- routing behavior has been redesigned
- per-route diagnostics are complete
- broader control-plane semantics have been finalized

## Scope

In scope:

- core API-router runtime baseline assessment
- explicit stop-line for the current core API-router runtime surface
- architecture/index documentation updates

Out of scope:

- routing redesign
- per-route contract changes
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase330-core-api-router-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the core API-router runtime surface as a stable
  baseline with a stop-line.
- Future core API-router expansion is treated as an explicit reopen rather than
  default continuation.
