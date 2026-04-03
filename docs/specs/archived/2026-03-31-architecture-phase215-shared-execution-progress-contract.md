# Phase 215 - Shared Execution Progress Contract

## Status
Status: Approved

## Why
- Phase 212 and Phase 214 gave `event-processor` and `puller` dedicated
  progress extractors, but the typed snapshots still lived separately inside
  each service package.
- The next useful slice is to give both execution services one shared contract
  layer for lightweight execution progress snapshots.

## Scope
- Keep rollout semantics unchanged.
- Add shared typed execution-progress snapshot contracts.
- Repoint `puller` and `event-processor` progress helpers to use those shared
  types.

## Implementation
- Add shared API contract types for:
  - poll-loop progress snapshots
  - consumer progress snapshots
- Update the `puller` poll progress helper to return the shared poll progress
  snapshot type.
- Update the `event-processor` consumer progress helper to return the shared
  consumer progress snapshot type.

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Both execution services share the same typed execution-progress contract
  layer.
- Existing rollout payload shape and semantics remain unchanged.
