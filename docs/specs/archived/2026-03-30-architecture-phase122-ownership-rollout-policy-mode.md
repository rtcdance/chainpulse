# Phase 122 Ownership Rollout Policy Mode

## Title
Phase 122 - Add report-only policy mode for monolithic ownership rollout advisory

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
- `docs/specs/2026-03-30-architecture-phase121-ownership-rollout-advisory-gate.md`

## Context
Phase 121 introduced normalized advisory semantics for ownership rollout, but
there is still no explicit policy mode that says how those advisory decisions
should be interpreted operationally.

## Problem Statement
Without a rollout policy mode, operators can see `allow|hold|unknown` but still
cannot tell whether the service is only reporting advice or whether a future
step may begin enforcing it.

## Scope
- Add a monolithic ownership rollout policy mode resolver.
- Support safe default `report-only`.
- Expose policy metadata in rollout readiness details, including:
  - `rollout_policy_mode`
  - `rollout_policy_action`
  - `rollout_policy_reason`
- Keep behavior non-blocking in this phase.
- Add focused tests for:
  - default policy mode
  - explicit env override normalization
  - readiness detail policy metadata

## Non-Goals
- No enforced blocking behavior.
- No ownership cutover automation.
- No microservice policy wiring.

## Selected Approach
- Resolve rollout policy mode from environment with safe fallback to
  `report-only`.
- Map current advisory decisions to report-only actions:
  - `allow` -> `report-allow`
  - `hold` -> `report-hold`
  - `unknown` -> `report-unknown`
- Keep the policy helper local to monolithic rollout aggregation for now.

## Data / Contract Impact
- Rollout readiness details expand with policy metadata:
  - `rollout_policy_mode`
  - `rollout_policy_action`
  - `rollout_policy_reason`

## Observability
- Readiness details now communicate both:
  - the advisory decision
  - how the running service is currently treating that decision

## Risks
- Low risk; additive metadata only.

## Rollback Plan
- Remove rollout policy mode resolver and related readiness detail fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase122-ownership-rollout-policy-mode.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase122-ownership-rollout-policy-mode.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a safe transition step from passive advisory reporting toward
  future guarded rollout policy.

## Implementation Summary
- Added a rollout policy resolver with safe default `report-only`.
- Ownership readiness details now expose:
  - `rollout_policy_mode`
  - `rollout_policy_action`
  - `rollout_policy_reason`
- Report-only policy currently maps advisory decisions to:
  - `allow -> report-allow`
  - `hold -> report-hold`
  - `unknown -> report-unknown`
- Added focused tests for default normalization, env override normalization,
  and readiness-detail policy metadata.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase122-ownership-rollout-policy-mode.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
