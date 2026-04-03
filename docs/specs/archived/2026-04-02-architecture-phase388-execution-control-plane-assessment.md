# Phase 388 - Execution Control Plane Assessment

## Status
Status: Approved

## Summary

Assess the execution control-plane state after `puller` gained a minimal
writable runtime control surface for pausing and resuming the polling loop.

## Problem

The execution-service line now has a strong read-only operator surface:

- `/health*`
- `/metrics`
- `/runtime/summary`

and it also has a first writable control slice:

- `puller` `GET /runtime/control`
- `puller` `POST /runtime/control/pause`
- `puller` `POST /runtime/control/resume`

The repository needs an explicit stage assessment before this writable control
surface is copied to other services by inertia.

## Decision

Record the current execution control-plane state as:

- a **minimal writable control pilot established on `puller`**

This means:

- the repository now has one real writable execution control slice
- the slice is intentionally scoped to the polling loop
- broader execution control-plane work should be treated as an explicit reopen
  from this pilot rather than default continuation

## Scope

In scope:

- update the execution-service coverage summary
- classify the current writable control state as a pilot
- define reopen conditions for wider execution control-plane work

Out of scope:

- new writable control endpoints
- `event-processor` parity implementation
- durable or distributed control coordination

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase388-execution-control-plane-assessment.md`

## Exit Criteria

- The repository explicitly records the current execution control-plane state.
- Future writable control expansion is framed as a deliberate reopen from the
  current pilot instead of an implicit next phase.
