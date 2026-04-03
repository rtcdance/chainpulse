# Phase 218 - Shared Execution Progress Builder

## Status
Status: Approved

## Why
- Phase 217 aligned `puller` and `event-processor` on one shared
  execution-progress facade entrypoint.
- The services still assembled that facade inline inside their rollout
  producers.
- The next useful slice is to give both services one shared input+builder
  path for execution-progress assembly.

## Scope
- Keep rollout semantics unchanged.
- Add one shared execution-progress input model.
- Add one shared builder that normalizes optional poll and consumer progress
  snapshots into the shared execution-progress facade.
- Repoint `puller` and `event-processor` to the shared builder.

## Implementation
- Add `RolloutExecutionProgressInput`.
- Add `BuildRolloutExecutionProgress(...)`.
- Omit empty poll and consumer snapshots from the shared facade so producers do
  not need to hand-roll presence checks.
- Update:
  - `puller` rollout producer
  - `event-processor` rollout producer
  to use the shared builder before appending progress reason details.
- Add contract-level tests for:
  - populated execution-progress inputs
  - empty execution-progress inputs

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Both execution services share one execution-progress input+builder layer.
- Existing rollout payload shape and semantics remain unchanged.
