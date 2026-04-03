# Phase 77 Governance Overview Summary

## Title
Phase 77 - Add compact governance overview block to CI job summary

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
- `scripts/lib/governance_summary.sh`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase77-governance-overview-summary.md`

## Context
Smoke and resolver highlights are now individually readable, but operators still
need to scan multiple sections to understand top-level governance state.

## Problem Statement
Without a compact overview block, the CI summary lacks a single place that
shows smoke and resolver status together for fast first-pass triage.

## Scope
- Add a `Governance Overview` summary helper for CI.
- Include smoke and resolver status, pass/fail totals, and regression status in
  one compact table.
- Wire the overview block into CI before smoke/resolver detail sections.

## Non-Goals
- No changes to smoke/resolver artifact contracts.
- No removal of existing per-surface highlight sections.

## Options Considered
- Option A: keep only separate smoke/resolver highlight sections.
- Option B: add a higher-level overview block before the detailed sections.

## Selected Approach
Choose Option B for faster triage while preserving the existing detailed
operator-facing sections.

## Data / Contract Impact
- CI job summary gains a new `Governance Overview` section.
- Existing smoke/resolver artifacts and highlight helpers remain unchanged.

## Risks
- Low; summary rendering only.

## Rollback Plan
- Remove the overview helper and its CI invocation.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase77-governance-overview-summary.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- `./scripts/compare-baseline-resolver-test.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase77-governance-overview-summary.md`
- `./scripts/test-append-governance-overview-summary.sh`

## Review Notes
- Approved to improve top-level governance triage ergonomics in CI.

## Implementation Summary
- Added `scripts/append-governance-overview-summary.sh` to append a compact
  smoke/resolver overview table to the CI job summary.
- Added `scripts/test-append-governance-overview-summary.sh` to validate the
  overview rendering contract.
- Updated `.github/workflows/ci.yml` to prepend the overview block before
  smoke/resolver detail sections.
- Updated operations dashboard doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase77-governance-overview-summary.md`
- `bash -n scripts/append-governance-overview-summary.sh`
- `bash -n scripts/test-append-governance-overview-summary.sh`
- `./scripts/test-append-governance-overview-summary.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/compare-baseline-scope-smoke.sh`
- `./scripts/compare-baseline-resolver-test.sh`
