# Phase 65 Custom Path Manual Changed-Baselines Preflight Smoke

## Title
Phase 65 - Add smoke scenario for custom path preflight with manual changed-baselines override consistency

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
- `docs/specs/2026-03-30-architecture-phase65-custom-path-manual-changed-baselines-preflight-smoke.md`

## Context
Phase 64 added custom path + no-refresh preflight coverage, but manual changed-baselines override behavior under custom path is not smoke-tested.

## Problem Statement
Without this scenario, regressions in manual changed-baselines override precedence could produce inconsistent preflight guidance in enterprise workflows.

## Scope
- Add smoke scenario for preflight with:
  - custom resolver baseline path override
  - resolver refresh disabled
  - explicit `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` value
- Validate:
  - preflight output uses manual changed-baselines value
  - resolver target line is still absent when refresh is disabled

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on auto-derived changed-baselines scenarios only.
- Option B: add explicit manual-override scenario.

## Selected Approach
Choose Option B to lock in manual override precedence and avoid governance ambiguity.

## Data / Contract Impact
- Smoke case matrix expands with one manual-override preflight scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; temp repo smoke-only assertions.

## Rollback Plan
- Remove manual-override preflight scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase65-custom-path-manual-changed-baselines-preflight-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase65-custom-path-manual-changed-baselines-preflight-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to enforce preflight consistency for manual changed-baselines overrides.

## Implementation Summary
- Added custom-path preflight manual changed-baselines override smoke scenario:
  - `custom_resolver_path_preflight_manual_changed_baselines_override`
- New scenario validates:
  - manual `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` value is preserved in
    preflight output
  - resolver target preview line remains absent when resolver refresh is disabled
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase65-custom-path-manual-changed-baselines-preflight-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
