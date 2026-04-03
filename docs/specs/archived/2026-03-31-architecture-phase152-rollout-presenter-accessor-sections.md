# Phase 152 Rollout Presenter Accessor Sections

## Title
Phase 152 - Group rollout presenter accessors by ownership, approval, and guarded sections

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
- `docs/specs/2026-03-31-architecture-phase151-rollout-presenter-section-assembler.md`

## Context
Phase 151 aligned the presenter layer with the summary layer by introducing
section builders and section assemblers. Shared presenter accessors still lived
as a flat group of functions.

## Problem Statement
Flat accessors leave the presenter internals slightly out of sync with the
section-oriented ownership/approval/guarded structure already used by the
descriptor builders and assemblers.

## Scope
- Add section-grouped presenter accessor assemblers.
- Rewire presenter section builders to consume grouped accessors.
- Keep output wording and ordering unchanged.

## Non-Goals
- No rollout behavior changes.
- No readiness, metrics, log schema, or console wording changes.

## Selected Approach
- Introduce grouped accessor structs for ownership, approval, and
  guarded-cutover fields plus a top-level accessor assembler.
- Reuse the existing value functions underneath the accessor groups to keep the
  refactor low risk.

## Data / Contract Impact
- No external contract change intended.
- Presenter output remains stable.

## Observability
- Refactor only; rollout console and structured log semantics remain unchanged.

## Risks
- Low: a section builder could be wired to the wrong accessor group.

## Rollback Plan
- Revert section builders to reference the flat accessors directly and remove
  the grouped accessor assembler types.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase152-rollout-presenter-accessor-sections.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase152-rollout-presenter-accessor-sections.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a final internal alignment pass so presenter accessors mirror the
  same section boundaries already used elsewhere in rollout presentation.

## Implementation Notes
- Added grouped presenter accessor structs and a top-level accessor assembler.
- Rewired section-specific presenter builders to consume their corresponding
  accessor groups.
- Added focused tests to ensure grouped accessors are assembled for all
  sections.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase152-rollout-presenter-accessor-sections.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
