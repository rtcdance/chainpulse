# Phase 79 Governance Overview Status Normalization

## Title
Phase 79 - Normalize governance overview row severity

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
- `docs/specs/2026-03-30-architecture-phase79-governance-overview-status-normalization.md`

## Context
Phase 78 expanded the overview to multiple governance surfaces, but row status
values still mix raw domain terms such as `pass`, `healthy`, `not_applicable`,
and `fail`.

## Problem Statement
Without normalized severity labels, the overview is harder to scan because each
row uses different status semantics.

## Scope
- Add normalized severity mapping to governance overview rows.
- Keep raw status and regression values visible.
- Use a compact severity scale suitable for future CI styling:
  - `ok`
  - `warn`
  - `fail`
  - `info`

## Non-Goals
- No changes to underlying governance artifact formats.
- No UI coloring implementation in this phase.

## Options Considered
- Option A: keep raw status values only.
- Option B: add normalized overview severity alongside raw values.

## Selected Approach
Choose Option B for clearer triage and future presentation flexibility.

## Data / Contract Impact
- `Governance Overview` gains a normalized `Level` column.
- Existing detailed artifacts remain unchanged.

## Risks
- Low; mapping mistakes could misclassify overview rows.

## Rollback Plan
- Remove normalized severity mapping and revert to raw status-only overview.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase79-governance-overview-status-normalization.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
- `./scripts/compare-ticket-registry-health.sh`
- `./scripts/export-migration-owner-drift-report.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase79-governance-overview-status-normalization.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make overview rows more comparable without changing source
  reports.

## Implementation Summary
- Extended `scripts/append-governance-overview-summary.sh` with normalized
  severity mapping for overview rows.
- Updated overview output to include a `Level` column alongside raw status and
  regression values.
- Extended overview regression test coverage and updated operations/architecture
  docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase79-governance-overview-status-normalization.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
- `./scripts/compare-ticket-registry-health.sh`
- `./scripts/export-migration-owner-drift-report.sh`
