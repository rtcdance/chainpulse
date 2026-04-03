# Phase 102 Governance Overview Parser Helper Cleanup

## Title
Phase 102 - Extract cleaner parser helpers for overview descriptor handling

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
- `docs/specs/2026-03-30-architecture-phase102-governance-overview-parser-helper-cleanup.md`

## Context
Phase 101 completed happy-path and malformed-path coverage for descriptors, but
the test script still keeps parsing and validation details fairly close to the
scenario code.

## Problem Statement
As descriptor support grows, keeping parser details inline in the main test file
makes the core scenario flow harder to scan than it needs to be.

## Scope
- Extract small parser helpers for setup, aggregate, and row descriptors.
- Preserve current descriptor formats and validation behavior.
- Keep test coverage and output contracts unchanged.

## Non-Goals
- No changes to overview rendering logic.
- No new scenario coverage beyond the existing suite.

## Options Considered
- Option A: keep parser logic inline.
- Option B: factor parser logic into narrower helper functions.

## Selected Approach
Choose Option B so the regression harness stays readable while preserving the
same descriptor-driven behavior and safeguards.

## Data / Contract Impact
- No output contract changes.
- Test harness internals become more modular.

## Risks
- Minimal; helper extraction could introduce parsing regressions if done carelessly.

## Rollback Plan
- Restore inline parser logic in the overview regression script.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase102-governance-overview-parser-helper-cleanup.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase102-governance-overview-parser-helper-cleanup.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to keep the overview regression harness easier to maintain.

## Implementation Summary
- Extracted parser helpers for setup, aggregate, and row descriptors.
- Kept existing validation and scenario coverage intact.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase102-governance-overview-parser-helper-cleanup.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
