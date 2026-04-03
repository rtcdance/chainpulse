# Phase 46 Baseline Changed-Set Tagging

## Title
Phase 46 - Add `changed_baselines` changelog tagging and alignment enforcement

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
- `scripts/check-migration-changelog-quality.sh`
- `scripts/update-migration-governance-baseline.sh`
- `scripts/check-migration-baseline-governance.sh`
- `scripts/smoke-baseline-governance-scope.sh`
- `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 45 unified smoke baseline refresh governance, but changelog entries still lacked explicit machine-readable tagging of which baseline files were changed.

## Problem Statement
Scope tags (`kpi-only|health-only|dual`) are coarse and cannot precisely encode whether `health`, `smoke`, or both baselines changed.

## Scope
- Extend changelog entry schema with required field:
  - `changed_baselines=<kpi[,health][,smoke]>`
- Extend changelog quality gate:
  - parse/validate `changed_baselines`
  - enforce scope vs changed-set compatibility
- Extend baseline refresh script:
  - auto-write `changed_baselines` from actual refresh toggles
  - optional override via `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES`
- Extend baseline governance check:
  - enforce latest changelog `changed_baselines` equals actual changed baseline set
  - control via `CHAINPULSE_MIGRATION_ENFORCE_CHANGED_BASELINES_ALIGNMENT=true|false`
- Update smoke fixtures to include `changed_baselines`.

## Non-Goals
- No runtime service behavior change.
- No migration to external change-management systems.

## Options Considered
- Option A: keep scope-only tagging.
- Option B: add explicit changed baseline set tagging and enforcement.

## Selected Approach
Choose Option B for stronger machine verifiability and enterprise audit quality.

## Data / Contract Impact
- Governance changelog format contract expanded with `changed_baselines`.
- No API/domain runtime contract impact.

## Risks
- Existing entries missing `changed_baselines` can fail checks.
- Mitigation: backfill historical entries and keep strict policy toggles explicit in CI.

## Rollback Plan
- Temporarily set:
  - `CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_CHANGED_BASELINES=false`
  - `CHAINPULSE_MIGRATION_ENFORCE_CHANGED_BASELINES_ALIGNMENT=false`

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase46-baseline-changed-set-tagging.md`
- `bash -n scripts/check-migration-changelog-quality.sh`
- `bash -n scripts/update-migration-governance-baseline.sh`
- `bash -n scripts/check-migration-baseline-governance.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
- `./scripts/smoke-baseline-governance-scope.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase46-baseline-changed-set-tagging.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Review Notes
- Approved to improve baseline change auditability with precise changed-set tags.

## Implementation Summary
- Added `changed_baselines` parsing/validation and scope compatibility checks.
- Added auto-write/override for changed baseline sets in refresh workflow.
- Added changed-set alignment gate in baseline governance checks.
- Updated smoke fixture changelog entries and docs/CI policy toggles.

## Final Verification
- Governance checks pass with `changed_baselines`-tagged changelog entries.
