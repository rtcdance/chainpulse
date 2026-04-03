# Phase 58 Resolver Refresh Preflight Smoke

## Title
Phase 58 - Add smoke coverage for resolver refresh flag and preflight changelog preview consistency

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
- `docs/specs/2026-03-30-architecture-phase58-resolver-refresh-preflight-smoke.md`

## Context
Phase 56 introduced optional resolver baseline refresh controls, but smoke tests do not verify that preflight output reflects resolver refresh intent consistently.

## Problem Statement
Without preflight consistency smoke checks, resolver refresh flag regressions can escape early governance validation.

## Scope
- Extend smoke suite fixture to include preflight script dependencies.
- Add preflight consistency scenarios for:
  - resolver refresh disabled
  - resolver refresh enabled
- Validate emitted preflight markdown `changed_baselines` and resolver target file line presence/absence.

## Non-Goals
- No runtime service/domain behavior changes.
- No baseline refresh mutation in smoke tests.

## Options Considered
- Option A: rely on manual preflight checks.
- Option B: automate preflight consistency smoke checks.

## Selected Approach
Choose Option B to keep governance policy checks deterministic and repeatable.

## Data / Contract Impact
- Smoke case matrix expands with preflight consistency cases.
- No changes to smoke artifact schema.

## Risks
- Minimal; scenario-level test coverage only.

## Rollback Plan
- Remove preflight-specific smoke scenarios and fixture copies.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase58-resolver-refresh-preflight-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase58-resolver-refresh-preflight-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to enforce resolver preflight intent consistency in smoke loops.

## Implementation Summary
- Extended smoke fixture setup to include preflight dependencies:
  - `scripts/preflight-migration-baseline-update.sh`
  - `scripts/lib/baseline_update_resolver.sh`
- Added preflight consistency smoke helper cases:
  - `preflight_without_resolver_refresh`
  - `preflight_with_resolver_refresh`
- New smoke assertions validate:
  - preflight `changed_baselines` preview value
  - resolver target baseline line presence/absence in markdown output
- Updated operations dashboard query doc and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase58-resolver-refresh-preflight-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
