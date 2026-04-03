# Phase 63 Custom Path Blocked Update Smoke

## Title
Phase 63 - Add negative smoke scenario for blocked update with custom resolver baseline path (no side effects)

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
- `docs/specs/2026-03-30-architecture-phase63-custom-path-blocked-update-smoke.md`

## Context
Phase 62 covered missing custom resolver path failure on governance check, but blocked update behavior with custom path override is not smoke-tested.

## Problem Statement
Without this scenario, blocked update regressions might still create custom baseline files or mutate changelog unexpectedly when custom path overrides are configured.

## Scope
- Add negative smoke scenario:
  - custom resolver baseline path + `CHAINPULSE_ALLOW_BASELINE_UPDATE=false`
- Validate:
  - update command fails
  - custom baseline file is not created
  - changelog remains unchanged

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on default-path blocked update smoke only.
- Option B: add custom-path blocked update smoke.

## Selected Approach
Choose Option B for stronger enterprise safety guarantees under custom path overrides.

## Data / Contract Impact
- Smoke case matrix expands with one custom-path blocked update negative scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; isolated temp repo smoke behavior.

## Rollback Plan
- Remove custom-path blocked update negative scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase63-custom-path-blocked-update-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase63-custom-path-blocked-update-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to ensure blocked updates stay side-effect free under custom path overrides.

## Implementation Summary
- Added custom-path blocked update negative smoke scenario:
  - `custom_resolver_baseline_path_blocked_update_should_not_create_file`
- New scenario validates:
  - update command fails with `CHAINPULSE_ALLOW_BASELINE_UPDATE=false`
  - custom resolver baseline file is not created
  - changelog is not mutated
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase63-custom-path-blocked-update-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
