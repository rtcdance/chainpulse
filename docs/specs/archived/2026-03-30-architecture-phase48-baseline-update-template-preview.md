# Phase 48 Baseline Update Template Preview

## Title
Phase 48 - Emit recommended changelog template preview for governed baseline updates

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
- `scripts/update-migration-governance-baseline.sh`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/guides/OPERATIONS_GUIDE.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 46 introduced strict changelog fields (`scope` + `changed_baselines`), but baseline refresh operators still needed to manually compose compliant changelog entries.

## Problem Statement
Manual changelog composition increases the chance of format or field mismatches, causing avoidable governance check failures.

## Scope
- Extend baseline refresh script to emit recommended changelog template preview:
  - default output:
    - `build/migration-governance/baseline-update-template.md`
- Add preview controls:
  - `CHAINPULSE_EMIT_BASELINE_UPDATE_TEMPLATE_PREVIEW=true|false`
  - `CHAINPULSE_BASELINE_UPDATE_TEMPLATE_OUTPUT`
- Preview content includes:
  - ticket
  - owner
  - resolved scope
  - resolved changed baseline set
  - rationale
  - copy-ready suggested changelog entry format

## Non-Goals
- No runtime service behavior changes.
- No automatic baseline refresh in CI.

## Options Considered
- Option A: keep script stdout-only status lines.
- Option B: emit structured template artifact for operators.

## Selected Approach
Choose Option B to reduce operator error and improve governance workflow usability.

## Data / Contract Impact
- Governance process artifact contract expanded with template preview markdown.
- No API/domain runtime contract impact.

## Risks
- Template could become stale if changelog schema changes.
- Mitigation: template is generated from live resolved fields in the same script.

## Rollback Plan
- Set `CHAINPULSE_EMIT_BASELINE_UPDATE_TEMPLATE_PREVIEW=false`.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase48-baseline-update-template-preview.md`
- `bash -n scripts/update-migration-governance-baseline.sh`
- Run update script with guarded temp baseline/changelog paths and verify template output exists.

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase48-baseline-update-template-preview.md`
- `./scripts/update-migration-governance-baseline.sh` (guarded run)

## Review Notes
- Approved to improve baseline update operability and reduce changelog formatting errors.

## Implementation Summary
- Added template preview output and env controls in baseline update script.
- Added doc references for template artifact path and controls.

## Final Verification
- Guarded baseline update run emits template preview with resolved `scope` and `changed_baselines`.
