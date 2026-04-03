# Phase 43 Baseline Scope Smoke Artifacts and Summary

## Title
Phase 43 - Export machine-readable baseline-scope smoke artifacts and append CI summary

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
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/guides/OPERATIONS_GUIDE.md`
- `docs/INDEX.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 42 added baseline scope smoke scenarios, but outputs were log-only and not easy to consume by dashboards or CI summaries.

## Problem Statement
Without machine-readable artifacts and summary surfacing, smoke-test outcomes are harder to query, compare, and audit in enterprise workflows.

## Scope
- Extend smoke script to export:
  - JSON artifact
  - Prometheus metrics artifact
  - Markdown summary artifact
- Append smoke markdown summary to CI policy-contract job summary.
- Document artifact paths and output env controls.

## Non-Goals
- No runtime service code changes.
- No historical storage backend for smoke results in this phase.

## Options Considered
- Option A: keep stdout-only smoke logs.
- Option B: add structured artifacts + CI summary append.

## Selected Approach
Choose Option B for observability, traceability, and automation friendliness.

## Data / Contract Impact
- Governance artifact contract expanded with baseline scope smoke outputs.
- No API contract or domain model impact.

## Risks
- Artifact schema drift could break downstream consumers.
- Mitigation: keep flat, stable fields and document output paths.

## Rollback Plan
- Remove CI summary append and keep smoke execution only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase43-baseline-scope-smoke-artifacts-and-summary.md`
- `bash -n scripts/smoke-baseline-governance-scope.sh`
- `./scripts/smoke-baseline-governance-scope.sh`
- Verify generated files:
  - `build/migration-governance/baseline-scope-smoke.json`
  - `build/migration-governance/baseline-scope-smoke.prom`
  - `build/migration-governance/baseline-scope-smoke.md`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase43-baseline-scope-smoke-artifacts-and-summary.md`
- `./scripts/smoke-baseline-governance-scope.sh`

## Review Notes
- Approved to improve CI visibility and downstream automation of smoke outcomes.

## Implementation Summary
- Added JSON/Prom/Markdown outputs in baseline scope smoke script.
- Added CI job summary append for smoke report.
- Updated governance and operations docs for artifact paths and env controls.

## Final Verification
- Smoke script passes and exports all artifacts; CI summary integration configured.
