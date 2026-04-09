Title: M2 Deployment Mode Parsing
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, docs

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1a` through `M1c` closed the monolithic baseline, but `M2` requires the cmd
layer to treat deployment mode as a real runtime contract instead of a README-only
environment variable. Right now `DEPLOYMENT_MODE` is documented, but the monolithic
entrypoint does not parse or surface it as a first-class runtime fact.

## Scope

This slice will:

1. Parse `DEPLOYMENT_MODE` in the monolithic cmd entrypoint.
2. Normalize supported values:
   - `monolithic`
   - `microservice`
3. Fall back safely to `monolithic` for unknown values.
4. Surface deployment-mode facts in startup output and `/runtime/summary`.
5. Add focused tests for parsing and summary surfacing.

## Non-Goals

This slice will not:

1. complete adapter switching between monolithic and microservice profiles
2. change service-layer business logic
3. modify microservice entrypoints yet
4. claim `M2` completion

## Selected Approach

Treat `DEPLOYMENT_MODE` parsing as the first real `M2` cmd-layer contract. Keep
the change narrow: normalize and persist deployment mode in configuration, expose
it in startup/runtime surfaces, and make unsupported values fall back safely to
`monolithic`. This creates a stable seam for the later adapter-factory slices
without pretending the whole dual-mode switch is already finished.

## Quality Gates

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-deployment-mode-parsing.md`

## Decision

Approved for implementation as the first `M2` slice.

## Implementation Notes

Implemented in:

- `cmd/monolithic/chainpulse/m2_deployment_mode.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/monolithic/chainpulse/main_test.go`
- `cmd/monolithic/chainpulse/runtime_summary.go`
- `cmd/monolithic/chainpulse/runtime_summary_test.go`

The monolithic entrypoint now parses and normalizes `DEPLOYMENT_MODE`, prints it
at startup, and exposes deployment-mode facts through `/runtime/summary`.

## Verification Summary

The following checks passed after implementation:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m2-deployment-mode-parsing.md`
