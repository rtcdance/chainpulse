# Phase 57 Resolver Changed-Baselines Smoke Coverage

## Title
Phase 57 - Add resolver changed-baselines smoke scenarios (positive + mismatch)

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
- `scripts/smoke-baseline-governance-scope.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`
- `docs/specs/2026-03-30-architecture-phase57-resolver-changed-baselines-smoke-coverage.md`

## Context
Phase 56 introduced `resolver` changed-baselines semantics, but smoke coverage does not explicitly test resolver-positive and resolver-mismatch governance behavior.

## Problem Statement
Without dedicated resolver smoke scenarios, policy regressions for resolver changed-baselines alignment may bypass fast feedback loops.

## Scope
- Extend smoke fixture setup to include resolver baseline file.
- Add resolver-positive scenario for changed-baselines alignment.
- Add resolver-mismatch negative scenario.
- Update operations/architecture docs to reflect expanded scenario coverage.

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structure change.

## Options Considered
- Option A: keep existing smoke coverage only.
- Option B: add resolver-specific smoke scenarios.

## Selected Approach
Choose Option B for end-to-end governance confidence after resolver policy expansion.

## Data / Contract Impact
- Smoke case matrix expands by resolver scenarios only.
- Existing smoke artifact schema remains unchanged.

## Risks
- Minimal; isolated to smoke fixtures and scenario matrix.

## Rollback Plan
- Remove resolver smoke scenarios and restore previous 5-case matrix.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase57-resolver-changed-baselines-smoke-coverage.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase57-resolver-changed-baselines-smoke-coverage.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to close governance smoke coverage gap for resolver semantics.

## Implementation Summary
- Extended smoke fixture setup to include resolver baseline file:
  - `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom` (fixture)
- Extended smoke mutate modes with resolver-only baseline mutation path.
- Added resolver-specific smoke scenarios:
  - `resolver_changed_baselines_alignment` (expected success)
  - `resolver_changed_baselines_mismatch_should_fail` (expected failure)
- Updated operations dashboard query doc scenario list.
- Updated architecture phase log and next phase pointer.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase57-resolver-changed-baselines-smoke-coverage.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
