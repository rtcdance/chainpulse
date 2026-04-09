Title: M2 Cmd Adapter Profile Selection
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, docs

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M2-1` made `DEPLOYMENT_MODE` a real monolithic cmd-layer contract, but the cmd
entrypoint still does not produce an explicit adapter/profile decision from that
mode. The startup path therefore knows the deployment intent, yet does not expose
which adapter profile is currently selected or how complete that selection is.

## Scope

This slice will:

1. Add a cmd-layer monolithic adapter profile resolver derived from deployment mode.
2. Surface the selected adapter profile in startup output and `/runtime/summary`.
3. Make the current boundary explicit:
   - monolithic profile is concrete for the current baseline
   - microservice profile is an intent/seam, not full adapter cutover yet
4. Add focused tests for adapter profile resolution and runtime summary surfacing.

## Non-Goals

This slice will not:

1. fully switch all monolithic adapters to production microservice implementations
2. change service-layer business logic
3. modify all microservice entrypoints
4. claim `M2` completion

## Selected Approach

Add a narrow cmd-layer adapter profile resolver that turns deployment mode into a
stable selection seam. The profile will be operator-visible and test-backed, so
later adapter-factory slices can attach real implementation switching to the same
contract without changing the summary model again.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-cmd-adapter-profile-selection.md`

## Decision

Approved for implementation as the second `M2` slice.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/m2_adapter_profile.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic cmd entrypoint now resolves a deployment-mode-aware adapter profile
and exposes that profile as a runtime/deployment fact. The profile is intentionally
stronger for `monolithic` and intentionally partial for `microservice`, matching
the current real boundary of `M2`.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-cmd-adapter-profile-selection.md`
