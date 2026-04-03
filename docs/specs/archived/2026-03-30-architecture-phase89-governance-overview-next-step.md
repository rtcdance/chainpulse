# Phase 89 Governance Overview Next Step

## Title
Phase 89 - Add aggregate next-step guidance to governance overview

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
- `docs/specs/2026-03-30-architecture-phase89-governance-overview-next-step.md`

## Context
Phase 88 added `Overall Route` and `Overall Playbook`, but operators still need
to mentally combine focus, owner domain, and playbook into one immediate action.

## Problem Statement
Separate aggregate hints are useful, yet degraded CI states are faster to triage
when the overview also provides a single concrete next-step sentence.

## Scope
- Add a top-level `Overall Next Step` line to `Governance Overview`.
- Derive the line from aggregate health, focus, route, and playbook hints.
- Keep the wording short and operational.

## Non-Goals
- No changes to per-row routing, playbook, or action hints.
- No changes to severity mapping or focus-priority rules.

## Options Considered
- Option A: keep separate aggregate hints only.
- Option B: add one synthesized next-step sentence.

## Selected Approach
Choose Option B so the CI overview can provide a direct first action during
degraded states while preserving the detailed table below.

## Data / Contract Impact
- `Governance Overview` gains one aggregate `Overall Next Step` line.
- Existing row contracts remain unchanged.

## Risks
- Aggregate wording may need tuning if new governance surfaces are added later.

## Rollback Plan
- Remove the aggregate next-step line and retain the separate aggregate hints.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase89-governance-overview-next-step.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase89-governance-overview-next-step.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make degraded-state triage more direct in the CI overview.

## Implementation Summary
- Added `Overall Next Step` derived from aggregate overview metadata.
- Updated regression coverage and supporting docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase89-governance-overview-next-step.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
