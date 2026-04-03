# Phase 90 Governance Overview Severity-Aware Next Step

## Title
Phase 90 - Refine aggregate next-step wording by severity

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
- `docs/specs/2026-03-30-architecture-phase90-governance-overview-severity-aware-next-step.md`

## Context
Phase 89 added `Overall Next Step`, but the current wording is still fairly
uniform and does not clearly distinguish hard failures from softer drift states.

## Problem Statement
Operators benefit from stronger urgency cues when governance health is `fail`
and gentler review-oriented wording when health is only `warn`.

## Scope
- Refine `Overall Next Step` wording based on aggregate severity.
- Keep existing routing and playbook derivation unchanged.
- Preserve the same top-level field name.

## Non-Goals
- No changes to per-row `Action`, `Route`, or `Playbook` logic.
- No changes to severity normalization or focus prioritization.

## Options Considered
- Option A: keep one generic wording template.
- Option B: tailor aggregate wording for `fail`, `warn`, `ok`, and `info`.

## Selected Approach
Choose Option B so the top-level guidance communicates urgency more clearly
without changing overview structure.

## Data / Contract Impact
- `Overall Next Step` wording changes for degraded states.
- No table schema changes.

## Risks
- Minimal; wording may need future tuning as more governance surfaces are added.

## Rollback Plan
- Revert to the previous generic `Overall Next Step` phrasing.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase90-governance-overview-severity-aware-next-step.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase90-governance-overview-severity-aware-next-step.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve urgency signaling in the aggregate governance overview.

## Implementation Summary
- Refined aggregate `Overall Next Step` phrasing for `fail` and `warn` states.
- Updated regression coverage and supporting docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase90-governance-overview-severity-aware-next-step.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
