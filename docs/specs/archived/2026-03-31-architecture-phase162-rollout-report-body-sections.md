# Phase 162 Rollout Report Body Sections

## Title
Phase 162 - Split monolithic rollout report body builder into sections

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
- `docs/specs/2026-03-31-architecture-phase161-rollout-report-body-builder.md`

## Context
Phase 161 extracted a monolithic rollout report body builder, but the helper
still populates all report fields in one long function. The report body already
has natural section boundaries that match the broader rollout summary model.

## Problem Statement
Without section boundaries, future rollout report producers will still face a
larger-than-necessary integration surface when they only need to reuse or adapt
part of the monolithic body builder.

## Scope
- Split the monolithic rollout report body builder into section helpers:
  - surface
  - approval
  - guarded-cutover
- Keep the external `/health/rollout` contract unchanged.

## Non-Goals
- No rollout decision changes.
- No payload schema changes.
- No handler, readiness, or route changes.

## Selected Approach
- Keep `buildOwnershipRolloutReportBody(...)` as the top-level entrypoint.
- Delegate section population to smaller helpers aligned with existing rollout
  summary decomposition.

## Data / Contract Impact
- No external JSON changes.
- Internal report assembly gains finer-grained reuse boundaries.

## Observability
- Preserves existing report semantics while improving maintainability and future
  producer reuse.

## Risks
- Low: structural refactor risk only, mitigated by helper-level tests.

## Rollback Plan
- Collapse the section helpers back into the single body builder if needed,
  without changing the report contract.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase162-rollout-report-body-sections.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase162-rollout-report-body-sections.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the next step in aligning rollout report assembly with the
  `surface / approval / guarded-cutover` decomposition already used elsewhere.

## Implementation Notes
- Split the body builder into `surface`, `approval`, and `guarded` helpers.
- Added focused section-level tests while preserving the top-level body builder.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase162-rollout-report-body-sections.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
