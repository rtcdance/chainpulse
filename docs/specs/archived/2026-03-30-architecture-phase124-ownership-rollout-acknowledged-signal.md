# Phase 124 Ownership Rollout Acknowledged Signal

## Title
Phase 124 - Add acknowledged signal for monolithic ownership rollout manual gate

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
- `docs/specs/2026-03-30-architecture-phase123-ownership-rollout-manual-gate-mode.md`

## Context
Phase 123 introduced `manual-gate` policy mode, but the system still cannot
express whether an operator has already reviewed and acknowledged the current
ownership rollout posture.

## Problem Statement
Without an explicit acknowledged signal, `manual-gate` can say "review needed"
but cannot distinguish pending review from acknowledged review.

## Scope
- Add an environment-driven acknowledged signal for ownership rollout policy.
- Keep runtime behavior non-blocking.
- Expose policy acknowledgment metadata through readiness details, including:
  - `rollout_policy_acknowledged`
  - `rollout_policy_ack_state`
- Add focused tests for:
  - acknowledgment env normalization
  - manual-gate acknowledged vs pending actions

## Non-Goals
- No persistent approval store.
- No automatic enforcement.
- No microservice rollout wiring.

## Selected Approach
- Resolve acknowledgment from an env flag with safe default `false`.
- Keep `report-only` unchanged.
- In `manual-gate`, map actions based on advisory decision and acknowledgment:
  - pending review -> `manual-review-*`
  - acknowledged review -> `manual-acknowledged-*`

## Data / Contract Impact
- Rollout readiness details expand with:
  - `rollout_policy_acknowledged`
  - `rollout_policy_ack_state`

## Observability
- Operators can now tell whether manual-gate is still awaiting human review or
  has already been explicitly acknowledged.

## Risks
- Low risk; additive metadata only.

## Rollback Plan
- Remove acknowledgment resolver and related readiness detail fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase124-ownership-rollout-acknowledged-signal.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase124-ownership-rollout-acknowledged-signal.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a non-blocking acknowledgment input that prepares the path for
  future operator-confirmed rollout controls.

## Implementation Summary
- Added env-driven acknowledgment input:
  - `CHAINPULSE_OWNERSHIP_ROLLOUT_ACKNOWLEDGED`
- Ownership readiness details now expose:
  - `rollout_policy_acknowledged`
  - `rollout_policy_ack_state`
- `manual-gate` now distinguishes:
  - pending review -> `manual-review-*`
  - acknowledged review -> `manual-acknowledged-*`
- Added focused tests for acknowledgment env parsing and acknowledged
  manual-gate action mapping.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase124-ownership-rollout-acknowledged-signal.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
