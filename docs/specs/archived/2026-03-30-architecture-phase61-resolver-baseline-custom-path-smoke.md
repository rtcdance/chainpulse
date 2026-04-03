# Phase 61 Resolver Baseline Custom Path Smoke

## Title
Phase 61 - Add smoke coverage for custom resolver baseline path override parity

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
- `docs/specs/2026-03-30-architecture-phase61-resolver-baseline-custom-path-smoke.md`

## Context
Phase 60 hardened guarded update blocking, but smoke coverage does not yet validate custom resolver baseline path override behavior across preflight/update/governance steps.

## Problem Statement
Without custom-path smoke checks, path override regressions can break enterprise repo layouts where governance baselines are stored in non-default locations.

## Scope
- Add smoke fixture custom resolver baseline path.
- Add preflight smoke assertion for resolver target line under custom path.
- Add guarded update smoke scenario with custom resolver baseline path and post-update governance check.
- Keep existing default-path behavior unchanged.

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on default-path smoke only.
- Option B: add custom-path parity smoke.

## Selected Approach
Choose Option B for enterprise layout compatibility and governance reliability.

## Data / Contract Impact
- Smoke case matrix expands with custom-path parity scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; temp repo smoke-only changes.

## Rollback Plan
- Remove custom-path parity smoke scenario and related fixture adjustments.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase61-resolver-baseline-custom-path-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase61-resolver-baseline-custom-path-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to ensure custom resolver baseline path parity in governance loops.

## Implementation Summary
- Added custom resolver baseline path parity smoke scenario:
  - `custom_resolver_baseline_path_parity`
- New scenario validates parity across preflight/update/governance:
  - preflight shows custom resolver baseline target file
  - guarded update writes resolver baseline metrics to custom file path
  - default resolver baseline file remains unchanged in custom-path case
  - post-update governance check passes with custom resolver baseline path override
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase61-resolver-baseline-custom-path-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
