Title: M2 Monolithic Query Surface Profile Selection
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, docs

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2-2` and `M2-3` established deployment-mode parsing plus a first real
storage-side adapter selection, but the monolithic query surface still always
cuts over to the indexing-backed query path. That means the adapter profile
still does not influence the query/runtime read surface, and the deployment
summary can drift from the actual selected query wiring.

## Scope

This slice will:

1. Make monolithic query surface selection depend on deployment mode.
2. Keep `monolithic` mode on the indexing-backed query/runtime path introduced
   in `M1a`.
3. Let `microservice` intent keep the managed-db/shared runtime query path
   instead of forcing the indexing-backed cutover.
4. Align runtime/deployment summary fields with the real selected query adapter.
5. Add focused tests for query surface profile selection and runtime summary
   surfacing.

## Non-Goals

This slice will not:

1. claim full dual-mode parity
2. replace monolithic transport/runtime boundaries with real microservice RPC
3. change microservice entrypoints
4. complete all remaining `M2` adapter-factory work

## Selected Approach

Introduce a narrow monolithic query-surface resolver in the cmd layer. The
resolver will choose between:

- the current indexing-backed query path for `monolithic`
- the existing managed-db/shared runtime query path for `microservice` intent

This keeps `M2` honest: deployment mode now drives a second real runtime
selection without pretending the whole application has already completed the
mode split.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-query-surface-profile-selection.md`

## Decision

Approved for implementation as the fourth `M2` slice.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/m1a_query_wiring.go`
- `cmd/monolithic/chainpulse/m2_adapter_profile.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic cmd path now resolves a deployment-mode-aware query surface. The
selected query adapter is surfaced consistently through both startup/deployment
facts and runtime summary output.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-monolithic-query-surface-profile-selection.md`
