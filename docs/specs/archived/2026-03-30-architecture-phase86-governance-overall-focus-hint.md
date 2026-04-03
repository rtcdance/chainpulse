# Phase 86 Governance Overall Focus Hint

## Title
Phase 86 - Add aggregate focus hint to governance overview

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
- `docs/specs/2026-03-30-architecture-phase86-governance-overall-focus-hint.md`

## Context
Phase 85 added `Overall Hint`, but it still does not identify which governance
surface should be inspected first when health is degraded.

## Problem Statement
Without an aggregate focus hint, operators still need to scan the table to find
the highest-priority row after seeing a degraded overall health state.

## Scope
- Add `Overall Focus` line below `Overall Hint`.
- Derive focus from the highest-priority degraded row:
  - `fail` rows first
  - then `warn` rows
  - otherwise `none`

## Non-Goals
- No changes to per-row `Action` logic.
- No changes to per-row severity mapping.

## Options Considered
- Option A: keep only aggregate health and hint.
- Option B: add one aggregate focus target.

## Selected Approach
Choose Option B to make the overview immediately actionable when degraded.

## Data / Contract Impact
- `Governance Overview` gains one aggregate focus line.
- Existing table structure remains unchanged.

## Risks
- Minimal; focus-priority rules may need future refinement.

## Rollback Plan
- Remove aggregate focus line from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase86-governance-overall-focus-hint.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase86-governance-overall-focus-hint.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve degraded-state triage in the CI overview.

## Implementation Summary
- Added an `Overall Focus` line derived from the highest-priority degraded row.
- Updated overview regression test coverage and operations/architecture docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase86-governance-overall-focus-hint.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
