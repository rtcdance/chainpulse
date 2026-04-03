# Phase 306 - WebSocket Connection Baseline Assessment

## Status
Status: Approved

## Summary

Assess the current websocket runtime connection surface after it reached compact
facts, posture, and reliability hint through `GetConnectionMetrics()`.

## Problem

After phase 305, the websocket plugin now exposes a compact connection metrics
surface with:

- running state
- client count
- transport posture
- connection posture
- reliability hint

The repository needs an explicit statement of whether that is enough to treat
the current websocket slice as a stable baseline with a stop-line, rather than
continuing to add small runtime metrics by default.

## Decision

Classify the current websocket work as:

- `stage-complete for the websocket connection baseline`

This means:

- the current websocket runtime surface is strong enough to pause by default
- the baseline already exposes compact runtime semantics for transport and
  connection state

It does **not** mean:

- websocket message-level parity
- subscription/control plane parity
- broader websocket protocol redesign

## Scope

In scope:

- websocket connection baseline assessment
- explicit stop-line for the current websocket runtime surface
- architecture/index documentation updates

Out of scope:

- websocket message envelope changes
- new websocket API families
- broader cross-protocol parity claims

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase306-websocket-connection-baseline-assessment.md`

## Exit Criteria

- The docs explicitly describe the websocket connection surface as a stable
  baseline with a stop-line.
- Future websocket expansion is treated as an explicit reopen rather than
  default continuation.
