# Phase 150 Rollout Presenter Sections

## Title
Phase 150 - Split rollout presenter descriptors into ownership, approval, and guarded sections

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
- `docs/specs/2026-03-31-architecture-phase149-rollout-presenter-shared-accessors.md`

## Context
After Phases 147-149, rollout presentation is descriptor-driven and uses shared
accessors, but console and log descriptors still live in one large flat list.

## Problem Statement
Keeping all presenter descriptors flat makes it harder to visually align the
presentation layer with the existing ownership, approval, and guarded-cutover
summary decomposition used elsewhere in the monolith rollout control plane.

## Scope
- Split presenter descriptors into ownership, approval, and guarded-cutover
  section builders.
- Preserve descriptor ordering in the final assembled presenter surfaces.
- Add focused tests for section boundaries.

## Non-Goals
- No rollout behavior changes.
- No readiness or metric contract changes.
- No output wording changes.

## Selected Approach
- Introduce section-level builder helpers for console presenter lines and
  structured log descriptors.
- Keep the top-level presenter builders as simple assemblers that concatenate
  the three sections in the existing order.

## Data / Contract Impact
- No external contract change intended.
- Presenter output order and wording remain stable.

## Observability
- Refactor only; rollout console and log semantics remain unchanged.

## Risks
- Low: a section could accidentally drop one descriptor or change descriptor
  ordering.

## Rollback Plan
- Collapse the section helpers back into the previous flat presenter builders.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase150-rollout-presenter-sections.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase150-rollout-presenter-sections.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a structural cleanup that aligns the presenter layer with the
  rollout summary's ownership/approval/guarded decomposition.

## Implementation Notes
- Split presenter console descriptors into ownership, approval, and guarded
  section helpers.
- Split lifecycle log descriptors into the same three section helpers.
- Added focused tests for section sizes and boundary labels/messages.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase150-rollout-presenter-sections.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
