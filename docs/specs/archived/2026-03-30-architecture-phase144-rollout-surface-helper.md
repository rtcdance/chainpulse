# Phase 144 Rollout Surface Helper

## Title
Phase 144 - Extract common rollout surface helper from ownership rollout summary

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
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `cmd/monolithic/chainpulse/ownership_rollout_surface.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase143-approval-summary-helper.md`

## Context
After Phases 142-143, approval and guarded-cutover branches are each extracted,
but `ownership_rollout_summary.go` still contains the shared readiness and
metric skeleton for the common rollout fields.

## Problem Statement
Without extracting the common surface helper, `ownership_rollout_summary.go`
still mixes orchestration with the base readiness/metrics plumbing, limiting how
small and focused the file can become.

## Scope
- Extract common rollout readiness and metric emission into a dedicated helper
  file.
- Keep approval and guarded-cutover helper integration unchanged.
- Keep current snapshot fields, readiness keys, and metric names stable.

## Non-Goals
- No new rollout states.
- No behavior change.
- No execution gating.

## Selected Approach
- Introduce a compact helper around the common rollout surface fields.
- Let `ownership_rollout_summary.go` compose:
  - common rollout surface helper
  - approval summary helper
  - guarded-cutover summary helper

## Data / Contract Impact
- No external contract change intended.
- Existing readiness keys and metric names remain stable.

## Observability
- Refactor only; rollout readiness and metrics surfaces must remain unchanged.

## Risks
- Medium-low: refactoring shared surface code could accidentally omit a common
  readiness key or metric if the helper wiring is incomplete.

## Rollback Plan
- Move the common readiness/metric skeleton back into
  `ownership_rollout_summary.go` and remove the helper file.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase144-rollout-surface-helper.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase144-rollout-surface-helper.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a containment refactor to make rollout summary assembly more
  obviously compositional without changing rollout behavior.

## Implementation Notes
- Added `cmd/monolithic/chainpulse/ownership_rollout_surface.go` to
  centralize the common rollout readiness and metric skeleton.
- Simplified `cmd/monolithic/chainpulse/ownership_rollout_summary.go` by
  delegating common surface population to the new helper while keeping approval
  and guarded-cutover helper composition intact.
- Added focused helper coverage in `cmd/monolithic/chainpulse/main_test.go`.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase144-rollout-surface-helper.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
