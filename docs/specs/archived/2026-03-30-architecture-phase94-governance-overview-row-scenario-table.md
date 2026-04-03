# Phase 94 Governance Overview Row Scenario Table

## Title
Phase 94 - Convert overview row assertions to a small scenario table

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
- `scripts/test-append-governance-overview-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase94-governance-overview-row-scenario-table.md`

## Context
Phase 93 extracted assertion helpers, but the primary overview scenario still
enumerates row expectations as long inline markdown strings.

## Problem Statement
Long row assertions make the main scenario section noisy and increase the
effort to add or update one overview row.

## Scope
- Introduce a compact row-scenario helper for the primary overview table.
- Preserve existing row coverage and exact rendered expectations.
- Keep aggregate assertions unchanged.

## Non-Goals
- No changes to overview rendering logic.
- No changes to row content or severity behavior.

## Options Considered
- Option A: keep explicit row assertions.
- Option B: move row checks into a tiny scenario table loop.

## Selected Approach
Choose Option B so the test file remains readable while preserving exact row
string matching.

## Data / Contract Impact
- No output contract changes.
- Test structure only.

## Risks
- Minimal; helper indirection may hide a typo if not kept simple.

## Rollback Plan
- Restore explicit inline row assertions.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase94-governance-overview-row-scenario-table.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase94-governance-overview-row-scenario-table.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to keep overview row coverage maintainable as more governance
  surfaces evolve.

## Implementation Summary
- Converted primary row assertions to a small scenario loop.
- Kept exact row expectations while reducing repeated inline assertions.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase94-governance-overview-row-scenario-table.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
