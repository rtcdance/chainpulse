# Phase 49 Baseline Update Preflight

## Title
Phase 49 - Add baseline update preflight preview script for scope and changed-set resolution

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
- `scripts/preflight-migration-baseline-update.sh`
- `scripts/update-migration-governance-baseline.sh`
- `Makefile`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/guides/OPERATIONS_GUIDE.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 48 added generated changelog template output during baseline updates, but operators still lacked a zero-mutation preflight step to inspect resolved governance fields before any file change.

## Problem Statement
Running the full update command to inspect resolution behavior can introduce unintended repository changes.

## Scope
- Add preflight script that performs validation and resolution only:
  - `scripts/preflight-migration-baseline-update.sh`
- Preflight computes:
  - resolved `scope`
  - resolved `changed_baselines`
  - target files per refresh flags
- Preflight writes markdown artifact:
  - default: `build/migration-governance/baseline-update-preflight.md`
- Add Makefile target:
  - `preflight-migration-baseline-update`

## Non-Goals
- No baseline file mutation in preflight mode.
- No runtime service behavior change.

## Options Considered
- Option A: rely only on update script output.
- Option B: add explicit no-mutation preflight command.

## Selected Approach
Choose Option B to improve safety and operator confidence before governed updates.

## Data / Contract Impact
- Governance workflow artifact set extended with preflight markdown output.
- No runtime API/domain contract impact.

## Risks
- Logic drift between preflight and update script over time.
- Mitigation: keep resolution rules aligned and minimal.

## Rollback Plan
- Remove preflight script and Makefile entry; retain update script template preview.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase49-baseline-update-preflight.md`
- `bash -n scripts/preflight-migration-baseline-update.sh`
- `./scripts/preflight-migration-baseline-update.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase49-baseline-update-preflight.md`
- `./scripts/preflight-migration-baseline-update.sh`

## Review Notes
- Approved to add safer operator preview step before governed baseline mutations.

## Implementation Summary
- Added standalone preflight script with resolved scope/changed-set preview.
- Added Makefile target for quick execution.
- Updated operations docs and index with preflight artifact reference.

## Final Verification
- Preflight script runs successfully and emits preview markdown with resolved governance fields.
