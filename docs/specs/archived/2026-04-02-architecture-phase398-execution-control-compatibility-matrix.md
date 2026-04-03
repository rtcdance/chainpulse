# Phase 398 - Execution Control Compatibility Matrix

## Status
Status: Approved

## Summary

Document the current compatibility shape between the two execution-control
pilots so future shared-control work starts from an explicit matrix instead of
informal comparison.

## Problem

The repository now has two real writable execution-control pilots:

- `puller` polling-loop control
- `event-processor` consume-loop intake control

That is a stronger state than a single pilot, but it also introduces a new
risk:

- people may assume the pilots are already fully aligned
- or assume they are too different to compare usefully

Both assumptions are unhelpful.

## Decision

Record an explicit compatibility matrix with two categories:

- **aligned control shape**
- **intentional service-specific differences**

### Aligned control shape

The current pilots are aligned on:

- read route: `GET /runtime/control`
- write routes: one pause action and one resume action
- response envelope shape:
  - `service`
  - `timestamp`
  - `control`
- shared control facts inside `control`:
  - `paused`
  - `state`
  - `reason`
  - `last_action`
  - `updated_unix`

### Intentional service-specific differences

The current pilots intentionally differ on:

- action target
  - `puller`: polling loop
  - `event-processor`: consume-loop intake
- route naming
  - `puller`: `/pause`, `/resume`
  - `event-processor`: `/pause-intake`, `/resume-intake`
- extra response metadata
  - `event-processor` includes `target=consume-loop-intake`
- runtime-state semantics behind `state`
  - each service maps `state` to its own owned execution boundary

## Scope

In scope:

- record current execution-control compatibility
- clarify which parts are already aligned
- clarify which parts remain intentionally different

Out of scope:

- implementing a shared control contract
- renaming existing routes
- changing control payloads

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase398-execution-control-compatibility-matrix.md`

## Exit Criteria

- The repository explicitly records a compatibility matrix for the current
  execution-control pilots.
- Future shared-control discussions can start from known alignments and known
  intentional differences instead of ad hoc comparison.
