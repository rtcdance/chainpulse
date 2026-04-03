# Phase 126 Ownership Progression Metrics

## Title
Phase 126 - Export effective ownership progression state through runtime metrics

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
- `docs/specs/2026-03-30-architecture-phase125-ownership-effective-progression-state.md`

## Context
Phase 125 introduced a normalized effective progression state, but it is still
only exposed through readiness details.

## Problem Statement
Without metrics for effective progression state, dashboards and alerting cannot
track rollout posture without polling `/health/ready`.

## Scope
- Export effective progression state through runtime gauges.
- Add a stable numeric code mapping for progression states.
- Emit progression metrics alongside existing ownership summary metrics.
- Add focused tests for code mapping and metric emission.

## Non-Goals
- No new dashboards in this phase.
- No cutover enforcement.
- No microservice rollout integration.

## Selected Approach
- Extend the existing ownership summary metric emission helper.
- Export:
  - progression state code
  - policy mode code
  - acknowledgment state code
- Keep string forms on health/readiness; use numeric gauges for metrics.

## Data / Contract Impact
- Runtime metrics expand with progression-related gauges.

## Observability
- Effective progression state becomes visible in metrics, terminal/runtime
  reporting, and readiness details.

## Risks
- Low risk; additive gauges only.

## Rollback Plan
- Remove progression metric code mapping and related metric emission.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase126-ownership-progression-metrics.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase126-ownership-progression-metrics.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first metrics surface for normalized ownership progression
  state.

## Implementation Summary
- Extended ownership summary metric emission with progression-related gauges.
- Added stable numeric mappings for:
  - rollout policy mode
  - rollout acknowledgment state
  - effective progression state
- Running and shutdown summary paths now export progression metrics alongside
  ownership summary metrics.
- Added focused tests for metric code mappings and emission.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase126-ownership-progression-metrics.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
