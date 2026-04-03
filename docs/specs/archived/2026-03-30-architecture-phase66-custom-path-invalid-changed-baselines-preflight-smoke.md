# Phase 66 Custom Path Invalid Changed-Baselines Preflight Smoke

## Title
Phase 66 - Add negative smoke scenario for invalid manual changed-baselines under custom path preflight

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
- `docs/specs/2026-03-30-architecture-phase66-custom-path-invalid-changed-baselines-preflight-smoke.md`

## Context
Phase 65 added manual changed-baselines override parity checks, but invalid manual values under custom path preflight are not smoke-tested.

## Problem Statement
Without this negative smoke case, validation regressions could allow malformed changed-baselines values and weaken governance guardrails.

## Scope
- Add smoke scenario for custom resolver path preflight with invalid:
  - `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES`
- Validate preflight command fails (expected failure path).
- Update operations and architecture docs with scenario coverage.

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on script-level validation only.
- Option B: add end-to-end smoke validation.

## Selected Approach
Choose Option B for robust and repeatable governance failure-path confidence.

## Data / Contract Impact
- Smoke case matrix expands with one invalid-manual-override negative scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; temp repo smoke-only behavior.

## Rollback Plan
- Remove invalid-manual-override scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase66-custom-path-invalid-changed-baselines-preflight-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase66-custom-path-invalid-changed-baselines-preflight-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to enforce explicit failure for invalid manual changed-baselines override values.

## Implementation Summary
- Added invalid manual changed-baselines override negative smoke scenario:
  - `custom_resolver_path_preflight_invalid_changed_baselines_should_fail`
- New scenario validates explicit preflight failure when
  `CHAINPULSE_BASELINE_UPDATE_CHANGED_BASELINES` contains invalid tokens under
  custom resolver path override flow.
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase66-custom-path-invalid-changed-baselines-preflight-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
