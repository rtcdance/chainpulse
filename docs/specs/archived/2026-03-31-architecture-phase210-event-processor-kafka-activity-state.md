# Phase 210 - Event Processor Kafka Activity State

## Status
Status: Approved

## Why
- Phase 207 exposed lightweight Kafka activity counts in `event-processor`
  rollout reasons, but raw counts alone still leave one practical gap:
  are we seeing signs of live activity, or does the processor look stalled?
- The next useful slice is a compact derived state that stays additive and
  builds on the same existing Kafka health details payload.

## Scope
- Keep the rollout report contract unchanged.
- Derive a compact `kafka_activity_state` from existing activity counts.
- Fold that derived state into `event-processor` runtime-derived advisory
  reasons.

## Implementation
- Add a small Kafka activity classifier:
  - `active`
  - `stalled`
- Thread the derived state through the `event-processor` runtime rollout state
  builder.
- Append `kafka_activity_state` to runtime-derived advisory reasons.
- Keep existing rollout enums and posture signals unchanged.

## Validation
- Add focused classifier coverage for the two activity states.
- Add focused runtime-state coverage for derived activity-state ingestion.
- Add focused producer and wired-handler coverage showing rollout reasons
  include `kafka_activity_state`.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `event-processor` rollout reasons expose a stronger activity semantic than
  raw Kafka counters alone.
- The new signal remains additive and does not redefine the rollout report
  contract or existing rollout enums.
