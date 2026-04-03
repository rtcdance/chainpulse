# Phase 217 - Shared Execution Progress Facade

## Status
Status: Approved

## Why
- Phase 215 aligned `puller` and `event-processor` on shared typed execution
  progress snapshots.
- Phase 216 aligned them on shared snapshot-to-reason appenders.
- The next useful slice is to stop making each service choose its own appender
  path and let both services hand one shared execution-progress facade to the
  rollout reason layer.

## Scope
- Keep rollout semantics unchanged.
- Add one shared execution-progress facade that can carry poll progress,
  consumer progress, or both.
- Add one shared reason helper that appends whichever execution-progress
  snapshots are present.
- Repoint `puller` and `event-processor` to the shared facade entrypoint.

## Implementation
- Add `RolloutExecutionProgress` to the shared execution-progress contract.
- Add `AppendRolloutExecutionProgressReason(...)` to the shared reason helper.
- Keep the existing poll-progress and consumer-progress appenders as the leaf
  formatting helpers.
- Update:
  - `puller` rollout producer
  - `event-processor` rollout producer
  so both services call the shared facade helper instead of directly choosing
  individual appenders.

## Validation
- Run focused `puller` rollout producer/runtime support tests.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Both execution services share one execution-progress facade layer above the
  leaf appenders.
- Existing rollout payload shape and semantics remain unchanged.
