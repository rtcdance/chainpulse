# Phase 93 Governance Overview Assertion Helpers

## Title
Phase 93 - Extract aggregate and row assertion helpers for governance overview tests

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
- `docs/specs/2026-03-30-architecture-phase93-governance-overview-assertion-helpers.md`

## Context
Phase 92 extracted fixture writers, but the overview regression script still
contains repeated aggregate and row assertions that make scenario additions
noisier than necessary.

## Problem Statement
As the aggregate-state matrix grows, repeated assertion lines make the test
harder to scan and increase maintenance cost when output wording changes.

## Scope
- Add helper functions for aggregate overview assertions.
- Add helper functions for common row assertions.
- Preserve existing scenario coverage and output contracts.

## Non-Goals
- No changes to overview rendering logic.
- No changes to fixture semantics or aggregate coverage breadth.

## Options Considered
- Option A: keep explicit inline assertions.
- Option B: extract reusable assertion helpers.

## Selected Approach
Choose Option B so the test file reads more like scenario intent and less like
repeated string-matching boilerplate.

## Data / Contract Impact
- No output contract changes.
- Test structure only.

## Risks
- Minimal; helper bugs could obscure a failed assertion if written poorly.

## Rollback Plan
- Revert helper extraction and restore inline assertions.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase93-governance-overview-assertion-helpers.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase93-governance-overview-assertion-helpers.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve overview regression readability and maintainability.

## Implementation Summary
- Added aggregate and row assertion helpers to the overview regression script.
- Kept existing scenario coverage while reducing repeated inline assertions.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase93-governance-overview-assertion-helpers.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
