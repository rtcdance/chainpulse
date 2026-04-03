# Phase 149 Rollout Presenter Shared Accessors

## Title
Phase 149 - Extract shared rollout presenter accessors for console and log symmetry

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
- `docs/specs/2026-03-31-architecture-phase148-rollout-log-descriptor-table.md`

## Context
After Phases 147-148, rollout console and structured log presentation are both
descriptor-driven. The console descriptor table still repeats many inline
value-rendering closures over the same rollout snapshot fields.

## Problem Statement
Leaving repeated field access closures inline increases the number of places
that need to change if rollout snapshot field paths or formatting rules evolve.

## Scope
- Extract shared rollout presenter value accessors.
- Reuse the accessors from descriptor-driven presenter lines.
- Add focused tests for representative accessors.

## Non-Goals
- No rollout behavior changes.
- No readiness or metric contract changes.
- No log message or console wording changes.

## Selected Approach
- Introduce typed accessor helpers for common rollout snapshot fields and use
  them from the console presenter descriptor table.
- Keep the descriptor tables themselves stable so external output stays
  unchanged.

## Data / Contract Impact
- No external contract change intended.
- Presenter output wording remains stable.

## Observability
- Refactor only; rollout console and log semantics remain unchanged.

## Risks
- Low: an accessor could point to the wrong field and skew a presenter line.

## Rollback Plan
- Inline accessors back into the presenter descriptor table and remove the
  shared accessor helpers.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase149-rollout-presenter-shared-accessors.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase149-rollout-presenter-shared-accessors.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a low-risk internal cleanup that reduces presenter field-path
  duplication while keeping rollout surfaces stable.

## Implementation Notes
- Added shared rollout presenter accessors for representative snapshot fields.
- Replaced inline value-rendering closures in the console descriptor table with
  the shared accessors.
- Added focused accessor coverage in `cmd/monolithic/chainpulse/main_test.go`.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase149-rollout-presenter-shared-accessors.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
