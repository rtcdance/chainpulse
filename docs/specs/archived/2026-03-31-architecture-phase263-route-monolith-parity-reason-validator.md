# Phase 263 - Route Monolith Parity Reason Validator

## Status
Status: Approved

## Why
- Both route-oriented services now expose shared monolith-backed parity
  posture and hint in advisory reason.
- That new reason surface should be locked behind a shared validator instead of
  repeating service-local `strings.Contains(...)` checks.

## Scope
- Add a shared validator for route-oriented monolith parity reason coverage in
  `pkg/plugins/api`.
- Update `api-service` and `api-gateway` focused rollout tests to use the
  shared validator.

## Implementation
- Added `ValidateRouteMonolithOwnershipParityReason(...)`.
- Added focused parity tests for the new validator.
- Replaced duplicated route/service-local monolith parity reason assertions in:
  - `cmd/microservices/api-service`
  - `cmd/microservices/api-gateway`

## Validation
- `go test ./pkg/plugins/api/...`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_report_integration_test.go`
- `go test rollout_report_producer.go rollout_report_sections.go rollout_report_producer_test.go rollout_runtime_support.go rollout_runtime_support_test.go main.go`

## Exit Criteria
- Route-oriented monolith parity posture/hint reason coverage is locked behind
  a shared validator.
- `api-service` and `api-gateway` tests no longer hand-roll the same monolith
  parity reason assertions.
