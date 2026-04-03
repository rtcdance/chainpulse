# Phase 37 Ticket Registry Health Telemetry

## Title
Phase 37 - Add ticket registry health telemetry and CI summary visibility

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
- `.github/workflows/ci.yml`
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 36 added registry-aware ticket verification and degrade modes, but governance observability still lacked explicit metrics that show registry check health and fallback usage.

## Problem Statement
Without dedicated telemetry for ticket registry verification, operations cannot quickly determine whether governance checks are running in strict mode, silently degrading, or repeatedly falling back.

## Scope
- Extend changelog quality gate to emit ticket registry health outputs:
  - `build/migration-governance/ticket-registry-health.prom`
  - `build/migration-governance/ticket-registry-health.md`
- Add health output env controls:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_OUTPUT`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HEALTH_MD_OUTPUT`
- Export counters:
  - `chainpulse_migration_ticket_registry_checks_total{mode,source,status,failure_mode}`
  - `chainpulse_migration_ticket_registry_fallback_events_total{reason,failure_mode}`
- Append registry health report to CI step summary.

## Non-Goals
- No runtime service code changes.
- No external issue tracker API integration beyond existing registry source modes.

## Options Considered
- Option A: rely only on script stdout logs.
- Option B: export machine-readable metrics and markdown summary artifacts.

## Selected Approach
Choose Option B for deterministic observability, CI artifact consistency, and dashboard integration readiness.

## Data / Contract Impact
- Governance metrics contract extended with ticket-registry health counters.
- No API or domain model contract changes.

## Risks
- Risk: metric cardinality growth if labels become too dynamic.
- Mitigation: fixed label set (`mode`, `source`, `status`, `failure_mode`, `reason`).

## Rollback Plan
- Revert to previous script behavior or ignore new outputs while preserving existing verification checks.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase37-ticket-registry-health-telemetry.md`
- `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=both CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE=file CHAINPULSE_MIGRATION_TICKET_REGISTRY_FILE=docs/operations/MIGRATION_TICKET_REGISTRY.txt CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE=enforce ./scripts/check-migration-changelog-quality.sh`
- Verify generated outputs:
  - `build/migration-governance/ticket-registry-health.prom`
  - `build/migration-governance/ticket-registry-health.md`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase37-ticket-registry-health-telemetry.md`
- `./scripts/check-migration-changelog-quality.sh`

## Review Notes
- Approved to strengthen governance check transparency and CI observability.

## Implementation Summary
- Added ticket registry health counters and markdown output in changelog quality gate.
- Added CI summary append step for ticket registry health report.
- Updated governance dashboard query doc with new metrics.

## Final Verification
- Changelog quality check passes with ticket registry health outputs generated.
