# Phase 84 Governance Overview Overall Health

## Title
Phase 84 - Add top-level overall health to governance overview

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
- `docs/specs/2026-03-30-architecture-phase84-governance-overview-overall-health.md`

## Context
The governance overview now exposes per-surface level, source, details, and
action metadata, but operators still need to scan the table to infer overall
governance health.

## Problem Statement
Without a top-level summary, CI cannot present one immediate governance health
indicator before the detailed overview table.

## Scope
- Add `Overall Health` summary lines above the governance overview table.
- Derive aggregate health from per-row normalized levels.
- Preserve the detailed per-row overview below the aggregate summary.

## Non-Goals
- No changes to per-row severity mapping rules.
- No changes to detailed artifact generation.

## Options Considered
- Option A: keep only per-row levels.
- Option B: add one aggregate top-level summary.

## Selected Approach
Choose Option B to make CI triage faster while preserving the richer per-row
table for diagnosis.

## Data / Contract Impact
- `Governance Overview` gains aggregate summary lines above the table.
- Existing row structure remains unchanged.

## Risks
- Low; aggregation logic could misrepresent health if mapping is too coarse.

## Rollback Plan
- Remove aggregate overall-health lines from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase84-governance-overview-overall-health.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase84-governance-overview-overall-health.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to give CI a single immediate governance health signal.

## Implementation Summary
- Added aggregate `Overall Health` summary lines above the governance overview
  table.
- Derived aggregate health from per-row normalized levels.
- Extended overview regression test coverage and updated operations/architecture
  docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase84-governance-overview-overall-health.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
