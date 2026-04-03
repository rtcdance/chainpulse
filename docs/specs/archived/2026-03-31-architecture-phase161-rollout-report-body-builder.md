# Phase 161 Rollout Report Body Builder

## Title
Phase 161 - Extract monolithic rollout report body builder

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
- `cmd/monolithic/chainpulse/ownership_rollout_report_body.go`
- `cmd/monolithic/chainpulse/ownership_rollout_report_body_test.go`
- `cmd/monolithic/chainpulse/ownership_rollout_summary.go`
- `docs/specs/2026-03-31-architecture-phase160-rollout-report-metadata-builder.md`

## Context
Phase 160 extracted a shared rollout report metadata builder. The monolithic
rollout report assembly still fills the full business body inline inside
`ownership_rollout_summary.go`.

## Problem Statement
Without a dedicated body builder, the rollout report assembly path still mixes
summary orchestration with contract population, which makes future reuse by
microservice producers more awkward.

## Scope
- Extract monolithic rollout report body population into a dedicated helper.
- Add a focused helper-level test for the report body population.
- Keep the external `/health/rollout` contract unchanged.

## Non-Goals
- No rollout logic changes.
- No report schema changes.
- No health/readiness or route changes.

## Selected Approach
- Introduce `buildOwnershipRolloutReportBody(...)` to populate the typed report
  body after metadata construction.
- Keep `reportDetails()` as a small orchestration method that combines metadata
  and body building.

## Data / Contract Impact
- No external JSON changes.
- Internal monolithic report assembly becomes more modular.

## Observability
- Keeps rollout report assembly easier to evolve while preserving current
  telemetry/report semantics.

## Risks
- Low: structural refactor risk only, mitigated by focused helper tests and
  existing package gates.

## Rollback Plan
- Move the body population logic back into `ownership_rollout_summary.go`
  without changing the typed contract.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase161-rollout-report-body-builder.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase161-rollout-report-body-builder.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the next step toward a cleaner `metadata builder + body builder`
  rollout report assembly model.

## Implementation Notes
- Added `buildOwnershipRolloutReportBody(...)`.
- Added helper-level tests for rollout report body population.
- Reduced `reportDetails()` to metadata builder plus body builder orchestration.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase161-rollout-report-body-builder.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
