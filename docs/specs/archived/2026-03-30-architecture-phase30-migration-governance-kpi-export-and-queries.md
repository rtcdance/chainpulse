# Phase 30 Migration Governance KPI Export and Queries

## Title
Phase 30 - Add migration governance KPI export and dashboard query templates

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
- `scripts/export-migration-governance-kpi.sh`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 29 hardened migration governance checks, but lacked a standard KPI export and dashboard query set for observability of migration progress.

## Problem Statement
Without KPI export, teams cannot track migration governance health trends across status/severity/domain in dashboards.

## Scope
- Add KPI export script from migration manifest:
  - emits Prometheus text file
  - emits Markdown snapshot
- Add CI step to generate and upload KPI artifacts.
- Add local full micro-loop step for KPI export.
- Add Makefile target for manual KPI export.
- Add dashboard query templates for governance KPIs.

## Non-Goals
- No runtime service metric endpoint changes.
- No external TSDB ingestion automation.

## Options Considered
- Option A: governance checks only, no KPI export.
- Option B: checks + export + dashboard query templates.

## Selected Approach
Choose Option B to operationalize governance visibility.

## Data / Contract Impact
Adds file-based KPI telemetry artifacts under `build/migration-governance/`.

## Risks
- KPI export reflects manifest snapshot, not real-time runtime state.
- Mitigation: run in CI and local loops for continuous refresh.

## Rollback Plan
Remove KPI export step from CI/micro-loop and keep checks-only workflow.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase30-migration-governance-kpi-export-and-queries.md`
- `./scripts/export-migration-governance-kpi.sh`
- `./scripts/check-migration-manifest.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase30-migration-governance-kpi-export-and-queries.md`
- `./scripts/export-migration-governance-kpi.sh`

## Review Notes
- Approved to close migration governance visibility gap.

## Implementation Summary
- Added KPI export artifacts, CI upload integration, and dashboard query templates.

## Final Verification
- Spec gate, manifest checks, and KPI export all pass.
