# Phase 109 Chain Indexer Shared Runtime Shadow Batch

## Title
Phase 109 - Feed real chain indexer batches into shared runtime in shadow mode

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
- `cmd/monolithic/chainpulse/main.go`
- `docs/specs/2026-03-30-architecture-phase108-monolithic-shared-runtime-additive-wiring.md`

## Context
Phase 108 made the monolith host a shared indexing runtime, but no real event
 batch from the existing indexing path is sent into that runtime yet.

## Problem Statement
Without a real batch handoff, the shared runtime still has no production-adjacent
 input path and cannot begin accumulating checkpoint or idempotency state from
 actual monolithic indexing flows.

## Scope
- Add an optional shared runtime hook to `DefaultChainIndexer`.
- Convert valid chain events into shared runtime envelopes before legacy event
  persistence.
- Wire monolithic chain indexers to the additive shared runtime.
- Keep shared runtime invocation in shadow mode: runtime errors are logged and
  measured, but do not block legacy indexing in this phase.
- Add focused unit tests for shadow batch forwarding and failure isolation.

## Non-Goals
- No removal or rewrite of legacy event storage logic.
- No microservice entrypoint changes in this phase.
- No replay execution, reorg handling, or DLQ fan-out changes yet.

## Selected Approach
- Use a narrow optional batch-runtime interface on `DefaultChainIndexer`.
- Build shared runtime envelopes from existing `core.BlockchainEvent` fields.
- Invoke shared runtime once per batch using only valid same-chain events.
- Treat runtime invocation as additive shadow processing so monolith behavior
  remains stable while contracts are exercised.

## Data / Contract Impact
- Internal `DefaultChainIndexer` contract expands with optional shared runtime
  shadow batch support.
- No external API or persistence contract changes in this phase.

## Observability
- Log shared runtime shadow batch failures with `chain_id` and event count.
- Record a counter for shadow batch forwarding failures.
- SLI intent: shadow runtime forwarding error rate should stay near zero before
  legacy-path ownership is moved.

## Risks
- Low risk because legacy indexing remains source of truth.
- Main risk is envelope conversion drift; keep conversion narrow and test it.

## Rollback Plan
- Remove shared runtime hook from `DefaultChainIndexer`.
- Remove monolith chain indexer wiring to the shared runtime.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase109-chain-indexer-shared-runtime-shadow-batch.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase109-chain-indexer-shared-runtime-shadow-batch.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the smallest real-event adoption step after additive monolithic
  runtime wiring.

## Implementation Summary
- Added an optional shared runtime shadow batch hook to `DefaultChainIndexer`.
- Forwarded valid same-chain events into shared runtime envelopes before legacy
  indexing continues.
- Wired monolithic chain indexers to the additive shared runtime.
- Added tests for shadow batch forwarding and failure isolation.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase109-chain-indexer-shared-runtime-shadow-batch.md`
- `go test ./pkg/services/indexing/... ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
