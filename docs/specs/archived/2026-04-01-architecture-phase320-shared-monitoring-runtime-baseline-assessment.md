# Phase 320 - Shared Monitoring Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current shared monitoring runtime surface after it reached compact
facts, posture, and reliability hint through `GetRuntimeMetrics(protocol)`.

## Problem

After phase 319, shared monitoring now exposes a compact runtime surface with:

- raw monitoring counters
- coverage posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current shared monitoring slice as a stable baseline with a stop-line,
rather than continuing to add small runtime metrics by default.

## Decision

Classify the current shared monitoring work as:

- `stage-complete for the shared monitoring runtime baseline`

This means:

- the current shared monitoring runtime surface is strong enough to pause by
  default
- the baseline already exposes compact runtime semantics for observed traffic
  mix and request health

It does **not** mean:

- protocol instrumentation has been redesigned
- per-request tracing is complete
- broader control-plane semantics have been finalized

## Scope

In scope:

- shared monitoring runtime baseline assessment
- explicit stop-line for the current shared monitoring runtime surface
- architecture/index documentation updates

Out of scope:

- instrumentation redesign
- per-request tracing contract changes
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase320-shared-monitoring-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the shared monitoring runtime surface as a
  stable baseline with a stop-line.
- Future shared monitoring expansion is treated as an explicit reopen rather
  than default continuation.
