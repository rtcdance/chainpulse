Title: M1a Monolithic EventBus Puller Indexer Wiring
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/core, pkg/plugins/pullers, pkg/services/indexing

## Status

Implemented.

## Problem Statement

The monolithic entrypoint already initializes shared indexing storage, shared indexing runtime, and per-chain indexers, but it still lacks the actual execution path that moves blockchain events through the monolith. In its current state there is no real `EventBus`, no per-chain HTTPS puller startup, and no `blockchain-events` subscription that routes events into the registered chain indexers. This leaves M1a incomplete because the monolith has indexing components without a live event flow.

## Scope

This slice will:

1. Create a monolithic `EventBus` in the monolith bootstrap path.
2. Subscribe the `blockchain-events` topic to the existing `MultiChainIndexer`.
3. Create one HTTPS JSON-RPC puller per configured chain and node URL.
4. Start and stop those pullers with monolithic lifecycle ownership.
5. Fix HTTPS puller event mapping so emitted events carry the configured chain ID and block hash.
6. Add focused tests for monolithic event routing and puller event mapping.

## Non-Goals

This slice will not:

1. Unify query runtime storage with monolithic indexing storage.
2. Add reorg handling to the monolithic pull path.
3. Add new control-plane or health endpoints for pullers.
4. Add backpressure, retries, or checkpoint redesign beyond existing puller behavior.
5. Change microservice execution paths.

## Options Considered

### Option A: Build a new monolithic polling loop from scratch

Pros:

1. Full control over loop behavior.

Cons:

1. Duplicates existing puller logic.
2. Expands scope beyond M1a.
3. Increases drift from microservice-capable puller components.

### Option B: Reuse existing HTTPS puller plugin and wire it into the monolith

Pros:

1. Smallest change that creates a real execution path.
2. Preserves adapter and bootstrap boundaries.
3. Keeps monolith and microservice pull semantics closer.

Cons:

1. Requires careful lifecycle management for goroutines and shutdown.
2. Exposes existing puller mapping gaps that must be fixed.

## Selected Approach

Use Option B.

The monolith will instantiate a shared `EventBus`, subscribe `blockchain-events` to a lightweight handler that type-checks and routes events into `MultiChainIndexer`, and create one HTTPS puller per configured chain/node pair. Pullers will be owned by monolithic bootstrap and run under a shared context so shutdown remains graceful. The HTTPS puller mapper will be corrected to emit the configured chain ID and block hash.

## Detailed Design

### Monolithic wiring

1. Parse configured chains and node URLs.
2. Build a shared `EventBus`.
3. Register a single event-bus subscriber on `blockchain-events`.
4. The subscriber will:
   1. assert the payload is a `core.BlockchainEvent`
   2. copy it to an addressable value
   3. route it into `MultiChainIndexer.IndexEventsFromChain(...)`
5. Build one `HTTPSJSONRPCPuller` per chain/node pair with chain-specific `core.Config`.
6. Start all pullers before waiting on shutdown.
7. Stop all pullers during shutdown.

### Puller mapping correction

The HTTPS puller will:

1. derive `ChainID` from configured service identity for the monolithic wiring slice
2. populate `BlockHash` from the JSON-RPC log payload

This keeps the change additive without redesigning the shared config model in this slice.

## Risks

1. Goroutine leaks if pullers are started without shared context shutdown.
2. Event routing silently failing if type assertions are loose.
3. Chain/node mismatch if chain and node list lengths diverge.

## Mitigations

1. Use a shared cancellable context for puller loops.
2. Fail monolithic startup when configured chains and node URLs cannot be aligned.
3. Add focused tests for:
   1. event bus to indexer routing
   2. puller event mapping

## Rollback Plan

1. Remove monolithic puller startup and subscriber wiring.
2. Revert HTTPS puller chain ID/block hash mapping changes.
3. Keep shared runtime and chain indexers in their current passive state.

## Test Strategy

1. Unit test monolithic event-bus subscriber routing into registered chain indexers.
2. Unit test HTTPS puller `logToEvent` mapping for:
   1. configured chain ID
   2. block hash propagation
3. Run focused monolithic and puller package tests.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `go test -short ./pkg/plugins/pullers/...`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-03-m1a-monolithic-eventbus-puller-indexer-wiring.md`

## Review Notes

1. This spec is intentionally limited to the first executable M1a slice.
2. Query-storage unification and reorg handling remain explicit follow-up work.

## Decision

Approved for implementation as the first M1a runtime-closure slice.

## Implementation Notes

Implemented with:

1. monolithic event-bus creation and `blockchain-events` subscription
2. per-chain HTTPS puller creation aligned to configured chains and node URLs
3. monolithic puller lifecycle start/stop ownership
4. HTTPS puller event mapping updates for configured chain ID and block hash propagation

Primary changed files:

1. `cmd/monolithic/chainpulse/main.go`
2. `cmd/monolithic/chainpulse/m1a_runtime_wiring.go`
3. `pkg/plugins/pullers/https_jsonrpc_puller.go`

## Verification Summary

Executed checks:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `go test -short ./pkg/plugins/pullers/...`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-03-m1a-monolithic-eventbus-puller-indexer-wiring.md`

Results:

1. focused monolithic tests passed
2. focused puller tests passed
3. spec approval check passed
