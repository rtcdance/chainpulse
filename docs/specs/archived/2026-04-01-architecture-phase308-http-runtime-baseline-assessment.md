# Phase 308 - HTTP Runtime Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current HTTP runtime surface after it reached compact facts,
posture, and reliability hint through `GetRuntimeMetrics()`.

## Problem

After phase 307, the HTTP plugin now exposes a compact runtime surface with:

- running state
- route count
- transport posture
- route posture
- runtime posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current HTTP slice as a stable baseline with a stop-line, rather than
continuing to add small runtime metrics by default.

## Decision

Classify the current HTTP work as:

- `stage-complete for the HTTP runtime baseline`

This means:

- the current HTTP runtime surface is strong enough to pause by default
- the baseline already exposes compact runtime semantics for route coverage and
  transport state

It does **not** mean:

- HTTP request-level source parity
- response/meta contract parity
- broader HTTP control-plane redesign

## Scope

In scope:

- HTTP runtime baseline assessment
- explicit stop-line for the current HTTP runtime surface
- architecture/index documentation updates

Out of scope:

- HTTP request/response contract changes
- new HTTP API families
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase308-http-runtime-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the HTTP runtime surface as a stable baseline
  with a stop-line.
- Future HTTP runtime expansion is treated as an explicit reopen rather than
  default continuation.
