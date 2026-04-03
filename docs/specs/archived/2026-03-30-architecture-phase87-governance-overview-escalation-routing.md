# Phase 87 Governance Overview Escalation Routing

## Title
Phase 87 - Add escalation routing hints to governance overview

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
- `docs/specs/2026-03-30-architecture-phase87-governance-overview-escalation-routing.md`

## Context
Phase 86 added `Overall Focus`, which identifies the first degraded governance
surface, but it still does not tell operators which likely responsibility domain
or operator group should be engaged first.

## Problem Statement
When governance health degrades, responders can identify the affected surface
but still need to infer the most likely escalation target from repository
context and report names.

## Scope
- Add a per-row `Route` column to `Governance Overview`.
- Add a top-level `Overall Route` line derived from `Overall Focus`.
- Keep routing hints lightweight and report-domain-oriented.

## Non-Goals
- No changes to severity mapping or `Overall Focus` priority rules.
- No attempt to encode paging policies or external on-call integrations.

## Options Considered
- Option A: keep focus-only hints and rely on humans to infer ownership.
- Option B: add lightweight route hints tied to each governance surface.

## Selected Approach
Choose Option B so the CI overview can immediately suggest the most likely
operator group or report domain without introducing external dependencies.

## Data / Contract Impact
- `Governance Overview` gains one `Route` table column.
- `Governance Overview` gains one `Overall Route` aggregate line.

## Risks
- Routing hints may need adjustment if team ownership changes.

## Rollback Plan
- Remove route metadata from the overview and revert to focus-only behavior.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase87-governance-overview-escalation-routing.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase87-governance-overview-escalation-routing.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve degraded-state escalation clarity in the CI overview.

## Implementation Summary
- Added per-surface route hints to the overview table.
- Added `Overall Route` derived from the highest-priority degraded surface.
- Updated regression coverage and supporting operations/architecture docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase87-governance-overview-escalation-routing.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
