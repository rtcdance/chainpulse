# Phase 310 - Shared TLS Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current shared TLS runtime surface after it reached compact facts,
posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 309, the shared TLS manager now exposes a compact runtime surface
with:

- enablement state
- reload TTL
- certificate posture
- reload posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current shared TLS slice as a stable baseline with a stop-line, rather than
continuing to add small runtime metrics by default.

## Decision

Classify the current shared TLS work as:

- `stage-complete for the shared TLS runtime baseline`

This means:

- the current shared TLS runtime surface is strong enough to pause by default
- the baseline already exposes compact runtime semantics for certificate state
  and reload cadence

It does **not** mean:

- transport-specific runtime wiring is complete everywhere
- TLS rotation orchestration has been redesigned
- broader control-plane parity has been achieved

## Scope

In scope:

- shared TLS runtime baseline assessment
- explicit stop-line for the current shared TLS runtime surface
- architecture/index documentation updates

Out of scope:

- plugin-specific runtime integration work
- TLS file-watching redesign
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase310-shared-tls-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the shared TLS runtime surface as a stable
  baseline with a stop-line.
- Future shared TLS expansion is treated as an explicit reopen rather than
  default continuation.
