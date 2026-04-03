# Phase 213 - Kafka Health Consumer Lag Export

## Status
Status: Approved

## Why
- Phase 211 and Phase 212 gave `event-processor` a consumer-progress posture
  and a dedicated extractor, but the extractor still depended primarily on
  side-channel Kafka interfaces.
- The next useful slice is to export a minimal consumer-progress view directly
  from Kafka health details so rollout helpers can consume a more standard
  runtime surface.

## Scope
- Keep rollout report semantics unchanged.
- Export minimal consumer progress from Kafka `Health().Details`.
- Make the `event-processor` consumer-progress helper prefer health details,
  while preserving existing interface-based fallbacks.

## Implementation
- Extend Kafka health details with:
  - `active_consumers`
  - `consumer_group_lag`
  - `consumer_group_metrics`
- Update the `event-processor` consumer-progress extractor to read those health
  fields first, then fall back to the existing helper interfaces if needed.
- Add focused coverage for:
  - Kafka health export
  - event-processor health-details fallback

## Validation
- Run focused Kafka plugin tests using `kafka_mq.go`, `kafka_mq_test.go`, and
  `kafka_mq_task10_test.go`.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- Kafka health exposes a minimal consumer progress export that rollout helpers
  can reuse.
- Existing rollout report shape and semantics remain unchanged.
