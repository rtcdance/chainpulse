# Phase 40 Governed Ticket Registry Health Baseline Refresh

## Title
Phase 40 - Govern ticket-registry health baseline refresh with changelog binding

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
- `scripts/check-migration-baseline-governance.sh`
- `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 39 introduced ticket-registry health baseline and delta regression checks, but baseline refresh governance still primarily targeted KPI baseline and did not explicitly enforce dual-baseline audit discipline.

## Problem Statement
If health baseline refresh is not bound to changelog and controlled update workflow, teams can accidentally mute regression signals by direct file edits.

## Scope
- Extend baseline update workflow to refresh ticket-registry health baseline by default:
  - `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- Add baseline refresh controls:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE_FILE`
  - `CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=true|false`
- Extend baseline governance check to require changelog update when either baseline changes:
  - KPI baseline
  - ticket-registry health baseline

## Non-Goals
- No automatic baseline refresh in CI.
- No change to runtime service behavior.

## Options Considered
- Option A: keep ticket-registry baseline outside existing governed update/check flow.
- Option B: unify both baselines under the same controlled refresh and changelog gate.

## Selected Approach
Choose Option B for enterprise auditability and consistent policy-as-code behavior.

## Data / Contract Impact
- No runtime API/data contract changes.
- Governance process contract expanded to dual-baseline enforcement.

## Risks
- Teams may unintentionally skip health baseline refresh in manual workflows.
- Mitigation: default refresh enabled and explicit override flag.

## Rollback Plan
- Disable health baseline refresh with `CHAINPULSE_REFRESH_TICKET_REGISTRY_HEALTH_BASELINE=false`.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase40-governed-ticket-registry-health-baseline-refresh.md`
- `bash -n scripts/update-migration-governance-baseline.sh`
- `bash -n scripts/check-migration-baseline-governance.sh`
- `./scripts/check-migration-baseline-governance.sh` (normal repo run)

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase40-governed-ticket-registry-health-baseline-refresh.md`
- `./scripts/check-migration-baseline-governance.sh`

## Review Notes
- Approved to align baseline refresh governance with enterprise audit controls.

## Implementation Summary
- Baseline update script now supports governed refresh of ticket-registry health baseline.
- Baseline governance check now enforces changelog binding for either baseline file change.
- Operations dashboard doc updated with new refresh controls.

## Final Verification
- Governance scripts pass syntax checks and baseline governance run succeeds in normal repo state.
