# Phase 207 - Event Processor Kafka Activity Signal

## Status
Status: Approved

## Why
- `event-processor` rollout reports already reflect dependency health, but they
  still do not expose any direct evidence of runtime Kafka activity.
- A lightweight next signal is to surface Kafka activity counts already present
  in `KafkaMQ.Health().Details`.

## Scope
- Keep the rollout report contract unchanged.
- Extend `event-processor` runtime-derived rollout state with lightweight Kafka
  activity details.
- Fold those details into advisory reasons.

## Implementation
- Extract from Kafka health details:
  - `message_count`
  - `error_count`
- Add those fields to `eventProcessorRolloutRuntimeState`.
- Append activity details to runtime-derived advisory reasons:
  - `kafka_message_count`
  - `kafka_error_count`

## Validation
- Add focused runtime support coverage for Kafka activity extraction.
- Add focused wired-handler coverage showing rollout reasons include Kafka
  activity details.
- Run focused `event-processor` rollout producer/runtime support tests.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `event-processor` runtime-derived rollout reports expose lightweight Kafka
  activity without changing the external payload shape.
- The new signal remains additive and does not redefine existing rollout enums.
