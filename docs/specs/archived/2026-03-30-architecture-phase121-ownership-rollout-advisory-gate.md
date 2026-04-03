# Phase 121 Ownership Rollout Advisory Gate

## Title
Phase 121 - Add a shared advisory gate helper for monolithic ownership rollout

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
- `docs/specs/2026-03-30-architecture-phase120-ownership-rollout-readiness-surface.md`

## Context
Phase 120 added rollout-readiness details to `/health/ready`, but the decision
semantics still live inline inside the response builder.

## Problem Statement
Without a shared advisory helper, future rollout gates or operator automation
would need to re-encode the same ownership progression rules in multiple places.

## Scope
- Introduce a shared monolithic ownership rollout advisory helper.
- Normalize advisory output to stable decision levels:
  - `allow`
  - `hold`
  - `unknown`
- Reuse the helper in ownership rollout readiness details.
- Add focused tests for `allow`, `hold`, and `unknown` cases.

## Non-Goals
- No enforcement or automatic cutover.
- No change to readiness HTTP status semantics.
- No microservice rollout changes.

## Selected Approach
- Keep the helper local to monolithic ownership aggregation code for now.
- Derive advisory decision from the existing ownership summary and mode:
  - `runtime-owned` -> `allow`
  - `shadow`, `legacy-only`, `idle` -> `hold`
  - malformed/ambiguous ownership totals -> `unknown`
- Continue exposing human-readable reasons alongside the decision.

## Data / Contract Impact
- Ownership readiness details expand with:
  - `rollout_gate_decision`
  - `rollout_gate_reason`

## Observability
- Rollout readiness now exposes both descriptive status and a normalized
  advisory decision suitable for future policy hooks.

## Risks
- Low risk; additive metadata only.

## Rollback Plan
- Remove the shared advisory helper and the added readiness detail fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase121-ownership-rollout-advisory-gate.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase121-ownership-rollout-advisory-gate.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as a preparation step for later operator-facing or policy-driven
  ownership progression gates.

## Implementation Summary
- Added a shared `ownershipRolloutAdvisory` helper in monolithic ownership
  aggregation code.
- Normalized rollout decision semantics to:
  - `allow`
  - `hold`
  - `unknown`
- Reused the helper in ownership readiness details and exposed:
  - `rollout_gate_decision`
  - `rollout_gate_reason`
- Added focused tests for `allow`, `hold`, and `unknown` advisory paths.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase121-ownership-rollout-advisory-gate.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
