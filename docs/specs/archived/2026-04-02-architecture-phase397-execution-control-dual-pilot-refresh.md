# Phase 397 - Execution Control Dual Pilot Refresh

## Status
Status: Approved

## Summary

Refresh the execution control-plane state after `puller` and
`event-processor` now both expose real writable runtime-control slices.

## Problem

The repository no longer has only one writable execution-control pilot:

- `puller` owns a polling-loop pause/resume slice
- `event-processor` now owns a consume-loop intake pause/resume slice

Without an updated assessment, the current state is easy to misread in two
different ways:

- as if execution control is still only a single-service pilot
- as if the two services now form a shared, fully aligned control baseline

Both readings are too coarse.

## Decision

Record the current execution control-plane state as:

- **service-shaped dual-pilot baseline**

Interpretation:

- the line now has two real writable control slices
- both slices are intentionally narrow and operationally honest
- but they are still service-shaped and not yet one shared control abstraction

## Scope

In scope:

- refresh execution-control wording in the coverage summary
- record the new maturity level after phase 396
- define the stop-line for the current dual-pilot state

Out of scope:

- introducing a shared control contract
- expanding control to more services
- redesigning auth, policy, or distributed coordination

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase397-execution-control-dual-pilot-refresh.md`

## Exit Criteria

- The repository explicitly records that execution control is no longer
  single-pilot.
- The repository also explicitly records that the line is still service-shaped
  rather than a shared baseline.
- Future control-plane expansion is framed as a deliberate reopen from this
  dual-pilot boundary.
