# Phase 67 Invalid Changed-Baselines Guarded Update Smoke

## Title
Phase 67 - Add negative smoke scenario for invalid manual changed-baselines in guarded update flow

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
- `docs/specs/2026-03-30-architecture-phase67-invalid-changed-baselines-guarded-update-smoke.md`

## Context
Phase 66 covered invalid manual changed-baselines in preflight path, but guarded update mutation path does not yet have equivalent negative smoke coverage.

## Problem Statement
Without guarded update negative coverage, invalid manual changed-baselines values could regress into mutation flow and potentially produce unintended baseline/changelog side effects.

## Scope
- Add negative smoke scenario for guarded update with:
  - `CHAINPULSE_ALLOW_BASELINE_UPDATE=true`
  - invalid `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES`
- Validate:
  - update command fails
  - resolver baseline file unchanged
  - changelog unchanged

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on preflight-only invalid-value checks.
- Option B: add guarded update invalid-value checks.

## Selected Approach
Choose Option B to enforce validation parity across preview and mutation paths.

## Data / Contract Impact
- Smoke case matrix expands with one guarded update invalid-value negative scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; temp repo smoke-only assertions.

## Rollback Plan
- Remove guarded update invalid-value scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase67-invalid-changed-baselines-guarded-update-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase67-invalid-changed-baselines-guarded-update-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to enforce invalid-value failure parity in guarded update flow.

## Implementation Summary
- Added guarded update invalid changed-baselines negative smoke scenario:
  - `guarded_update_invalid_changed_baselines_should_fail`
- New scenario validates:
  - guarded update command fails with invalid manual changed-baselines override
  - resolver baseline file remains unchanged
  - changelog remains unchanged
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase67-invalid-changed-baselines-guarded-update-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
