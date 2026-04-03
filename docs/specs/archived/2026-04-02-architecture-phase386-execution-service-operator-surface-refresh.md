# Phase 386 - Execution Service Operator Surface Refresh

## Status
Status: Approved

## Summary

Refresh the execution-service stage assessment after both `event-processor`
and `puller` now expose a read-only `/runtime/summary` route in addition to
their existing `/health*` and `/metrics` surfaces.

## Problem

Phase 383 recorded a stronger symmetric execution-service baseline around:

- `/health*`
- `/metrics`

Since then, both execution services have moved one step higher and now also
expose a compact read-only operator-facing runtime summary route. The
repository needs an updated written stop-line so future work is assessed
against the current operator surface rather than the older metrics-only
baseline.

## Decision

Refresh the architecture coverage summary to classify the current execution
service line as a stronger read-only operator surface baseline that includes:

- `/health`
- `/health/ready`
- `/health/live`
- `/health/components`
- `/health/rollout`
- `/metrics`
- `/runtime/summary`

Keep the slice documentation-only:

- no new runtime implementation
- no mutable control endpoints
- no metrics or transport redesign

## Scope

In scope:

- update execution-service stage wording in the coverage summary
- record the stronger symmetric read-only operator surface baseline
- define an updated stop-line and reopen conditions

Out of scope:

- write/control endpoints
- new execution-service runtime routes
- non-execution service changes

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase386-execution-service-operator-surface-refresh.md`

## Exit Criteria

- The repository explicitly records the stronger execution-service read-only
  operator baseline.
- Future execution-service work is framed as an explicit reopen from this new
  baseline rather than an implicit continuation from the older
  `/health* + /metrics` stage.
