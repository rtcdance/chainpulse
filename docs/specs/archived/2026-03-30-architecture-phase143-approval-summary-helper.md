# Phase 143 Approval Summary Helper

## Title
Phase 143 - Extract approval summary helper from ownership rollout summary

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
- `cmd/monolithic/chainpulse/ownership_approval_summary.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase142-guarded-cutover-summary-helper.md`

## Context
Phase 142 extracted the guarded-cutover branch into its own helper, but
`ownership_rollout_summary.go` still carries the full approval summary assembly
logic inline.

## Problem Statement
Without extracting an approval summary helper, rollout summary assembly still
mixes the broader rollout path with a dense approval-oriented branch, reducing
symmetry with the guarded-cutover side and making future changes harder to
contain.

## Scope
- Extract approval summary assembly into a dedicated helper file.
- Centralize:
  - manual approval checkpoint derivation
  - operator handoff derivation
  - approval work item derivation
  - approval checklist derivation
  - approval readiness detail population
  - approval metric emission
- Keep current snapshot fields and external outputs unchanged.

## Non-Goals
- No new approval states.
- No behavior change.
- No execution gating.

## Selected Approach
- Introduce a focused approval summary helper struct with derivation and
  surface methods.
- Preserve the existing snapshot field layout so current call sites remain
  stable.

## Data / Contract Impact
- No external contract change intended.
- Existing approval readiness keys and metric names remain stable.

## Observability
- Refactor only; approval-related observability surfaces must remain unchanged.

## Risks
- Medium-low: approval summary refactoring could accidentally omit one branch of
  approval readiness or metric output if wiring is incomplete.

## Rollback Plan
- Inline approval summary derivation, readiness, and metric code back into
  `ownership_rollout_summary.go` and remove the helper file.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase143-approval-summary-helper.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase143-approval-summary-helper.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a symmetry and containment refactor that keeps approval logic
  cohesive without changing rollout behavior.

## Implementation Notes
- Added `cmd/monolithic/chainpulse/ownership_approval_summary.go` to
  centralize approval summary derivation, readiness detail population, and
  metric emission.
- Simplified `cmd/monolithic/chainpulse/ownership_rollout_summary.go` by
  delegating approval branch assembly to the new helper while keeping snapshot
  fields stable.
- Added focused helper coverage in `cmd/monolithic/chainpulse/main_test.go`.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase143-approval-summary-helper.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
