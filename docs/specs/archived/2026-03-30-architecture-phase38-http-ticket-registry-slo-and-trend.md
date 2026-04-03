# Phase 38 HTTP Ticket Registry SLO and Trend

## Title
Phase 38 - Add HTTP ticket registry latency SLO policy and trend-ready telemetry

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
- `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- `docs/ARCHITECTURE.md`

## Context
Phase 37 exposed ticket registry health counters, but HTTP-backed registry mode still lacked explicit latency SLO policy and dedicated trend metrics.

## Problem Statement
When registry source is `http`, teams need deterministic controls for slow dependency behavior and visibility into SLO breaches over time, otherwise governance checks can degrade silently.

## Scope
- Add HTTP registry latency capture (ms) to changelog quality gate.
- Add configurable HTTP SLO policy controls:
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MS`
  - `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE=off|warn|enforce`
- Export new telemetry:
  - `chainpulse_migration_ticket_registry_http_latency_ms{source,status,slo_mode}`
  - `chainpulse_migration_ticket_registry_http_slo_violations_total{source,slo_mode}`
- Extend ticket registry health markdown output with latency/SLO fields.

## Non-Goals
- No changes to runtime service traffic path.
- No external time-series storage design in this phase.

## Options Considered
- Option A: rely on timeout-only behavior.
- Option B: add explicit latency metric + SLO policy mode + violation counter.

## Selected Approach
Choose Option B to provide auditable SLO controls and trend-ready metrics while preserving existing verification behavior.

## Data / Contract Impact
- Governance telemetry contract expanded with HTTP latency and SLO violation metrics.
- Existing metric names and labels remain backward compatible.

## Risks
- Enforce mode may block pipelines under transient network latency.
- Mitigation: `warn` default and explicit `off|warn|enforce` switch.

## Rollback Plan
- Set `CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE=off` to disable SLO enforcement and warnings.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase38-http-ticket-registry-slo-and-trend.md`
- `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=both CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE=file CHAINPULSE_MIGRATION_TICKET_VERIFY_FAILURE_MODE=enforce ./scripts/check-migration-changelog-quality.sh`
- HTTP mode smoke test (local registry endpoint):
  - `CHAINPULSE_MIGRATION_TICKET_VERIFY_MODE=both CHAINPULSE_MIGRATION_TICKET_REGISTRY_SOURCE=http CHAINPULSE_MIGRATION_TICKET_REGISTRY_URL=<local-url> CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MS=0 CHAINPULSE_MIGRATION_TICKET_REGISTRY_HTTP_SLO_MODE=warn ./scripts/check-migration-changelog-quality.sh`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase38-http-ticket-registry-slo-and-trend.md`
- `./scripts/check-migration-changelog-quality.sh`

## Review Notes
- Approved to improve enterprise governance resiliency and SLO observability for external registry dependency.

## Implementation Summary
- Added HTTP latency measurement and SLO policy controls.
- Added trend-ready latency/violation metrics.
- Extended registry health markdown output for CI artifact readability.

## Final Verification
- Changelog quality check passes in file mode and HTTP mode smoke tests.
