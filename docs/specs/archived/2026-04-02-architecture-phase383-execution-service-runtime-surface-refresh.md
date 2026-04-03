# Phase 383 - Execution Service Runtime Surface Refresh

## Status
Status: Approved

## Summary

Refresh the execution-service runtime-surface assessment after both
`event-processor` and `puller` now expose a real `/metrics` route in addition
to their existing `/health*` runtime surfaces.

## Problem

Phase 272 captured the execution-service line at the point where both services
had a minimal symmetric health/runtime baseline. Since then, the line has moved
forward:

- `event-processor` now exposes `/metrics`
- `puller` now exposes `/metrics`

The repository needs an updated written stage boundary so the execution service
line is assessed against its current runtime surface instead of the older
health-only baseline.

## Decision

Refresh the architecture coverage summary to classify the current execution
service line as a stronger runtime-surface baseline that includes:

- `/health`
- `/health/ready`
- `/health/live`
- `/health/components`
- `/health/rollout`
- `/metrics`

Keep the slice documentation-only:

- no new runtime implementation
- no metrics contract redesign
- no broader service-plane expansion beyond the completed runtime routes

## Scope

In scope:

- update execution-service stage wording in the coverage summary
- record the stronger symmetric `/health* + /metrics` baseline
- define an updated stop-line and reopen conditions

Out of scope:

- new execution-service routes
- Prometheus or metrics-format redesign
- non-execution service changes

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase383-execution-service-runtime-surface-refresh.md`

## Exit Criteria

- The repository explicitly records the stronger execution-service runtime
  baseline.
- Future execution-service work is framed as an explicit reopen from this new
  baseline rather than an implicit continuation from the older health-only one.
