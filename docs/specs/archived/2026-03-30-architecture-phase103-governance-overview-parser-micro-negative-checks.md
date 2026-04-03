# Phase 103 Governance Overview Parser Micro Negative Checks

## Title
Phase 103 - Add parser-focused micro negative checks for overview descriptors

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
- `docs/specs/2026-03-30-architecture-phase103-governance-overview-parser-micro-negative-checks.md`

## Context
Phase 102 extracted parser helpers for setup, aggregate, and row descriptors,
while Phase 101 already covered malformed descriptors at the script flow level.

## Problem Statement
The current negative tests prove the harness fails on bad descriptors, but they
still exercise those failures through higher-level wrapper functions instead of
checking parser-layer entry points directly.

## Scope
- Add micro negative checks that call parser-layer functions directly.
- Cover malformed aggregate descriptors at the parser level.
- Cover malformed row descriptors at the parser level.

## Non-Goals
- No changes to overview rendering logic.
- No changes to descriptor formats or validation rules.

## Options Considered
- Option A: keep only script-level malformed descriptor checks.
- Option B: add parser-focused micro negative checks alongside them.

## Selected Approach
Choose Option B so parser-layer failures stay explicit even if wrapper flow
changes later.

## Data / Contract Impact
- No output contract changes.
- Test harness gains finer-grained negative coverage.

## Risks
- Minimal; validation wording becomes slightly more contractual.

## Rollback Plan
- Remove parser-focused micro negative checks while retaining broader malformed
  descriptor tests.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase103-governance-overview-parser-micro-negative-checks.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase103-governance-overview-parser-micro-negative-checks.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to make parser-layer validation failures more directly exercised.

## Implementation Summary
- Added parser-level micro negative checks for aggregate and row descriptor parsing.
- Kept existing higher-level malformed descriptor tests intact.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase103-governance-overview-parser-micro-negative-checks.md`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
