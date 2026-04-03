# Phase 209 - Puller Poll Activity State

## Status
Status: Approved

## Why
- Phase 206 added lightweight pull-loop progress to `puller` rollout reasons,
  but raw counters alone still leave one important question open:
  is the poll loop currently active, or has it gone stale?
- The next useful slice is a stronger semantic derived from the same lightweight
  data, without introducing a new external contract.

## Scope
- Keep the rollout report payload shape unchanged.
- Derive a compact `poll_activity_state` from:
  - poll interval
  - poll count
  - last poll unix timestamp
- Fold that derived state into `puller` runtime-derived rollout reasons.

## Implementation
- Add a small classifier for pull-loop activity:
  - `no-polls-yet`
  - `active`
  - `stale`
- Thread the derived state through the `puller` runtime rollout state builder.
- Append `poll_activity_state` to runtime-derived advisory reasons.
- Keep existing rollout enums and posture signals unchanged.

## Validation
- Add focused classifier coverage for the three activity states.
- Add focused runtime-state coverage for derived activity-state ingestion.
- Add wired-handler coverage showing rollout reasons include
  `poll_activity_state`.
- Run focused `puller` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `puller` rollout reasons expose a stronger progress semantic than raw poll
  counters alone.
- The new signal remains additive and does not redefine the rollout report
  contract or existing rollout enums.
