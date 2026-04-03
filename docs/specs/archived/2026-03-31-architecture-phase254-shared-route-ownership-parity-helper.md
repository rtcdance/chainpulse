# Phase 254 - Shared Route Ownership Parity Helper

## Status
Status: Approved

## Why
- The route-oriented ownership parity baseline was already present in both
  `api-service` and `api-gateway`, but the implementation was still duplicated.
- That duplication made the baseline more fragile than it needed to be just as
  the work moved from “make the gap visible” toward “prepare for deeper parity”.

## Scope
- Move the shared route-oriented ownership parity marker logic into
  `pkg/plugins/api`.
- Keep current external rollout semantics unchanged for `api-service` and
  `api-gateway`.

## Implementation
- Add shared ownership parity helpers in `pkg/plugins/api` for:
  - hint generation
  - review-field assembly
  - reason-part appending
- Update:
  - `cmd/microservices/api-service/rollout_report_producer.go`
  - `cmd/microservices/api-gateway/rollout_report_producer.go`
- Update focused tests to assert against the shared helper output instead of
  service-local string assembly.

## Validation
- Run `go test ./pkg/plugins/api/...`
- Run focused `api-service` rollout tests.
- Run focused `api-gateway` rollout tests.

## Exit Criteria
- Route-oriented ownership parity markers are assembled through a shared helper.
- `api-service` and `api-gateway` keep the same observable ownership parity
  semantics after the refactor.
