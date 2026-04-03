# Phase 118 Ownership Summary Metrics

## Title
Phase 118 - Export monolithic ownership summary and mode through runtime metrics

## Type
- architecture
- indexing

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
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `docs/specs/2026-03-30-architecture-phase117-monolithic-ownership-mode-status.md`

## Context
Phase 117 exposed ownership mode in monolithic console output, but that state is
still not exported into the runtime metrics/report surface.

## Problem Statement
Without metrics for service-level ownership totals and mode, dashboards and
automated checks cannot track ownership rollout state unless operators inspect
console output directly.

## Scope
- Export aggregated shadow-owned event total as a gauge.
- Export aggregated legacy-owned event total as a gauge.
- Export ownership chain count as a gauge.
- Export ownership mode as a numeric gauge code with stable mapping.
- Emit these metrics at running-summary time and shutdown-summary time.
- Add focused tests for metric emission and mode-code mapping.

## Non-Goals
- No new external dashboard wiring.
- No changes to ownership semantics.
- No microservice changes.

## Selected Approach
- Keep ownership summary metric emission in monolithic entrypoint helper code.
- Use a simple mode-to-code mapping:
  - `idle=0`
  - `legacy-only=1`
  - `shadow=2`
  - `runtime-owned=3`
  - `unknown=9`

## Data / Contract Impact
- Monolithic runtime metrics expand with:
  - `indexing_runtime_shadow_owned_events`
  - `indexing_runtime_legacy_owned_events`
  - `indexing_runtime_ownership_chains`
  - `indexing_runtime_ownership_mode_code`

## Observability
- Ownership rollout state is now visible in runtime metrics, not only terminal
  output.
- This prepares the project for later dashboard and health/report integration.

## Risks
- Low risk; additive gauges only.

## Rollback Plan
- Remove ownership summary metric emission and related tests.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase118-ownership-summary-metrics.md`
- `go test ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase118-ownership-summary-metrics.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first runtime-metrics surface for service-level ownership
  rollout state.

## Implementation Summary
- Added runtime metric emission for aggregated ownership totals and ownership
  mode code in monolithic mode.
- Running and shutdown summaries now export:
  - `indexing_runtime_shadow_owned_events`
  - `indexing_runtime_legacy_owned_events`
  - `indexing_runtime_ownership_chains`
  - `indexing_runtime_ownership_mode_code`
- Added focused tests for mode-code mapping and metric emission.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase118-ownership-summary-metrics.md`
- `go test ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
