# Phase 59 Guarded Update Flow Smoke

## Title
Phase 59 - Add guarded update-flow smoke scenario for resolver refresh governance

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
- `docs/specs/2026-03-30-architecture-phase59-guarded-update-flow-smoke.md`

## Context
Phase 58 covered preflight consistency, but smoke tests do not yet validate guarded update-flow behavior under `CHAINPULSE_ALLOW_BASELINE_UPDATE=true`.

## Problem Statement
Without update-flow smoke coverage, regressions in baseline mutation and changelog insertion logic can go undetected until later CI/policy checks.

## Scope
- Extend smoke fixture dependencies for update-flow scripts.
- Add guarded update-flow scenario with resolver refresh enabled.
- Validate:
  - resolver baseline file mutation
  - changelog insertion with expected changed-baselines tag
  - post-update governance check success against diff ref

## Non-Goals
- No runtime service/domain behavior changes.
- No production baseline file mutation outside smoke temp repos.

## Options Considered
- Option A: rely on preflight-only smoke.
- Option B: add guarded update-flow smoke path.

## Selected Approach
Choose Option B for stronger policy confidence and end-to-end governance validation.

## Data / Contract Impact
- Smoke case matrix expands with guarded update-flow scenario.
- Existing smoke artifact schema remains unchanged.

## Risks
- Minimal; isolated to temp repo smoke execution.

## Rollback Plan
- Remove guarded update-flow scenario and related fixture script copies.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase59-guarded-update-flow-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase59-guarded-update-flow-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to extend smoke from preflight checks to guarded update-flow validation.

## Implementation Summary
- Extended smoke fixture setup with guarded update-flow dependencies:
  - `scripts/update-migration-governance-baseline.sh`
  - `scripts/export-migration-governance-kpi.sh`
  - `scripts/test-baseline-update-resolver.sh`
  - `docs/operations/MIGRATION_MANIFEST.csv` (fixture)
- Added guarded update-flow smoke case:
  - `guarded_update_with_resolver_refresh`
- New smoke assertions validate:
  - resolver baseline file mutation after guarded update
  - changelog insertion with `changed_baselines=kpi,resolver`
  - post-update governance check pass with `CHAINPULSE_MIGRATION_BASELINE_DIFF_REF=HEAD~1`
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase59-guarded-update-flow-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
