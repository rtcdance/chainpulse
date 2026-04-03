# Phase 62 Custom Resolver Baseline Missing Smoke

## Title
Phase 62 - Add negative smoke scenario for missing custom resolver baseline path in governance check

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
- `docs/specs/2026-03-30-architecture-phase62-custom-resolver-baseline-missing-smoke.md`

## Context
Phase 61 covered custom resolver baseline path parity for successful flows, but missing-path failure behavior is not smoke-tested.

## Problem Statement
Without missing-path negative smoke coverage, governance checks may regress and produce unclear behavior when custom baseline path configuration is broken.

## Scope
- Add negative smoke scenario for governance check with missing custom resolver baseline file.
- Validate governance check fails as expected and scenario is recorded as pass (expected failure).
- Update operations/architecture docs with scenario list changes.

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on script-level manual validation.
- Option B: add end-to-end smoke missing-path scenario.

## Selected Approach
Choose Option B for explicit enterprise failure-path governance coverage.

## Data / Contract Impact
- Smoke case matrix expands with one missing-path negative scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; temp repo smoke-only scenario.

## Rollback Plan
- Remove missing-path negative scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase62-custom-resolver-baseline-missing-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase62-custom-resolver-baseline-missing-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to harden missing custom baseline path failure-path coverage.

## Implementation Summary
- Added missing custom resolver baseline path negative smoke scenario:
  - `custom_resolver_baseline_path_missing_should_fail`
- New scenario validates governance check fails when
  `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE` points to a missing file.
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase62-custom-resolver-baseline-missing-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
