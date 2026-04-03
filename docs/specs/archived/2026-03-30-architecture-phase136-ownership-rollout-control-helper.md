# Phase 136 Ownership Rollout Control Helper

## Title
Phase 136 - Extract ownership rollout control helpers from monolithic entrypoint

## Type
- refactor
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
2026-03-30

## Related Modules
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/ownership_rollout_control.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase135-ownership-rollout-summary-helper.md`

## Context
Phase 135 extracted the rollout summary snapshot and metrics/readiness assembly,
but the monolithic entrypoint still carries the full control-plane type and
classifier graph for advisory, policy, progression, cutover, and approval
signals.

## Problem Statement
Without a dedicated control helper, `main.go` remains a dense source of rollout
decision logic, making future guarded cutover work harder to review and evolve
safely.

## Scope
- Extract rollout control types, code mappings, environment normalization, and
  classifier helpers into a dedicated helper file.
- Keep the summary helper and entrypoint behavior unchanged.
- Keep existing readiness fields, metric names, and console/log wording stable.
- Update tests as needed to validate the extracted helper structure.

## Non-Goals
- No new rollout states.
- No cutover enforcement.
- No microservice integration.

## Selected Approach
- Introduce a focused helper file for ownership rollout control primitives.
- Leave `main.go` responsible for orchestration and presentation only.
- Preserve existing package-level helper visibility so current tests continue
  to exercise the same rollout semantics.

## Data / Contract Impact
- No external contract change intended.
- Existing readiness keys and runtime metric names remain stable.

## Observability
- Refactor only.
- Existing rollout signals remain intact across readiness, metrics, and console
  outputs.

## Risks
- Medium-low: moving a dense classifier graph can accidentally break helper
  wiring or tests if imports or package-level references drift.

## Rollback Plan
- Move rollout control helpers back into `main.go` and delete the extracted
  helper file.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase136-ownership-rollout-control-helper.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase136-ownership-rollout-control-helper.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a containment refactor that keeps rollout semantics stable while
  reducing control-plane density in the monolithic entrypoint.

## Implementation Notes
- Added `cmd/monolithic/chainpulse/ownership_rollout_control.go` to host
  rollout control types, classifier helpers, policy normalization, and code
  mapping functions.
- Reduced `cmd/monolithic/chainpulse/main.go` to orchestration-oriented logic
  by removing inline rollout control primitives.
- Kept `cmd/monolithic/chainpulse/ownership_rollout_summary.go` as the shared
  assembly layer above the extracted control helpers.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase136-ownership-rollout-control-helper.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
