# Phase 123 Ownership Rollout Manual Gate Mode

## Title
Phase 123 - Add manual-gate policy mode for monolithic ownership rollout

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
- `docs/specs/2026-03-30-architecture-phase122-ownership-rollout-policy-mode.md`

## Context
Phase 122 introduced `report-only` policy mode, but there is still no way to
express a stronger operational stance such as "human approval required before
progressing" while keeping runtime behavior non-blocking.

## Problem Statement
Without a `manual-gate` mode, teams can only see passive reporting and cannot
use readiness details to signal that ownership rollout should pause for human
review even when the service itself remains operational.

## Scope
- Add `manual-gate` as a supported ownership rollout policy mode.
- Keep runtime behavior non-blocking in this phase.
- Expose manual-gate policy actions through rollout readiness details.
- Add focused tests for:
  - policy mode normalization
  - `manual-gate` action mapping for `allow|hold|unknown`

## Non-Goals
- No automatic blocking or enforcement.
- No operator approval store.
- No microservice policy wiring.

## Selected Approach
- Extend the existing rollout policy resolver with `manual-gate`.
- Map advisory decisions to manual-review-oriented actions:
  - `allow` -> `manual-review-allow`
  - `hold` -> `manual-review-hold`
  - `unknown` -> `manual-review-unknown`
- Keep policy reasoning explicit so downstream operators know manual review is
  expected but not yet enforced.

## Data / Contract Impact
- `rollout_policy_mode` may now be `manual-gate`.
- `rollout_policy_action` may now use `manual-review-*` values.

## Observability
- Readiness details can now distinguish:
  - passive reporting
  - human-review-required posture

## Risks
- Low risk; additive metadata only.

## Rollback Plan
- Remove `manual-gate` normalization and action mapping.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase123-ownership-rollout-manual-gate-mode.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase123-ownership-rollout-manual-gate-mode.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a safe intermediate policy mode between passive reporting and
  future guarded enforcement.

## Implementation Summary
- Added `manual-gate` as a supported ownership rollout policy mode.
- `manual-gate` currently remains non-blocking and maps advisory decisions to:
  - `allow -> manual-review-allow`
  - `hold -> manual-review-hold`
  - `unknown -> manual-review-unknown`
- Added focused tests for mode normalization and manual-gate action mapping.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase123-ownership-rollout-manual-gate-mode.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
