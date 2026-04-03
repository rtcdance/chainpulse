# Phase 258 - Monolith Readiness Ownership Parity Source

## Status
Status: Approved

## Why
- The route-oriented ownership parity line already had a shared source
  abstraction, but it still only consumed service-local bool assembly.
- The next useful step was to land the first real shared ownership/runtime
  source path using monolith readiness details that already exist and are
  stable.

## Scope
- Add a shared adapter that converts monolith readiness rollout details into a
  route ownership parity source snapshot.
- Keep current route-oriented microservice semantics unchanged.

## Implementation
- Extend `RouteOwnershipParitySourceSnapshot` to carry:
  - monolith ownership mode
  - monolith rollout ready
  - monolith rollout status
  - monolith rollout reason
- Add:
  - `BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(...)`
- Add focused tests covering:
  - `shadow`
  - `runtime-owned`
  - `idle`

## Validation
- Run `go test ./pkg/plugins/api/...`

## Exit Criteria
- The shared ownership parity source layer has a real monolith-readiness-backed
  adapter instead of only direct bool assembly.
- The next deeper ownership/runtime parity step can reuse that adapter path
  instead of inventing a new source boundary.
