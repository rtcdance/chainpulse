# Phase 85 Governance Overall Health Hint

## Title
Phase 85 - Add aggregate hint for governance overall health

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
- `docs/specs/2026-03-30-architecture-phase85-governance-overall-health-hint.md`

## Context
Phase 84 added aggregate overall health, but operators still need to translate
that status into an immediate next-step mindset.

## Problem Statement
Without a short aggregate hint, the overview still requires a small mental step
to decide whether the situation is stable, warning-only, or failure-first.

## Scope
- Add a short `Overall Hint` line below `Overall Health`.
- Derive the hint from aggregate health:
  - `ok`
  - `warn`
  - `fail`
  - `info`

## Non-Goals
- No changes to per-row action hints.
- No changes to aggregate health computation.

## Options Considered
- Option A: keep aggregate health only.
- Option B: add a short aggregate interpretation hint.

## Selected Approach
Choose Option B to make the overview more immediately actionable without adding
verbosity.

## Data / Contract Impact
- `Governance Overview` gains one aggregate hint line.
- Existing table shape remains unchanged.

## Risks
- Minimal; summary rendering only.

## Rollback Plan
- Remove the aggregate hint line from overview rendering.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase85-governance-overall-health-hint.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase85-governance-overall-health-hint.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve top-level readability of governance health.

## Implementation Summary
- Added an `Overall Hint` line derived from aggregate governance health.
- Updated overview regression test coverage and operations/architecture docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase85-governance-overall-health-hint.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
