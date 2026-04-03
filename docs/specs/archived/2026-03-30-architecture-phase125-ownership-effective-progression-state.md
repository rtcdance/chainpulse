# Phase 125 Ownership Effective Progression State

## Title
Phase 125 - Add effective progression state for monolithic ownership rollout

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
- `docs/specs/2026-03-30-architecture-phase124-ownership-rollout-acknowledged-signal.md`

## Context
Phase 124 completed the advisory, policy-mode, and acknowledgment layers, but
downstream consumers still need to interpret several fields together.

## Problem Statement
Without a single effective progression state, dashboards and future cutover
logic must repeatedly combine advisory decision, policy mode, and ack state.

## Scope
- Add a shared effective progression state helper.
- Normalize effective state to stable values:
  - `observe`
  - `review-required`
  - `acknowledged`
  - `ready-for-cutover`
  - `unknown`
- Expose progression state metadata through readiness details.
- Add focused tests for each normalized effective state.

## Non-Goals
- No enforced cutover.
- No automatic ownership switching.
- No microservice rollout integration.

## Selected Approach
- Derive effective state from:
  - advisory decision
  - policy mode
  - acknowledgment state
- Keep helper local to monolithic rollout aggregation until shared by another
  runtime surface.

## Data / Contract Impact
- Rollout readiness details expand with:
  - `rollout_effective_state`
  - `rollout_effective_reason`

## Observability
- The rollout control chain now yields a single derived state suitable for:
  - dashboards
  - operator review
  - future guarded cutover logic

## Risks
- Low risk; additive metadata only.

## Rollback Plan
- Remove effective progression state helper and related readiness detail fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase125-ownership-effective-progression-state.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase125-ownership-effective-progression-state.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the final non-blocking aggregation layer before future
  operator-driven or enforced cutover controls.

## Implementation Summary
- Added a shared effective progression helper that normalizes rollout control
  state to:
  - `observe`
  - `review-required`
  - `acknowledged`
  - `ready-for-cutover`
  - `unknown`
- Readiness details now expose:
  - `rollout_effective_state`
  - `rollout_effective_reason`
- Added focused tests for all normalized progression states.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase125-ownership-effective-progression-state.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
