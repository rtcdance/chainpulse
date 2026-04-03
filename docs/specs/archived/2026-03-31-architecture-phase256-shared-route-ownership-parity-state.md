# Phase 256 - Shared Route Ownership Parity State

## Status
Status: Approved

## Why
- After moving route-oriented ownership parity hints and approval work items
  into shared helpers, the next remaining duplication was the input assembly
  itself.
- `api-service` and `api-gateway` were still each reconstructing the same
  small ownership parity state shape before handing it to shared helpers.

## Scope
- Add a shared route-oriented ownership parity state/input model in
  `pkg/plugins/api`.
- Keep current ownership parity output semantics unchanged.

## Implementation
- Add shared route ownership parity state assembly in:
  - `pkg/plugins/api/rollout_ownership_parity.go`
- Update:
  - `cmd/microservices/api-service/rollout_report_producer.go`
  - `cmd/microservices/api-gateway/rollout_report_producer.go`
- Route both advisory reason assembly and approval work-item assembly through
  the shared state model.

## Validation
- Run `go test ./pkg/plugins/api/...`
- Run focused `api-service` rollout tests.
- Run focused `api-gateway` rollout tests.

## Exit Criteria
- Route-oriented ownership parity uses a shared state/input model instead of
  rebuilding the same parity inputs separately in each service.
- `api-service` and `api-gateway` keep the same observable ownership parity
  semantics after the refactor.
