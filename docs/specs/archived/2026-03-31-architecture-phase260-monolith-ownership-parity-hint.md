# Phase 260 - Monolith Ownership Parity Hint

## Status
Status: Approved

## Why
- The shared ownership parity source layer already carries raw monolith
  readiness facts and a compact monolith parity posture.
- The next low-risk improvement is a shared recommendation/hint layer so future
  route-oriented parity work can consume a stable next-step suggestion instead
  of re-deriving it from posture ad hoc.

## Scope
- Add a shared monolith ownership parity hint helper in `pkg/plugins/api`.
- Keep current route-oriented microservice behavior unchanged.

## Implementation
- Extend `RouteOwnershipParitySourceSnapshot` with `MonolithParityHint`.
- Add:
  - `BuildMonolithOwnershipParityHint(...)`
- Populate the hint through
  `BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(...)`.
- Add focused tests for all compact monolith parity postures.

## Validation
- Run `go test ./pkg/plugins/api/...`

## Exit Criteria
- Monolith readiness-backed ownership parity snapshots now include a shared
  recommendation/hint layer in addition to posture.
- The next route-oriented parity step can reuse a stable monolith posture
  recommendation instead of rebuilding it locally.
