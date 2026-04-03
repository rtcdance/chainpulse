# Phase 153 Ownership Rollout Health Report Surface

## Title
Phase 153 - Add dedicated `/health/rollout` surface for monolithic ownership rollout status

## Type
- feature
- architecture
- observability

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Implemented

## Owner
platform-team

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-31

## Related Modules
- `pkg/plugins/api/health_check_handler.go`
- `pkg/plugins/api/gateway_router_integration.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `docs/specs/2026-03-31-architecture-phase152-rollout-presenter-accessor-sections.md`

## Context
The monolithic ownership rollout control plane now has a fairly complete
summary, readiness, metrics, and console surface. Operators still need to
reconstruct a coherent report by stitching together `/health`, `/health/ready`,
and logs.

## Problem Statement
Without a dedicated rollout report endpoint, there is no single HTTP surface
that returns the current ownership rollout snapshot in one place for operators,
debug tooling, or future dashboards.

## Scope
- Add an optional rollout report provider to the health check handler.
- Add a dedicated `GET /health/rollout` route through gateway runtime routing.
- Expose the current monolithic ownership rollout summary snapshot through the
  new endpoint.

## Non-Goals
- No change to readiness semantics.
- No rollout decision logic changes.
- No authentication or authorization changes.

## Selected Approach
- Introduce a lightweight `RolloutReportResponse` payload served by the health
  handler when a rollout report provider is registered.
- Wire the monolithic runtime to provide `buildOwnershipRolloutSummary(...)`
  output directly.
- Keep the route read-only and non-blocking.

## Data / Contract Impact
- Adds a new HTTP response type and route:
  - `GET /health/rollout`
- Existing `/health`, `/health/ready`, `/health/live`, and `/health/components`
  contracts remain unchanged.

## Observability
- The new endpoint is itself an observability/report surface for rollout state.
- Existing metrics and console/log surfaces remain unchanged.

## Risks
- Medium-low: exposing a rollout snapshot through HTTP could drift from the
  internal summary if the provider wiring is incomplete.

## Rollback Plan
- Remove the rollout report provider, route, and response type, falling back to
  `/health` and `/health/ready` only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase153-ownership-rollout-health-report-surface.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase153-ownership-rollout-health-report-surface.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first dedicated report/export surface for the monolithic
  ownership rollout control plane.

## Implementation Notes
- Added optional rollout report provider support to the health check handler.
- Added `GET /health/rollout` route wiring in gateway runtime integration.
- Wired the monolithic runtime to expose `buildOwnershipRolloutSummary(...)`
  through the new handler surface.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase153-ownership-rollout-health-report-surface.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
