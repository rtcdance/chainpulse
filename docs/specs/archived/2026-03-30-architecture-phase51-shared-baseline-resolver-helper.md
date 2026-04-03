# Phase 51 Shared Baseline Resolver Helper

## Title
Phase 51 - Introduce shared scope/changed-set resolver helper for preflight and update scripts

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
- `scripts/lib/baseline_update_resolver.sh`
- `scripts/update-migration-governance-baseline.sh`
- `scripts/preflight-migration-baseline-update.sh`
- `docs/ARCHITECTURE.md`

## Context
Phase 49/50 introduced preflight and template preview flows, but both scripts duplicated scope and changed-baseline resolution logic.

## Problem Statement
Duplicated resolution logic creates drift risk and inconsistent behavior between preflight and update execution paths.

## Scope
- Add shared resolver helper:
  - `scripts/lib/baseline_update_resolver.sh`
- Move shared logic into helper functions:
  - scope validation
  - changed baseline normalization
  - resolved scope computation
  - resolved changed baseline computation
- Update scripts to source and reuse helper:
  - `scripts/update-migration-governance-baseline.sh`
  - `scripts/preflight-migration-baseline-update.sh`

## Non-Goals
- No runtime service behavior changes.
- No governance policy change; this phase is refactor-only.

## Options Considered
- Option A: keep duplicated logic.
- Option B: extract shared helper and source from both scripts.

## Selected Approach
Choose Option B to reduce maintenance risk and keep preflight/update behavior consistent.

## Data / Contract Impact
- No contract change; behavior is preserved.

## Risks
- Incorrect sourcing path may break script execution.
- Mitigation: use script-dir absolute resolution and run syntax/functional checks.

## Rollback Plan
- Revert scripts to inlined resolver logic and remove helper file.

## Test and Verification Plan
- `bash -n scripts/lib/baseline_update_resolver.sh`
- `bash -n scripts/preflight-migration-baseline-update.sh`
- `bash -n scripts/update-migration-governance-baseline.sh`
- `./scripts/preflight-migration-baseline-update.sh`
- guarded run of `./scripts/update-migration-governance-baseline.sh` with temp files

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase51-shared-baseline-resolver-helper.md`
- script syntax and functional checks above

## Review Notes
- Approved to reduce policy logic drift risk between preflight and update workflows.

## Implementation Summary
- Added shared resolver helper and wired both scripts to source it.
- Removed duplicated normalization/resolution logic from both scripts.

## Final Verification
- Preflight and guarded update runs produce expected resolved `scope` and `changed_baselines`.
