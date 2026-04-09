Title: M2 Monolithic Indexing Storage Profile Selection
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/application/bootstrap, cmd/monolithic/chainpulse

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2-1` and `M2-2` established deployment-mode parsing and a cmd-layer adapter
profile seam, but the profile does not yet drive any real adapter choice. The
current monolithic indexing storage path still always constructs the same
in-memory database/cache pair, so `M2` has not yet crossed from "profile exists"
into "profile affects runtime wiring".

## Scope

This slice will:

1. Make monolithic indexing storage selection depend on deployment mode.
2. Keep `monolithic` mode on the current monolithic memory database/cache pair.
3. Let `microservice` intent switch the indexing database selection to a
   different compatible database implementation while preserving the block
   snapshot seam needed by monolithic reorg handling.
4. Keep cache selection stable for now if the current core-compatible contract
   does not support a broader cutover yet.
5. Add focused tests for the new storage profile selection.

## Non-Goals

This slice will not:

1. complete all adapter switching for `M2`
2. replace monolithic query runtime wiring with production microservice adapters
3. change microservice entrypoints
4. claim full dual-mode parity

## Selected Approach

Introduce a deployment-mode-aware indexing storage selection in bootstrap. Keep
the monolithic memory cache unchanged for now, but let the database selection
change under `microservice` intent by wrapping the existing compatible mock
database with a minimal block snapshot store. This keeps the current monolithic
reorg seam alive while making adapter profile selection affect real runtime wiring.

## Quality Gates

1. `go test -short ./pkg/application/bootstrap/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-indexing-storage-profile-selection.md`

## Decision

Approved for implementation as the third `M2` slice.

## Implementation Notes

Implemented in:

- `pkg/application/bootstrap/indexing_storage.go`
- `pkg/application/bootstrap/indexing_storage_test.go`
- `cmd/monolithic/chainpulse/main_test.go`

Monolithic indexing storage is now deployment-mode-aware: `monolithic` retains
the existing monolithic memory database, while `microservice` intent selects a
different compatible database path and keeps the reorg snapshot seam intact.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./pkg/application/bootstrap/... ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-indexing-storage-profile-selection.md`
