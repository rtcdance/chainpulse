# Phase 56 Resolver Baseline Refresh Controls

## Title
Phase 56 - Add governed resolver baseline refresh controls and changelog alignment

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
- `scripts/preflight-migration-baseline-update.sh`
- `scripts/update-migration-governance-baseline.sh`
- `scripts/check-migration-changelog-quality.sh`
- `scripts/check-migration-baseline-governance.sh`
- `scripts/test-baseline-update-resolver.sh`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/guides/OPERATIONS_GUIDE.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 55 added resolver delta comparison, but baseline refresh workflow does not yet support governed resolver baseline refresh and corresponding changelog alignment semantics.

## Problem Statement
Without refresh controls and policy alignment for resolver baseline, resolver regression governance remains partially manual and can drift from changelog intent tags.

## Scope
- Add opt-in resolver baseline refresh control in preflight/update workflow.
- Extend changed-baselines normalization/validation to include `resolver` tag.
- Extend baseline governance diff check to include resolver baseline file and alignment check.
- Keep scope semantics unchanged (`kpi-only|health-only|dual`), while allowing `resolver` as orthogonal changed-baselines tag.
- Update CI changelog allowlist default and operations docs.

## Non-Goals
- No runtime service behavior change.
- No new deployment profile or API contract change.

## Options Considered
- Option A: keep resolver baseline refresh manual.
- Option B: add opt-in resolver refresh with changelog alignment policy.

## Selected Approach
Choose Option B for enterprise governance consistency and auditability.

## Data / Contract Impact
- New refresh controls:
  - `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE` (default `false`)
  - `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE`
- `changed_baselines` allowlist expands from `kpi,health,smoke` to `kpi,health,smoke,resolver`.

## Risks
- Minimal; governance script changes only.

## Rollback Plan
- Revert resolver refresh env controls and `resolver` changed-baselines support.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase56-resolver-baseline-refresh-controls.md`
- `bash -n scripts/lib/baseline_update_resolver.sh`
- `bash -n scripts/preflight-migration-baseline-update.sh`
- `bash -n scripts/update-migration-governance-baseline.sh`
- `bash -n scripts/check-migration-changelog-quality.sh`
- `bash -n scripts/check-migration-baseline-governance.sh`
- `bash -n scripts/test-baseline-update-resolver.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/preflight-migration-baseline-update.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase56-resolver-baseline-refresh-controls.md`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Review Notes
- Approved to close resolver baseline governance loop with opt-in refresh controls.

## Implementation Summary
- Extended shared changed-baselines resolver normalization to support `resolver`
  token.
- Added resolver refresh controls to preflight/update workflow:
  - `CHAINPULSE_MIGRATION_BASELINE_RESOLVER_TEST_BASELINE_FILE`
  - `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE` (default `false`)
- Update script now supports optional resolver baseline refresh by copying:
  - `build/migration-governance/baseline-resolver-test.prom`
  - to resolver baseline file target.
- Extended changelog quality gate default allowlist to:
  - `kpi,health,smoke,resolver`
- Extended baseline governance diff alignment to include resolver baseline file
  and expected changed-baselines tagging.
- Kept scope semantics unchanged and treated resolver-only changed-baselines as
  scope-orthogonal in quality check compatibility logic.
- Updated CI policy env and operations/architecture docs.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase56-resolver-baseline-refresh-controls.md`
- `bash -n scripts/lib/baseline_update_resolver.sh`
- `bash -n scripts/preflight-migration-baseline-update.sh`
- `bash -n scripts/update-migration-governance-baseline.sh`
- `bash -n scripts/check-migration-changelog-quality.sh`
- `bash -n scripts/check-migration-baseline-governance.sh`
- `bash -n scripts/test-baseline-update-resolver.sh`
- `./scripts/test-baseline-update-resolver.sh`
- `./scripts/preflight-migration-baseline-update.sh`
- `CHAINPULSE_REFRESH_BASELINE_RESOLVER_TEST_BASELINE=true ./scripts/preflight-migration-baseline-update.sh`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
