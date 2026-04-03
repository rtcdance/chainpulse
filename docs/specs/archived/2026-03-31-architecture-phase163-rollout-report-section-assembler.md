# Phase 163 Rollout Report Section Assembler

## Title
Phase 163 - Add rollout report section assembler for monolithic body building

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
- `cmd/monolithic/chainpulse/ownership_rollout_report_sections.go`
- `cmd/monolithic/chainpulse/ownership_rollout_report_body.go`
- `cmd/monolithic/chainpulse/ownership_rollout_report_body_test.go`
- `docs/specs/2026-03-31-architecture-phase162-rollout-report-body-sections.md`

## Context
Phase 162 split the monolithic rollout report body builder into `surface`,
`approval`, and `guarded-cutover` sections. The top-level body builder still
calls those helpers directly rather than going through a single section
assembler.

## Problem Statement
Without a section assembler, rollout report body building still lacks the same
`section builder -> assembler -> entrypoint` structure already used in summary
and presenter assembly.

## Scope
- Add a rollout report section assembler for monolithic body building.
- Add tests for section assembly and application.
- Keep the external `/health/rollout` contract unchanged.

## Non-Goals
- No rollout logic changes.
- No schema changes.
- No handler or route changes.

## Selected Approach
- Introduce `buildOwnershipRolloutReportSections(...)` as an assembler over the
  existing report body sections.
- Keep `buildOwnershipRolloutReportBody(...)` as the top-level entrypoint that
  applies assembled sections to the typed report details.

## Data / Contract Impact
- No external JSON changes.
- Internal report assembly becomes more structurally aligned with the rest of
  the rollout control stack.

## Observability
- Preserves report semantics while making future producer reuse and extension
  more systematic.

## Risks
- Low: structural refactor risk only, mitigated by focused tests and existing
  package gates.

## Rollback Plan
- Inline the section assembly back into the body builder without changing the
  report contract.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase163-rollout-report-section-assembler.md`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase163-rollout-report-section-assembler.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the next structural alignment step for rollout report body
  assembly.

## Implementation Notes
- Added `ownershipRolloutReportSections` and its assembler helper.
- Updated the body builder to consume assembled sections.
- Added assembler/application tests.

## Verification Results
- `scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase163-rollout-report-section-assembler.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `go test ./pkg/plugins/api/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
