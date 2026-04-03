# Phase 114 Shadow Write Ownership Metrics

## Title
Phase 114 - Add metrics for shared-runtime-owned writes in monolithic shadow mode

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
- `pkg/services/indexing/chain_indexer.go`
- `pkg/services/indexing/chain_indexer_test.go`
- `docs/specs/2026-03-30-architecture-phase113-shadow-duplicate-write-guard.md`

## Context
Phase 113 added a duplicate-write guard so the monolithic legacy path can skip
duplicate writes after the shared runtime has already persisted the same in-flight
event object.

## Problem Statement
Without explicit ownership metrics, operators and future rollout work cannot
easily see how many writes are being handled by shared runtime versus falling
back to the legacy path.

## Scope
- Add a counter for events whose write ownership is satisfied by shared runtime
  before the legacy path reaches persistence.
- Emit the counter only on successful shadow-owned suppression paths.
- Add focused regression coverage for metric emission behavior.

## Non-Goals
- No change to ownership rules.
- No new external metrics backend integration.
- No replay, reorg, or microservice changes.

## Selected Approach
- Record a counter at the point where `DefaultChainIndexer` consumes a
  shadow-write mark and skips the duplicate legacy write.
- Use stable labels already used elsewhere in monolithic runtime metrics.

## Data / Contract Impact
- Internal metrics contract expands with
  `indexing_runtime_shadow_owned_events_total`.
- No external API or persistence contract changes.

## Observability
- Metric: `indexing_runtime_shadow_owned_events_total`
  labels:
  - `chain_id`
  - `service=monolithic`
  - `operation=shadow_owned_write`
- SLI intent: trend shared-runtime write ownership growth while validating
  fallback remains available.

## Risks
- Low risk; additive metric only.

## Rollback Plan
- Remove the metric emission and related tests.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase114-shadow-write-ownership-metrics.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase114-shadow-write-ownership-metrics.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the next observability slice after duplicate-write suppression.

## Implementation Summary
- Added `indexing_runtime_shadow_owned_events_total` emission when the legacy
  path consumes a shared-runtime shadow write mark and skips the duplicate
  legacy persistence step.
- Added regression coverage for both the shadow-owned success path and the
  legacy fallback path where the metric must remain zero.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase114-shadow-write-ownership-metrics.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
