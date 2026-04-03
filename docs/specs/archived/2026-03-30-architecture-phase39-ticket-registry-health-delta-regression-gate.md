# Phase 39 Ticket Registry Health Delta Regression Gate

## Title
Phase 39 - Add ticket registry health baseline delta and PR regression signal workflow

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
- `scripts/compare-ticket-registry-health.sh`
- `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- `.github/workflows/ci.yml`
- `scripts/dev-micro-loop.sh`
- `Makefile`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 38 added HTTP ticket-registry latency/SLO telemetry, but teams still lacked baseline diff automation to quickly identify governance health regressions at PR time.

## Problem Statement
Without baseline comparison, operators must manually inspect raw metrics and can miss regression patterns (fallback increments or SLO violation growth).

## Scope
- Add ticket registry health delta script:
  - `scripts/compare-ticket-registry-health.sh`
  - outputs:
    - `build/migration-governance/ticket-registry-health-delta.tsv`
    - `build/migration-governance/ticket-registry-health-delta.md`
- Add baseline file:
  - `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- Add regression policy mode:
  - `CHAINPULSE_MIGRATION_REGISTRY_HEALTH_DELTA_FAILURE_MODE=warn|enforce`
- Integrate delta generation into:
  - CI `policy-contract` summary
  - local full micro-loop
  - Makefile target

## Non-Goals
- No runtime service behavior change.
- No auto-baseline update automation in CI.

## Options Considered
- Option A: rely on per-run health snapshot only.
- Option B: add baseline delta and explicit regression signals.

## Selected Approach
Choose Option B to improve PR-level detectability and support progressive enforcement (`warn` to `enforce`).

## Data / Contract Impact
- No runtime API contract changes.
- Governance artifact contract extended with delta outputs.

## Risks
- Baseline drift can trigger noisy diffs if not refreshed deliberately.
- Mitigation: explicit baseline file and guarded update workflow.

## Rollback Plan
- Stop invoking delta script in CI/local loops and rely on health snapshot only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase39-ticket-registry-health-delta-regression-gate.md`
- `./scripts/check-migration-changelog-quality.sh`
- `./scripts/compare-ticket-registry-health.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase39-ticket-registry-health-delta-regression-gate.md`
- `./scripts/compare-ticket-registry-health.sh`

## Review Notes
- Approved to make ticket-registry governance regression detection explicit and CI-visible.

## Implementation Summary
- Added health-delta compare script and baseline file.
- Added CI summary append for delta markdown.
- Added local dev-loop and Makefile integration.

## Final Verification
- Health delta artifacts are generated and regression status is reported.
