# Phase 64 Custom Path Preflight No-Refresh Smoke

## Title
Phase 64 - Add negative smoke scenario for custom resolver path preflight with refresh disabled

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
- `docs/specs/2026-03-30-architecture-phase64-custom-path-preflight-no-refresh-smoke.md`

## Context
Phase 63 hardened blocked update side-effect checks, but preflight behavior for custom resolver path when refresh is disabled lacks explicit smoke coverage.

## Problem Statement
Without this negative smoke case, preflight output regressions may leak resolver target lines even when resolver refresh is disabled.

## Scope
- Add smoke scenario for custom resolver path preflight with:
  - `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE=<custom>`
  - `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=false`
- Validate preflight markdown does not contain resolver target file preview line.
- Update operations/architecture docs with scenario list changes.

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on default-path no-refresh preflight scenario only.
- Option B: add custom-path no-refresh preflight scenario.

## Selected Approach
Choose Option B for explicit custom-path preflight guardrail coverage.

## Data / Contract Impact
- Smoke case matrix expands with one custom-path no-refresh scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; temp repo smoke-only assertions.

## Rollback Plan
- Remove custom-path no-refresh preflight scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase64-custom-path-preflight-no-refresh-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase64-custom-path-preflight-no-refresh-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to prevent resolver preflight target leakage when refresh is disabled under custom path overrides.

## Implementation Summary
- Added custom-path no-refresh preflight smoke scenario:
  - `custom_resolver_path_preflight_no_refresh_should_not_show_target`
- New scenario validates no resolver target preview leakage in preflight output
  when:
  - `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=false`
  - custom resolver baseline path override is configured
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase64-custom-path-preflight-no-refresh-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
