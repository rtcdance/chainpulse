# Phase 196 - API Gateway Rollout Runtime Wiring

## Status
Status: Approved

## Why
- Phase 195 added an `api-gateway` rollout producer skeleton, but it still
  lived as a disconnected producer path rather than a service-entrypoint
  runtime wiring slice.
- The next additive step is to wire the shared rollout producer into the
  gateway's runtime health surface without pulling in full retrieval-service
  initialization.

## Scope
- Wire `api-gateway` runtime rollout components at the service entrypoint.
- Keep the rollout report contract unchanged.
- Keep the slice additive and low-risk.

## Implementation
- Add a small runtime-support helper that:
  - builds a minimal health handler backed by a no-op database manager
  - wires event query, event subscription, and health handlers onto the gateway
  - registers the `api-gateway` rollout report producer on the health handler
- Update `api-gateway` main to use the helper and propagate `instance_id`.
- Add focused tests for:
  - component wiring
  - `/health/rollout` route visibility through gateway integration

## Validation
- Run focused Go tests for the new helper and producer files.
- Run `go test ./pkg/plugins/api/...`.

## Exit Criteria
- `api-gateway` runtime wiring includes a real shared rollout producer path.
- The shared `/health/rollout` surface is reachable through gateway
  integration in focused tests.
- External rollout contract shape remains unchanged.
