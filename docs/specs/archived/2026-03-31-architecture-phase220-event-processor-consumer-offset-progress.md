# Phase 220 - Event-Processor Consumer Offset Progress

## Status
Status: Approved

## Why
- The `event-processor` rollout report already exposes lightweight Kafka
  activity, active consumers, and consumer lag.
- That is useful, but it still does not tell operators whether the consumer
  side has advanced any tracked offsets.
- The next useful slice is to expose a minimal offset-based progress signal
  through the existing Kafka health path and thread it into the
  `event-processor` rollout report.

## Scope
- Keep rollout semantics unchanged.
- Add a minimal Kafka health export for the maximum tracked consumer offset.
- Thread that offset into `event-processor` consumer progress snapshots and
  rollout reasons.
- Reuse the shared execution-progress contract and reason helpers.

## Implementation
- Export `max_tracked_offset` from Kafka health details and consumer-group
  status.
- Extend `RolloutConsumerProgressSnapshot` with `CurrentOffset`.
- Extend shared execution-progress reason helpers so consumer progress details
  can append `consumer_offset`.
- Update `event-processor` consumer-progress extraction to read:
  - `active_consumers`
  - `consumer_group_lag`
  - `max_tracked_offset`
- Update `event-processor` runtime rollout state and tests to carry the new
  offset signal through focused producer/runtime-support coverage.

## Validation
- Run focused Kafka MQ file-based tests.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Kafka health exports a minimal tracked-offset progress signal.
- `event-processor` rollout reasons can expose `consumer_offset` through the
  shared execution-progress path without changing higher-level rollout policy
  semantics.
