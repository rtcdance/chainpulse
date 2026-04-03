# Phase 257 - Route Ownership Parity Source Abstraction

## Status
Status: Approved

## Why
- The route-oriented ownership parity baseline now shares helpers, work-item
  assembly, and a shared parity state model.
- The next useful step toward deeper parity is a minimal source abstraction so
  later ownership/runtime signals can plug into a stable provider boundary
  instead of being threaded ad hoc through each service.

## Scope
- Add a minimal shared route ownership parity source abstraction in
  `pkg/plugins/api`.
- Keep current route-oriented ownership parity semantics unchanged.

## Implementation
- Add:
  - `RouteOwnershipParitySourceSnapshot`
  - `RouteOwnershipParitySource`
  - `RouteOwnershipParitySourceFunc`
  - `BuildRouteOwnershipParityStateFromSource(...)`
- Update:
  - `cmd/microservices/api-service/rollout_report_producer.go`
  - `cmd/microservices/api-gateway/rollout_report_producer.go`
- Route ownership parity state assembly through the shared source boundary.

## Validation
- Run `go test ./pkg/plugins/api/...`
- Run focused `api-service` rollout tests.
- Run focused `api-gateway` rollout tests.

## Exit Criteria
- Route-oriented ownership parity no longer assumes only direct bool-to-state
  assembly.
- A shared provider boundary exists for the first future ownership/runtime
  source integration step.
