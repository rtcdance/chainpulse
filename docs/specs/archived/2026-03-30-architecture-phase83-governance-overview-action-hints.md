# Phase 83 Governance Overview Action Hints

## Title
Phase 83 - Add action hints to governance overview rows

## Type
- architecture
- operations

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
- `scripts/append-governance-overview-summary.sh`
- `scripts/test-append-governance-overview-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase83-governance-overview-action-hints.md`

## Context
The overview now exposes status, severity, primary source, and detail artifacts,
but it still leaves the operator to decide which file to inspect first.

## Problem Statement
Without action hints, the overview surfaces facts but does not directly guide
the next troubleshooting step.

## Scope
- Add an `Action` column to `Governance Overview`.
- Provide a short recommended next step per row based on level and available
  artifacts.
- Keep the hints compact and deterministic.

## Non-Goals
- No AI-generated prose or long remediation text.
- No changes to underlying governance data.

## Options Considered
- Option A: keep overview informational only.
- Option B: add short per-row action hints.

## Selected Approach
Choose Option B so the overview not only reports state but also suggests the
next most useful follow-up.

## Data / Contract Impact
- `Governance Overview` gains an `Action` column.
- Existing artifacts remain unchanged.

## Risks
- Low; hint rules may need refinement over time.

## Rollback Plan
- Remove the `Action` column from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase83-governance-overview-action-hints.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase83-governance-overview-action-hints.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve triage guidance in the CI overview.

## Implementation Summary
- Added an `Action` column to `Governance Overview`.
- Populated each row with short next-read guidance derived from overview level
  and available source/detail artifacts.
- Extended overview regression test coverage and updated operations/architecture
  docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase83-governance-overview-action-hints.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
