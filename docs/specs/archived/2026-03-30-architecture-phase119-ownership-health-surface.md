# Phase 119 Ownership Health Surface

## Title
Phase 119 - Expose monolithic ownership summary and mode through `/health` details

## Type
- architecture
- indexing
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
2026-03-30

## Related Modules
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `pkg/plugins/api/health_check_handler.go`
- `docs/specs/2026-03-30-architecture-phase118-ownership-summary-metrics.md`

## Context
Phase 118 moved service-level ownership rollout state into runtime metrics, but
operators still cannot see that state from the HTTP health surface that is most
commonly queried during debugging and readiness validation.

## Problem Statement
Without ownership summary details in `/health`, engineers must combine terminal
output and runtime metrics to understand whether monolithic indexing is still
`legacy-only`, in `shadow`, or already `runtime-owned`.

## Scope
- Add an additive runtime component/details extension point to the health check
  handler.
- Expose monolithic ownership summary as an `indexing_runtime` component in
  `/health` and `/health/components`.
- Include at least:
  - `ownership_mode`
  - `shadow_owned_events`
  - `legacy_owned_events`
  - `ownership_chains`
- Add focused tests for:
  - health handler runtime component injection
  - monolithic ownership component content

## Non-Goals
- No change to health endpoint status-code semantics.
- No microservice wiring.
- No ownership rollout behavior change.

## Selected Approach
- Keep the health handler contract backward compatible by extending
  `ComponentStatus` with optional `details`.
- Add a small optional provider hook so monolithic mode can inject ownership
  state without coupling the API package to indexing internals.
- Build the ownership component in `cmd/monolithic/chainpulse/main.go` from the
  existing service-level aggregation helpers.

## Data / Contract Impact
- `/health` and `/health/components` may now include an additional
  `indexing_runtime` component with a stable detail map.
- Existing component fields remain unchanged.

## Observability
- Ownership rollout state becomes visible in:
  - terminal summary
  - runtime metrics
  - HTTP health details

## Risks
- Low risk; additive health payload only.
- Cached health responses will cache ownership details for the existing health
  interval window, which is acceptable for the current debugging surface.

## Rollback Plan
- Remove the optional runtime component provider and monolithic ownership health
  wiring.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase119-ownership-health-surface.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase119-ownership-health-surface.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first HTTP health-surface exposure for ownership rollout
  state in monolithic mode.

## Implementation Summary
- Extended `ComponentStatus` with optional `details`.
- Added an optional runtime component provider to `HealthCheckHandler`.
- Monolithic mode now injects an `indexing_runtime` health component carrying:
  - `ownership_mode`
  - `shadow_owned_events`
  - `legacy_owned_events`
  - `ownership_chains`
- Added focused tests for health handler injection and monolithic ownership
  component construction.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase119-ownership-health-surface.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
