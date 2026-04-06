Title: M1a Monolithic Reorg Handler Wiring
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/services/reorg, pkg/adapters/indexing

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The monolithic runtime now has a real `Puller -> EventBus -> Indexer` execution path and an indexing-backed query surface, but it still does not wire the existing application-level `reorg.ReorgHandler` into that execution path. This leaves one of the explicit M1a blueprint gaps open: the monolith can ingest and query events, but it still lacks a truthful rollback seam when the same block number reappears with a different block hash.

## Scope

This slice will:

1. Add per-chain application-level `ReorgHandler` ownership to monolithic puller runtime.
2. Persist minimal block snapshots into the monolithic indexing database so `ReorgHandler` can compare known block hashes against stored canonical block state.
3. Run reorg detection on ingested monolithic puller events before updating block snapshots.
4. Trigger rollback through `HandleReorg(...)` when a block hash mismatch is detected for an already-known block number.
5. Surface compact monolithic reorg wiring facts in `/runtime/summary`.
6. Add focused tests for:
   1. block snapshot persistence
   2. reorg detection and rollback
   3. runtime summary reorg surfacing

## Non-Goals

This slice will not:

1. Redesign the puller polling loop.
2. Add chain-specific finality depth tuning beyond a simple monolithic default.
3. Introduce a new reorg storage schema.
4. Claim full production-grade reorg parity for every runtime mode.
5. Change microservice reorg behavior.

## Options Considered

### Option A: Leave `ReorgHandler` disconnected and only document the gap

Pros:

1. No runtime behavior risk.

Cons:

1. Leaves an explicit M1a blueprint gap open.
2. Keeps the monolithic indexing loop unable to express rollback ownership.

### Option B: Add a narrow monolithic reorg seam around existing `ReorgHandler`

Pros:

1. Smallest truthful change that gives the monolith rollback ownership.
2. Reuses the existing application service instead of introducing another reorg mechanism.
3. Fits the current M1a boundary.

Cons:

1. Requires minimal block snapshot persistence in the monolithic in-memory database.
2. Still leaves deeper finality policy work for later milestones.

## Selected Approach

Use Option B.

The monolithic puller runtime will own one `ReorgHandler` per chain. As events arrive, the runtime will compare the current block number and block hash against stored canonical block snapshots. On mismatch, it will invoke `HandleReorg(...)`, then persist the new canonical block snapshot and update the handler's in-memory hash tracker. The monolithic runtime summary will expose that this reorg seam is wired, active, and currently best-effort/in-memory.

## Detailed Design

### Per-chain reorg ownership

The monolithic puller runtime will maintain per-chain state containing:

1. a `ReorgHandler`
2. the last detected reorg block
3. total detected reorg count
4. last reorg error, if any

### Block snapshot persistence

The monolithic in-memory indexing database will support storing a minimal `core.Block` snapshot keyed by block number. For this slice, the snapshot only needs:

1. `Number`
2. `Hash`

This is enough for the current `ReorgHandler` contract to compare stored and incoming hashes.

### Detection flow

For each incoming monolithic event:

1. ask the chain reorg handler to detect whether the event's `BlockNumber` now has a different `BlockHash`
2. if a reorg is detected:
   1. call `HandleReorg(...)`
   2. record runtime counters/state
3. persist the current block snapshot as the new canonical block
4. update the handler's known block hash cache

## Risks

1. The reorg seam could overclaim readiness if block snapshots are not really persisted.
2. Rollback could remove events from the monolithic store while async indexing is still in flight.
3. Current in-memory design still does not represent full finality policy.

## Mitigations

1. Persist real block snapshots into the same monolithic indexing database.
2. Keep the summary wording explicit that this is an in-memory monolithic rollback seam.
3. Add focused tests for mismatch detection and rollback behavior.

## Rollback Plan

1. Remove monolithic `ReorgHandler` ownership from puller runtime.
2. Remove monolithic block snapshot persistence helpers.
3. Remove reorg surfacing from monolithic runtime summary.

## Test Strategy

1. Unit test block snapshot persistence in monolithic memory database.
2. Unit test monolithic runtime reorg detection and rollback.
3. Update monolithic runtime summary test with reorg wiring fields.
4. Run focused monolithic and application package tests.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `go test -short ./pkg/adapters/indexing/... ./pkg/services/reorg/...`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-reorg-handler-wiring.md`

## Decision

Approved for implementation as the fourth M1a runtime-closure slice.

## Implementation Notes

Implemented with:

1. per-chain `reorg.ReorgHandler` ownership inside monolithic puller runtime
2. minimal block snapshot persistence in monolithic in-memory indexing storage
3. event-time reorg detection and rollback handling in monolithic runtime observation flow
4. monolithic `/runtime/summary` reorg wiring posture and counters

Primary changed files:

1. `cmd/monolithic/chainpulse/m1a_runtime_wiring.go`
2. `cmd/monolithic/chainpulse/runtime_summary.go`
3. `pkg/adapters/indexing/monolithic_memory_storage.go`

## Verification Summary

Executed checks:

1. `go test -short ./cmd/monolithic/chainpulse/... ./pkg/adapters/indexing/... ./pkg/services/reorg/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-reorg-handler-wiring.md`

Results:

1. focused monolithic tests passed
2. focused indexing adapter tests passed
3. spec approval check passed
