# Phase 259 - Monolith Ownership Parity Posture

## Status
Status: Approved

## Why
- The shared ownership parity source layer now has a concrete adapter for
  monolith readiness rollout details.
- The next low-risk improvement is to compress those raw monolith facts into a
  compact posture that route-oriented parity code can consume more easily.

## Scope
- Add a compact monolith ownership parity posture classifier in
  `pkg/plugins/api`.
- Keep the current route-oriented microservice behavior unchanged.

## Implementation
- Extend `RouteOwnershipParitySourceSnapshot` with
  `MonolithParityPosture`.
- Add:
  - `ClassifyMonolithOwnershipParityPosture(...)`
- Populate the posture through
  `BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(...)`.
- Add focused tests for:
  - `runtime-owned`
  - `shadow`
  - `legacy-only`
  - `idle`
  - fallback investigate posture

## Validation
- Run `go test ./pkg/plugins/api/...`

## Exit Criteria
- Monolith readiness-backed ownership parity snapshots now include a compact
  posture instead of only raw mode/status/ready facts.
- The next route-oriented parity step can consume this posture layer directly.
