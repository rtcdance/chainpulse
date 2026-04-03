# Phase 146 Rollout Presenter Helper

## Title
Phase 146 - Extract rollout presenter helper for monolithic ownership rollout output

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
- `cmd/monolithic/chainpulse/ownership_rollout_presenter.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase145-rollout-section-assembler.md`

## Context
After Phases 135-145, ownership rollout classification and summary assembly are
mostly modularized. The monolithic entrypoint still repeats large startup and
shutdown blocks for rollout logging and console presentation.

## Problem Statement
Keeping rollout presentation inline in `main.go` makes the entrypoint denser
than necessary and increases the chance of startup and shutdown output drifting
apart during future changes.

## Scope
- Extract rollout lifecycle logging into a dedicated presenter helper.
- Extract rollout console summary formatting into the same presenter layer.
- Keep existing log messages and console output wording stable.

## Non-Goals
- No new rollout states or policy behavior.
- No change to readiness or metric contracts.
- No execution gating.

## Selected Approach
- Add a dedicated `ownership_rollout_presenter.go` helper with one function for
  structured lifecycle logs and one function for console summary printing.
- Update `main.go` to delegate startup and shutdown rollout presentation to the
  helper.

## Data / Contract Impact
- No external contract change intended.
- Existing log event names and console summary wording remain stable.

## Observability
- Refactor only; rollout console and structured log surfaces must remain
  behaviorally equivalent.

## Risks
- Medium-low: startup/shutdown wording or capitalization could drift during the
  extraction if the presenter helper is not kept exact.

## Rollback Plan
- Inline rollout lifecycle logging and console formatting back into `main.go`
  and remove the presenter helper file.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase146-rollout-presenter-helper.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase146-rollout-presenter-helper.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a presentation-layer containment refactor that keeps rollout
  behavior stable while reducing `main.go` density.

## Implementation Notes
- Added `cmd/monolithic/chainpulse/ownership_rollout_presenter.go` for rollout
  lifecycle logs and console summary printing.
- Simplified `cmd/monolithic/chainpulse/main.go` by replacing repeated startup
  and shutdown rollout output with presenter helper calls.
- Added presenter-focused coverage in `cmd/monolithic/chainpulse/main_test.go`.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase146-rollout-presenter-helper.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
