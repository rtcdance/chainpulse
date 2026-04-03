# Phase 88 Governance Overview Playbook Routing

## Title
Phase 88 - Add playbook hints to governance overview

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
- `docs/specs/2026-03-30-architecture-phase88-governance-overview-playbook-routing.md`

## Context
Phase 87 added ownership-oriented route hints, but degraded governance states
still require operators to infer which local playbook or operational document
should be opened first.

## Problem Statement
Knowing the likely owner domain shortens escalation, but it does not fully
remove ambiguity around the first artifact to consult during triage.

## Scope
- Add a per-row `Playbook` column to `Governance Overview`.
- Add a top-level `Overall Playbook` line derived from `Overall Focus`.
- Keep playbook hints local to repository documentation or generated artifacts.

## Non-Goals
- No changes to severity mapping, route mapping, or focus-priority rules.
- No external ticketing, paging, or runbook system integration.

## Options Considered
- Option A: rely on `Action` plus `Route`.
- Option B: add explicit playbook/doc routing.

## Selected Approach
Choose Option B so degraded states can immediately point responders to both the
likely owner domain and the most useful first document.

## Data / Contract Impact
- `Governance Overview` gains one `Playbook` table column.
- `Governance Overview` gains one `Overall Playbook` aggregate line.

## Risks
- Playbook pointers may need periodic updates as docs evolve.

## Rollback Plan
- Remove playbook metadata and keep route/action-only guidance.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase88-governance-overview-playbook-routing.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase88-governance-overview-playbook-routing.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make degraded-state triage more operationally direct in CI.

## Implementation Summary
- Added per-surface playbook hints to the overview table.
- Added `Overall Playbook` derived from the highest-priority degraded surface.
- Updated regression coverage and supporting docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase88-governance-overview-playbook-routing.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
