# Phase 216 - Shared Execution Progress Reason Helper

## Status
Status: Approved

## Why
- Phase 215 aligned `puller` and `event-processor` on shared typed execution
  progress snapshots, but the services still appended those snapshots to rollout
  reasons using separate local helper code.
- The next useful slice is to let both services share one progress-to-reason
  helper layer.

## Scope
- Keep rollout semantics unchanged.
- Add shared helpers for writing execution-progress snapshots into rollout
  reason parts.
- Repoint `puller` and `event-processor` to those shared helpers.

## Implementation
- Add shared reason appenders for:
  - poll progress snapshots
  - consumer progress snapshots
- Add contract-level tests for both appenders.
- Replace local progress reason appenders in:
  - `puller`
  - `event-processor`

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Both execution services share one progress-to-reason helper layer.
- Existing rollout payload shape and semantics remain unchanged.
