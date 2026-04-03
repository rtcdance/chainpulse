# Phase 266 - Monolith Parity Recommendation Bundle

## Status
Status: Approved

## Why
- Route-oriented services already share monolith parity posture, hint, target
  decision, and action guidance.
- The next cleanup step is to treat that quartet as a shared recommendation
  bundle so validators and future endgame assessment can consume one stable
  shape instead of four separate arguments.

## Scope
- Add a shared monolith parity recommendation bundle in `pkg/plugins/api`.
- Add a shared validator entrypoint for the recommendation bundle.
- Move route-oriented focused tests onto the shared bundle validator while
  keeping the existing reason surface unchanged.

## Implementation
- Added `MonolithOwnershipParityRecommendationBundle`.
- Added `BuildMonolithOwnershipParityRecommendationBundle(...)`.
- Added `ValidateRouteMonolithOwnershipParityRecommendationBundle(...)`.
- Updated route-oriented focused tests in:
  - `cmd/microservices/api-service`
  - `cmd/microservices/api-gateway`

## Validation
- `go test ./pkg/plugins/api/...`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_report_integration_test.go`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_runtime_support.go rollout_runtime_support_test.go main.go`

## Exit Criteria
- Route-oriented monolith parity output can be consumed as a single shared
  recommendation bundle.
- Shared validators and route-oriented tests no longer need to reason about
  the recommendation fields independently.
