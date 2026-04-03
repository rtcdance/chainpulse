# Phase 60 Update Blocked Smoke

## Title
Phase 60 - Add negative smoke scenario for blocked baseline updates when allow flag is false

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
- `docs/specs/2026-03-30-architecture-phase60-update-blocked-smoke.md`

## Context
Phase 59 covered guarded update-flow success path, but negative-path enforcement for blocked updates is not smoke-tested.

## Problem Statement
Without blocked-path smoke validation, accidental relaxation of `CHAINPULSE_ALLOW_BASELINE_UPDATE` guard could allow unauthorized baseline mutations.

## Scope
- Add negative smoke scenario that executes update-flow with allow flag unset/false.
- Validate command fails and key baseline/changelog files remain unchanged.
- Update operations and architecture docs with scenario coverage.

## Non-Goals
- No runtime service/domain behavior changes.
- No CI workflow structural changes.

## Options Considered
- Option A: rely on unit-level script checks only.
- Option B: add end-to-end smoke negative-path scenario.

## Selected Approach
Choose Option B for stronger enterprise governance guard confidence.

## Data / Contract Impact
- Smoke case matrix expands with one blocked-update negative scenario.
- Existing smoke artifact schema unchanged.

## Risks
- Minimal; isolated to temp repo smoke validation.

## Rollback Plan
- Remove blocked-update negative scenario and related doc updates.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase60-update-blocked-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase60-update-blocked-smoke.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to enforce blocked-path governance protection in smoke loops.

## Implementation Summary
- Added blocked update negative smoke scenario:
  - `guarded_update_blocked_without_allow_flag`
- New scenario validates:
  - update command fails when `CHAINPULSE_ALLOW_BASELINE_UPDATE=false`
  - resolver baseline file is unchanged
  - changelog is not mutated
- Updated operations dashboard scenario list and architecture phase log.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase60-update-blocked-smoke.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
