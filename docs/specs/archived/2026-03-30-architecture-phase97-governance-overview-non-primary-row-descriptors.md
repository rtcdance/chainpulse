# Phase 97 Governance Overview Non-Primary Row Descriptors

## Title
Phase 97 - Add non-primary row descriptor helpers for governance overview tests

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
- `docs/specs/2026-03-30-architecture-phase97-governance-overview-non-primary-row-descriptors.md`

## Context
Phase 96 reorganized the overview regression harness into setup/run/assert
stages, but only the primary overview state has compact row descriptors today.

## Problem Statement
If row-level assertions need to expand into `warn`, `info`, or `ok` states,
the test could regress back to long inline markdown strings unless a matching
descriptor path already exists.

## Scope
- Add descriptor-based row helpers for non-primary overview states.
- Keep existing row coverage unchanged.
- Add light non-primary row checks that prove the descriptors work.

## Non-Goals
- No changes to overview rendering logic.
- No change to the primary scenario row expectations.

## Options Considered
- Option A: wait until non-primary row checks are needed.
- Option B: add lightweight descriptor support now.

## Selected Approach
Choose Option B so future non-primary row coverage can expand without another
round of structural test refactoring.

## Data / Contract Impact
- No output contract changes.
- Test structure only.

## Risks
- Minimal; additional row checks slightly increase test verbosity.

## Rollback Plan
- Remove non-primary row descriptor helpers and keep aggregate-only checks for
  non-primary states.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase97-governance-overview-non-primary-row-descriptors.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase97-governance-overview-non-primary-row-descriptors.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to keep future non-primary row coverage cheap to extend.

## Implementation Summary
- Added reusable row-descriptor assertion helpers for non-primary overview
  states.
- Added representative non-primary row checks without changing output logic.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase97-governance-overview-non-primary-row-descriptors.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
