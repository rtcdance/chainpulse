# Phase 255 - Shared Route Ownership Work Item Helper

## Status
Status: Approved

## Why
- After moving route-oriented ownership parity hints into shared helpers, the
  next remaining duplication sat in the approval work-item assembly.
- `api-service` and `api-gateway` still built ownership parity work items with
  service-local plumbing even though the actual semantics were the same shape.

## Scope
- Move route-oriented ownership parity approval work-item assembly into
  `pkg/plugins/api`.
- Keep current observable work-item semantics unchanged.

## Implementation
- Add a shared ownership parity approval work-item helper in
  `pkg/plugins/api/rollout_ownership_parity.go`.
- Update:
  - `cmd/microservices/api-service/rollout_report_producer.go`
  - `cmd/microservices/api-gateway/rollout_report_producer.go`
- Extend focused tests so shared helper output is validated directly.

## Validation
- Run `go test ./pkg/plugins/api/...`
- Run focused `api-service` rollout tests.
- Run focused `api-gateway` rollout tests.

## Exit Criteria
- Route-oriented ownership parity work items are assembled through a shared
  helper instead of duplicated service-local logic.
- `api-service` and `api-gateway` keep the same observable ownership parity
  work-item semantics after the refactor.
