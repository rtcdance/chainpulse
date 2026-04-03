# Phase 262 - API Gateway Monolith Parity Reason Exposure

## Status
Status: Approved

## Why
- `api-service` already exposes shared monolith parity posture and hint through
  a readiness-backed ownership source adapter.
- `api-gateway` should follow the same deeper route-oriented ownership/runtime
  parity path so both route-facing services consume the same shared signal
  shape.

## Scope
- Add a readiness-backed ownership source adapter path for `api-gateway`.
- Expose shared monolith parity posture and hint in the `api-gateway`
  rollout advisory reason.
- Lock the behavior with focused producer and runtime route coverage.

## Implementation
- Added `newAPIGatewayRolloutReportProducerWithReadinessDetails(...)`.
- Added `buildAPIGatewayOwnershipParitySourceFromReadinessDetails(...)`.
- Added `buildAPIGatewayRuntimeRolloutComponentsWithReadinessDetails(...)`.
- Updated `api-gateway` focused rollout tests to assert:
  - `monolith_parity_posture: ...`
  - `monolith_parity_hint: ...`

## Validation
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_runtime_support.go rollout_runtime_support_test.go main.go`
- `go test ./pkg/plugins/api/...`

## Exit Criteria
- `api-gateway` rollout advisory reason can expose shared monolith parity
  posture and hint through a readiness-backed ownership source adapter.
- Producer and runtime route tests lock the new reason surface while keeping
  the overall ownership parity decision conservative.
