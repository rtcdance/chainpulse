# Phase 116 Monolithic Ownership Summary

## Title
Phase 116 - Surface aggregated ownership summary in monolithic service output

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
- `docs/specs/2026-03-30-architecture-phase115-indexer-ownership-status-counters.md`

## Context
Phase 115 made ownership visible at the chain-indexer status level, but the
monolithic service output still does not summarize that ownership split across
the whole service.

## Problem Statement
Without a service-level ownership summary, operators and developers still need
to inspect per-chain status maps to understand how much write ownership has
shifted to shared runtime overall.

## Scope
- Add a helper that aggregates `shadow_owned_events` and `legacy_owned_events`
  across all registered chain indexers.
- Print the aggregated ownership summary in monolithic running output.
- Print the final aggregated ownership summary again during graceful shutdown.
- Add focused unit tests for the aggregation helper.

## Non-Goals
- No change to ownership semantics.
- No new external endpoints or APIs.
- No microservice changes.

## Selected Approach
- Keep the aggregation logic inside the monolithic entrypoint layer.
- Use only existing indexer status fields.
- Treat missing or malformed fields as zero to keep summary output resilient.

## Data / Contract Impact
- No persistence changes.
- Monolithic console/status output expands with aggregated ownership summary.

## Observability
- Service output now includes:
  - total shadow-owned events
  - total legacy-owned events
  - number of indexed chains considered
- SLI intent: make ownership shift visible from the monolithic process surface
  without needing per-chain inspection.

## Risks
- Low risk; additive summary only.

## Rollback Plan
- Remove the aggregation helper and summary prints.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase116-monolithic-ownership-summary.md`
- `go test ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase116-monolithic-ownership-summary.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the service-level visibility slice after chain-level ownership
  counters were added.

## Implementation Summary
- Added monolithic ownership aggregation helper for per-chain status surfaces.
- Running output and shutdown output now print aggregated shadow-owned and
  legacy-owned event totals plus indexed chain count.
- Added focused tests for aggregation behavior and resilient numeric coercion.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase116-monolithic-ownership-summary.md`
- `go test ./cmd/monolithic/chainpulse/... ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
