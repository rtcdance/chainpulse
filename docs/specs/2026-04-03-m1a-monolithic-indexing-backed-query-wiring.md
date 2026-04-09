Title: M1a Monolithic Indexing Backed Query Wiring
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/adapters/indexing, pkg/plugins/api, pkg/services/query

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The monolithic runtime now has a real `Puller -> EventBus -> Indexer` execution path, but the monolithic query surface is still primarily wired through `BuildRuntimeWiring()` and its separate database manager adapters. That means the monolithic indexing path and the monolithic query path do not yet read from the same data plane, which keeps M1a incomplete even though events can now be ingested.

## Scope

This slice will:

1. Add a monolithic indexing-backed event store adapter over the monolithic in-memory indexing database.
2. Add a monolithic indexing-backed metadata store adapter with best-effort synthetic metadata.
3. Add a monolithic domain query adapter backed by the same monolithic indexing database.
4. Rewire monolithic bootstrap so:
   1. `/events`
   2. `/events/{id}`
   3. `/events/name/{eventName}`
   4. `/events/contract/{address}`
   read from the indexing-backed monolithic data plane.
5. Surface the query/indexing alignment posture in monolithic runtime summary.

## Non-Goals

This slice will not:

1. Redesign the legacy query service implementation.
2. Unify the generic `BuildRuntimeWiring()` implementation for all runtimes.
3. Fix the existing numeric-only `/events/chain/{chainId}` contract.
4. Add reorg-aware query semantics.
5. Change microservice query paths.

## Options Considered

### Option A: Replace the shared query runtime globally

Pros:

1. Stronger long-term convergence.

Cons:

1. Too large for current M1a scope.
2. Risks destabilizing api-service and gateway behavior outside monolith.

### Option B: Add monolithic-only indexing-backed query adapters and override wiring in monolith bootstrap

Pros:

1. Smallest change that closes the current monolithic data-plane gap.
2. Keeps legacy shared query runtime intact for other entrypoints.
3. Fits the current M1a milestone boundary.

Cons:

1. Leaves global runtime wiring convergence for later work.
2. Requires explicit documentation of remaining `/events/chain/{chainId}` mismatch.

## Selected Approach

Use Option B.

We will add small indexing-backed adapters and wire them only into the monolithic bootstrap. This gives the monolith a truthful read path over the same indexing storage it writes to, without broadening scope into a full query runtime redesign.

## Detailed Design

### Indexing-backed event store

The event store adapter will:

1. read from `core.DatabasePlugin`
2. use `GetEvent`, `GetAllEvents`, and in-memory filtering for the required query surfaces
3. support:
   1. get by event ID
   2. get all events
   3. get by contract
   4. get by event name

### Indexing-backed metadata store

The metadata store adapter will:

1. derive synthetic metadata from the stored event
2. keep metadata reads best-effort and non-blocking
3. avoid introducing a second write path in this slice

### Monolithic bootstrap override

The monolithic entrypoint will:

1. build indexing-backed event retrieval service after indexing storage is initialized
2. build a matching event query handler from that retrieval service
3. build a monolithic domain query adapter from the same indexing storage
4. replace the default runtime-wiring query/event handler references before the gateway is initialized

## Risks

1. The monolithic query path could diverge from existing API response expectations.
2. Synthetic metadata could be incomplete compared to full PostgreSQL-backed metadata.
3. The existing numeric chain-id API contract remains mismatched with string chain configuration.

## Mitigations

1. Keep the scope to handlers already covered by monolithic smoke and focused tests.
2. Mark metadata as best-effort rather than pretending full parity.
3. Explicitly defer `/events/chain/{chainId}` to a later slice.

## Rollback Plan

1. Remove monolithic indexing-backed adapters.
2. Restore monolith to the default runtime wiring query/event handler references.
3. Leave monolithic ingestion closure intact.

## Test Strategy

1. Unit test indexing-backed event store filtering behavior.
2. Unit test monolithic domain query adapter against indexing storage.
3. Unit test monolithic runtime summary alignment posture.
4. Focused monolithic package tests.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `go test -short ./pkg/adapters/indexing/...`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-03-m1a-monolithic-indexing-backed-query-wiring.md`

## Review Notes

1. This is still an M1a slice, not a full query subsystem redesign.
2. The remaining chain-id API mismatch is explicitly deferred.

## Decision

Approved for implementation as the second M1a runtime-closure slice.

## Implementation Notes

Implemented with:

1. indexing-backed monolithic event store adapter
2. synthetic metadata adapter over the same monolithic indexing database
3. indexing-backed monolithic domain query service
4. monolithic bootstrap override so core `/events` read paths use indexing-backed storage
5. monolithic runtime summary query-alignment surfacing

Primary changed files:

1. `pkg/adapters/indexing/monolithic_query_adapters.go`
2. `cmd/monolithic/chainpulse/m1a_query_wiring.go`
3. `cmd/monolithic/chainpulse/main.go`
4. `cmd/monolithic/chainpulse/runtime_summary.go`

## Verification Summary

Executed checks:

1. `go test -short ./pkg/adapters/indexing/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-03-m1a-monolithic-indexing-backed-query-wiring.md`

Results:

1. focused indexing adapter tests passed
2. focused monolithic tests passed
3. spec approval check passed
