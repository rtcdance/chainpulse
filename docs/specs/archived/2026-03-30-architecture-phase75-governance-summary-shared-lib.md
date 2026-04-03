# Phase 75 Governance Summary Shared Library

## Title
Phase 75 - Extract shared governance job-summary rendering library

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
- `scripts/lib/governance_summary.sh`
- `scripts/append-baseline-scope-smoke-summary.sh`
- `scripts/append-baseline-resolver-test-summary.sh`
- `scripts/test-append-baseline-scope-smoke-summary.sh`
- `scripts/test-append-baseline-resolver-test-summary.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase75-governance-summary-shared-lib.md`

## Context
Phase 72-74 introduced smoke and resolver CI summary helpers with nearly
identical parsing and delta-rendering logic.

## Problem Statement
Keeping multiple near-duplicate summary helpers increases maintenance cost and
raises the chance of rendering drift between smoke and resolver governance
surfaces.

## Scope
- Extract shared markdown parsing and delta rendering helpers into a common
  shell library.
- Refactor smoke and resolver summary entrypoints to use the shared library.
- Keep existing helper script interfaces and rendered output behavior stable.

## Non-Goals
- No changes to CI workflow call sites.
- No changes to smoke/resolver artifact contracts or governance semantics.

## Options Considered
- Option A: keep duplicated helper logic.
- Option B: extract a shared summary rendering library behind the existing
  helper entrypoints.

## Selected Approach
Choose Option B to reduce duplication while preserving the established CI
integration and operator-facing output format.

## Data / Contract Impact
- Internal script structure changes only.
- Existing summary script interfaces and rendered markdown remain compatible.

## Risks
- Low; shell-library extraction could introduce sourcing or quoting regressions.

## Rollback Plan
- Inline the shared helper logic back into the smoke and resolver entrypoint
  scripts.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase75-governance-summary-shared-lib.md`
- `bash -n scripts/lib/governance_summary.sh`
- `bash -n scripts/append-baseline-scope-smoke-summary.sh`
- `bash -n scripts/append-baseline-resolver-test-summary.sh`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- `./scripts/compare-baseline-resolver-test.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase75-governance-summary-shared-lib.md`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`

## Review Notes
- Approved to reduce duplication while preserving established CI summary
  behavior.

## Implementation Summary
- Added `scripts/lib/governance_summary.sh` for shared markdown parsing,
  validation, and delta-highlight rendering helpers.
- Refactored smoke and resolver summary entrypoints to source the shared
  library while preserving their existing interfaces and rendered output.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase75-governance-summary-shared-lib.md`
- `bash -n scripts/lib/governance_summary.sh`
- `bash -n scripts/append-baseline-scope-smoke-summary.sh`
- `bash -n scripts/append-baseline-resolver-test-summary.sh`
- `./scripts/test-append-baseline-scope-smoke-summary.sh`
- `./scripts/test-append-baseline-resolver-test-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- `./scripts/compare-baseline-resolver-test.sh`
