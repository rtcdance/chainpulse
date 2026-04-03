# Phase 41 Baseline Scope Intent Tagging

## Title
Phase 41 - Enforce changelog scope intent tagging for baseline updates

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
- `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 40 unified dual-baseline governance, but changelog entries still lacked explicit intent tagging to distinguish update scope and support deterministic policy checks.

## Problem Statement
Without explicit scope tags, baseline changes can be ambiguous in audit trails and harder to validate against actual changed files.

## Scope
- Extend changelog entry schema with required scope field:
  - `scope=kpi-only|health-only|dual`
- Add changelog scope policy controls:
  - `CHAINPULSE_MIGRATION_CHANGELOG_SCOPE_ALLOWLIST`
  - `CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE`
- Update baseline refresh script to write scope automatically:
  - default auto-derived:
    - `dual` when health baseline refresh is enabled
    - `kpi-only` when health baseline refresh is disabled
  - optional override:
    - `CHAINPULSE_BASELINE_UPDATE_SCOPE`
- Extend baseline governance check to enforce scope alignment with baseline diff:
  - `CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=true|false`

## Non-Goals
- No runtime service behavior change.
- No migration to external ticketing system metadata model.

## Options Considered
- Option A: keep free-form rationale-only changelog semantics.
- Option B: add explicit scope field and enforce diff alignment.

## Selected Approach
Choose Option B for stronger enterprise auditability and machine-verifiable intent.

## Data / Contract Impact
- Governance changelog format contract expanded with `scope=...`.
- Existing legacy format can be temporarily tolerated only when scope requirement is disabled.

## Risks
- Existing entries without scope can fail strict checks.
- Mitigation: backfill historical entries and keep opt-out toggle for emergency migration windows.

## Rollback Plan
- Temporarily set `CHAINPULSE_MIGRATION_CHANGELOG_REQUIRE_SCOPE=false`.
- Temporarily set `CHAINPULSE_MIGRATION_ENFORCE_BASELINE_SCOPE_ALIGNMENT=false`.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase41-baseline-scope-intent-tagging.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`
- Validate override paths:
  - `CHAINPULSE_BASELINE_UPDATE_SCOPE=dual CHAINPULSE_ALLOW_BASELINE_UPDATE=true ./scripts/update-migration-governance-baseline.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase41-baseline-scope-intent-tagging.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/check-migration-baseline-governance.sh`

## Review Notes
- Approved to enforce explicit baseline update intent and improve audit traceability.

## Implementation Summary
- Added scope parsing/allowlist checks to changelog quality gate.
- Added scope auto-write and optional override in baseline refresh workflow.
- Added baseline-diff vs scope alignment enforcement in governance check.
- Backfilled existing changelog entry to include scope tag.

## Final Verification
- Changelog quality and baseline governance checks pass with scope-tagged changelog entries.
