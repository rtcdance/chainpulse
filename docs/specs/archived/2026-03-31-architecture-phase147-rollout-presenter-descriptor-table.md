# Phase 147 Rollout Presenter Descriptor Table

## Title
Phase 147 - Convert rollout presenter labels into a descriptor table

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
2026-03-31

## Related Modules
- `cmd/monolithic/chainpulse/ownership_rollout_presenter.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase146-rollout-presenter-helper.md`

## Context
Phase 146 extracted lifecycle rollout presentation into a dedicated presenter
helper. The helper still formats console output through a long sequence of
manual `fmt.Fprintf` calls.

## Problem Statement
Manual startup/shutdown formatting keeps the presenter more repetitive than it
needs to be and makes future rollout field additions easier to miss or format
inconsistently across lifecycles.

## Scope
- Convert rollout presenter console lines into a descriptor table.
- Preserve current running/shutdown wording, casing, and value formatting.
- Add focused tests for descriptor label stability.

## Non-Goals
- No rollout behavior changes.
- No readiness or metric contract changes.
- No new rollout fields.

## Selected Approach
- Introduce a `ownershipRolloutPresenterLine` descriptor table with separate
  running/shutdown labels and a value renderer.
- Keep `printOwnershipRolloutSummary(...)` as the single public presenter entry
  and make it iterate the descriptors.

## Data / Contract Impact
- No external contract change intended.
- Console summary wording remains stable.

## Observability
- Refactor only; console rollout presentation remains semantically unchanged.

## Risks
- Medium-low: a descriptor entry could use the wrong label casing or wrong
  value accessor, causing console output drift.

## Rollback Plan
- Inline the descriptor table back into `printOwnershipRolloutSummary(...)` and
  remove the helper functions.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase147-rollout-presenter-descriptor-table.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase147-rollout-presenter-descriptor-table.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk presenter cleanup that reduces repetition without
  changing rollout semantics.

## Implementation Notes
- Converted console presenter lines into a descriptor table with lifecycle-aware
  labels and value renderers.
- Added focused tests for descriptor boundary labels and lifecycle prefix
  behavior.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase147-rollout-presenter-descriptor-table.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
