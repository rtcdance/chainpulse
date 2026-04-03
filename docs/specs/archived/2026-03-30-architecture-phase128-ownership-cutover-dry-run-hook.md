# Phase 128 Ownership Cutover Dry-Run Hook

## Title
Phase 128 - Add dry-run guarded cutover hook for monolithic ownership rollout

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
- `docs/specs/2026-03-30-architecture-phase127-ownership-progression-console-summary.md`

## Context
Phase 127 aligned console, health, and metrics around rollout progression, but
the service still does not expose a concrete cutover decision surface.

## Problem Statement
Without a guarded cutover hook, operators can observe rollout posture but
cannot see what the service would decide if cutover gating were enabled.

## Scope
- Add a non-blocking dry-run cutover decision helper.
- Derive a stable dry-run action from effective progression state:
  - `would-allow`
  - `would-hold`
  - `would-unknown`
- Expose dry-run decision metadata through readiness details.
- Add focused tests for dry-run decision mapping.

## Non-Goals
- No enforced cutover.
- No mutation of runtime behavior.
- No microservice integration.

## Selected Approach
- Keep the hook local to monolithic rollout aggregation.
- Reuse effective progression state as the single input.
- Treat only `ready-for-cutover` as `would-allow`.

## Data / Contract Impact
- Readiness details expand with:
  - `rollout_cutover_dry_run_action`
  - `rollout_cutover_dry_run_reason`

## Observability
- Operators can now see both rollout posture and the service's non-binding
  cutover recommendation.

## Risks
- Low risk; additive metadata only.

## Rollback Plan
- Remove dry-run cutover helper and related readiness detail fields.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase128-ownership-cutover-dry-run-hook.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase128-ownership-cutover-dry-run-hook.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first deliberate cutover-decision surface while remaining
  fully dry-run.

## Implementation Summary
- Added a dry-run cutover decision helper based on effective progression state.
- Readiness details now expose:
  - `rollout_cutover_dry_run_action`
  - `rollout_cutover_dry_run_reason`
- Dry-run cutover currently maps:
  - `ready-for-cutover -> would-allow`
  - other known states -> `would-hold`
  - `unknown -> would-unknown`
- Added focused tests for dry-run decision mapping.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase128-ownership-cutover-dry-run-hook.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
