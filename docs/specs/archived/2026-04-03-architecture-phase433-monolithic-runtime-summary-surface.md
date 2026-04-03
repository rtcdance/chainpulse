Title: Phase 433 Monolithic Runtime Summary Surface
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/monolithic/chainpulse, pkg/application/indexing, pkg/plugins/api, docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md

## Status

Status: Approved

## Context

The repository already has a minimum blueprint-aligned runnable app across
`api-gateway`, `api-service`, `puller`, and `event-processor`. The monolithic
entrypoint still exposes the existing health and metrics surfaces, but it does
not yet surface the shared indexing runtime contract in a compact read-only
summary the way the microservice entrypoints now do.

The monolithic mode already owns the shared indexing runtime skeleton and the
ownership rollout surface. The missing piece is a compact operator-facing
runtime summary that makes the shared indexing lifecycle visible from the
top-level monolithic entrypoint.

## Problem Statement

The monolithic runnable mode still hides the shared indexing runtime state
behind startup logs and rollout metrics. That weakens the repository's
blueprint-aligned "single-process debug" story because the runtime state is not
yet exposed as a simple, structured summary.

## Scope

- expose a compact `/runtime/summary` surface from the monolithic entrypoint
- include shared indexing runtime state and ownership rollout posture
- keep the change additive and non-breaking for the existing runnable baseline

## Non-Goals

- no new indexing execution semantics
- no monolith/microservice rewrite
- no change to the existing ownership rollout behavior
- no new storage or replay behavior

## Options Considered

1. Leave the monolithic runtime surface as-is.
   - Rejected because the shared indexing runtime is already present and should
     be visible in the runnable baseline.
2. Add a brand-new monolithic control plane.
   - Rejected because the current goal is observability and closure, not a new
     writable surface.
3. Add a compact runtime summary provider to the existing gateway integration.
   - Selected because it is the smallest additive step that makes the monolithic
     runtime state discoverable without changing runtime ownership.

## Selected Approach

Wire the monolithic entrypoint's existing API gateway plugin to a runtime
summary provider that reports:

- shared indexing runtime state
- ownership rollout summary
- gateway runtime route posture
- a compact metrics snapshot

This keeps the monolithic mode aligned with the blueprint's emphasis on being a
first-class runnable/debuggable app while preserving current runtime behavior.

## Data / Contract Impact

No breaking runtime contract changes are expected. The new endpoint is additive
and read-only.

## Risks

- The summary can drift from the actual shared runtime if the provider is not
  kept in sync with future indexing lifecycle changes.
- The monolithic summary may still be more skeletal than the microservice
  summaries because it reflects the current state of the shared indexing
  runtime skeleton.

## Rollback Plan

- remove the runtime summary provider wiring from the monolithic gateway
- keep the existing health, metrics, and ownership rollout surfaces unchanged

## Test and Verification Plan

- run focused monolithic runtime summary tests
- run `go test -short ./cmd/monolithic/chainpulse/...`
- run the spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-03-architecture-phase433-monolithic-runtime-summary-surface.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; go test -short ./cmd/monolithic/chainpulse/...`

## Review Notes

- Approved as the smallest additive step to make the monolithic runtime state
  visible as a structured summary.

## Implementation Summary

The monolithic entrypoint now exposes a read-only `/runtime/summary` surface
through the existing gateway plugin integration. The summary is additive and
includes the shared indexing runtime contract, ownership rollout posture,
gateway wiring posture, and a compact metrics snapshot without changing the
existing runnable baseline behavior.

## Final Verification

The following focused verification passed:

- `unset GOROOT; export GOCACHE=/tmp/chainpulse-go-build-cache; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; go test -short ./cmd/monolithic/chainpulse/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-03-architecture-phase433-monolithic-runtime-summary-surface.md`
