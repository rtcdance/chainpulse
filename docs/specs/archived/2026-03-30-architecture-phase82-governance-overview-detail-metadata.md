# Phase 82 Governance Overview Detail Metadata

## Title
Phase 82 - Add detail metadata to governance overview rows

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
- `docs/specs/2026-03-30-architecture-phase82-governance-overview-detail-metadata.md`

## Context
Phase 81 added primary source metadata to overview rows, but regression and
secondary-detail artifacts are still implicit.

## Problem Statement
Without detail metadata, operators still need to guess which delta or
supplementary file to inspect after opening the primary artifact.

## Scope
- Add a `Details` column to `Governance Overview`.
- Populate each row with its secondary/detail artifact where applicable.
- Keep the overview compact and readable.

## Non-Goals
- No clickable link rendering in this phase.
- No changes to artifact generation.

## Options Considered
- Option A: keep only the primary source artifact.
- Option B: add a secondary/details artifact hint for each row.

## Selected Approach
Choose Option B to further reduce investigation friction from the top-level
overview.

## Data / Contract Impact
- `Governance Overview` table gains a `Details` column.
- Existing artifacts remain unchanged.

## Risks
- Low; summary rendering only.

## Rollback Plan
- Remove the `Details` column from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase82-governance-overview-detail-metadata.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase82-governance-overview-detail-metadata.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to provide direct detail-artifact hints from the overview.

## Implementation Summary
- Added a `Details` column to `Governance Overview`.
- Populated each overview row with a delta or supplementary artifact hint.
- Extended overview regression test coverage and updated operations/architecture
  docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase82-governance-overview-detail-metadata.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
