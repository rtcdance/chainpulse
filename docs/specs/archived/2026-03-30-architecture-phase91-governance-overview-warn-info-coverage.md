# Phase 91 Governance Overview Warn/Info Coverage

## Title
Phase 91 - Add warn/info aggregate coverage for governance overview

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
- `docs/specs/2026-03-30-architecture-phase91-governance-overview-warn-info-coverage.md`

## Context
Phase 90 refined `Overall Next Step` wording by severity, but the regression
coverage still exercises only the aggregate failure path in detail.

## Problem Statement
Without dedicated warn/info coverage, the softer aggregate states could regress
silently even though they are now part of the operator-facing CI guidance.

## Scope
- Extend overview regression tests with a `warn` aggregate scenario.
- Extend overview regression tests with an `info` aggregate scenario.
- Assert `Overall Health`, `Overall Hint`, `Overall Focus`, `Overall Route`,
  `Overall Playbook`, and `Overall Next Step` for both scenarios.

## Non-Goals
- No changes to overview rendering logic.
- No changes to existing failure-path assertions.

## Options Considered
- Option A: keep only failure-path coverage.
- Option B: add explicit aggregate-state coverage for warn/info.

## Selected Approach
Choose Option B so the top-level severity-aware guidance is regression-tested
across the full set of meaningful aggregate health states.

## Data / Contract Impact
- No output contract changes.
- Test coverage only.

## Risks
- Minimal; test fixture maintenance increases slightly.

## Rollback Plan
- Remove the additional warn/info assertions if they prove too brittle.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase91-governance-overview-warn-info-coverage.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase91-governance-overview-warn-info-coverage.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to strengthen aggregate overview regression coverage.

## Implementation Summary
- Added dedicated warn/info aggregate scenarios to overview regression tests.
- Updated supporting docs and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase91-governance-overview-warn-info-coverage.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
