Title: M1a Monolithic Chain Route Contract Alignment
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/plugins/api, cmd/monolithic/chainpulse, pkg/adapters/indexing

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

The monolithic query surface now reads core `/events` paths from indexing-backed storage, but `/events/chain/{chainId}` still keeps a legacy numeric-only contract. That mismatches the current monolithic chain configuration, which uses string chain identifiers like `ethereum` and `polygon`, and leaves M1a with a visible contract break in one of the main read routes.

## Scope

This slice will:

1. Update the event query handler so `/events/chain/{chainId}` can use string chain IDs when a domain-backed query service is available.
2. Preserve the current numeric retrieval fallback for existing non-monolithic paths.
3. Add focused tests for:
   1. string chain IDs through the domain-backed monolithic path
   2. existing numeric path compatibility

## Non-Goals

This slice will not:

1. Redesign the retrieval service chain API.
2. Change GraphQL or other protocol contracts.
3. Add reorg-aware chain query semantics.
4. Rewrite the broader query metadata model.

## Options Considered

### Option A: Redesign retrieval service to support string chain IDs everywhere

Pros:

1. Stronger long-term alignment.

Cons:

1. Too large for current M1a boundary.
2. Risks broader compatibility drift.

### Option B: Make the handler string-aware when a domain query service is present

Pros:

1. Minimal change.
2. Solves the monolithic contract gap directly.
3. Preserves existing numeric fallback behavior.

Cons:

1. Leaves retrieval service interface unchanged for now.

## Selected Approach

Use Option B.

The handler will prefer the domain-backed query path for string chain IDs. Numeric chain IDs will keep existing behavior. This lets monolithic use real configured chain names without broadening scope into a full query interface redesign.

## Risks

1. Domain query path could accidentally change numeric behavior.
2. Empty-result semantics could diverge between string and numeric paths.

## Mitigations

1. Keep numeric path tests intact.
2. Add a focused string-chain route test over the monolithic query surface.

## Rollback Plan

1. Revert handler string-chain branch.
2. Restore numeric-only `/events/chain/{chainId}` behavior.

## Test Strategy

1. Update event query handler tests for string chain path support.
2. Add monolithic query surface test for `/events/chain/ethereum`.
3. Run focused monolithic and API package tests.

## Quality Gates

1. `go test -short ./pkg/plugins/api ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-chain-route-contract-alignment.md`

## Decision

Approved for implementation as the third M1a runtime-closure slice.

## Implementation Notes

Implemented with:

1. string-aware `/events/chain/{chainId}` handler flow when a domain query service is available
2. preserved numeric retrieval fallback for existing non-monolithic paths
3. monolithic query surface tests for `/events/chain/ethereum`

Primary changed files:

1. `pkg/plugins/api/event_query_handler.go`
2. `pkg/plugins/api/event_query_handler_test.go`
3. `cmd/monolithic/chainpulse/m1a_query_wiring_test.go`

## Verification Summary

Executed checks:

1. `go test -short ./pkg/plugins/api ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-chain-route-contract-alignment.md`

Results:

1. focused API tests passed
2. focused monolithic tests passed
3. spec approval check passed
