# Phase 81 Governance Overview Source Metadata

## Title
Phase 81 - Add source metadata to governance overview rows

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
- `docs/specs/2026-03-30-architecture-phase81-governance-overview-source-metadata.md`

## Context
The governance overview now provides compact status and severity signals, but it
does not directly indicate which artifact backs each top-level row.

## Problem Statement
Without source metadata, operators must infer which generated file to inspect
after spotting an issue in the overview.

## Scope
- Add a `Source` column to `Governance Overview`.
- Populate each row with the primary artifact path backing that row.
- Keep the overview compact and readable.

## Non-Goals
- No clickable link rendering in this phase.
- No changes to underlying artifact generation.

## Options Considered
- Option A: keep overview rows source-agnostic.
- Option B: include explicit source metadata for each row.

## Selected Approach
Choose Option B to reduce investigation friction after top-level issue
detection.

## Data / Contract Impact
- `Governance Overview` table gains a `Source` column.
- Existing detail sections and artifacts remain unchanged.

## Risks
- Low; summary rendering only.

## Rollback Plan
- Remove the `Source` column from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase81-governance-overview-source-metadata.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase81-governance-overview-source-metadata.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make overview rows faster to investigate in CI.

## Implementation Summary
- Added a `Source` column to `Governance Overview`.
- Populated each overview row with the basename of its primary backing
  artifact.
- Extended overview regression test coverage and updated operations/architecture
  docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase81-governance-overview-source-metadata.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
