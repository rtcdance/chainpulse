# Phase 211 - Event Processor Consumer Progress State

## Status
Status: Approved

## Why
- Phase 210 upgraded Kafka activity into `active` versus `stalled`, but the
  event processor still lacked one important consumer-side view:
  does the consumer group appear idle, healthy, or lagging?
- The Kafka plugin already exposes conservative consumer-group status and lag
  hooks, so the next slice is to reuse those signals instead of inventing a new
  progress source.

## Scope
- Keep the rollout report contract unchanged.
- Reuse Kafka consumer-group status and metrics when they are available.
- Derive a compact `consumer_progress_state` and fold it into
  runtime-derived advisory reasons.

## Implementation
- Extend runtime rollout state with:
  - `active_consumers`
  - `consumer_lag`
  - `consumer_progress_state`
- Reuse Kafka plugin methods when present:
  - `GetConsumerGroupStatus()`
  - `GetConsumerGroupMetrics()`
- Add a conservative progress classifier:
  - `idle`
  - `lagging`
  - `active`
  - `monitoring`
- Append consumer-progress details to runtime-derived advisory reasons.

## Validation
- Add focused classifier coverage for the new consumer progress states.
- Add focused runtime-state coverage for consumer-group status/lag ingestion.
- Add focused producer and wired-handler coverage showing rollout reasons
  include consumer progress details.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `event-processor` rollout reasons expose one more real consumer-side progress
  semantic beyond Kafka activity counts alone.
- The new signal remains additive and does not redefine the rollout report
  contract or existing rollout enums.
