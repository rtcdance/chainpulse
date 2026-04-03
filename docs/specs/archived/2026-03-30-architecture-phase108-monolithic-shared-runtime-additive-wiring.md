# Phase 108 Monolithic Shared Runtime Additive Wiring

## Title
Phase 108 - Add additive monolithic wiring for shared indexing runtime

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
- `cmd/monolithic/chainpulse/main.go`
- `pkg/application/bootstrap/indexing_runtime.go`
- `pkg/application/bootstrap/indexing_runtime_test.go`
- `pkg/application/indexing/runtime.go`
- `docs/specs/2026-03-30-architecture-phase104-indexer-runtime-closure-slice.md`
- `docs/specs/2026-03-30-architecture-phase107-shared-indexing-runtime-read-paths.md`

## Context
Phases 105-107 introduced the first shared indexing runtime contract plus
write/read helper behavior, but the monolithic entrypoint still does not build
or run that shared runtime.

## Problem Statement
Without additive monolith wiring, the shared runtime remains isolated test
scaffolding instead of a real deployment-mode contract that can be evolved
toward monolith/microservice parity.

## Scope
- Add a bootstrap helper that builds a monolithic shared indexing runtime using
  in-memory/no-op ports.
- Initialize and start that shared runtime from the monolithic entrypoint.
- Expose shared runtime state through startup/shutdown logs and metrics.
- Stop the shared runtime during graceful shutdown.
- Add focused unit tests for bootstrap helper behavior.

## Non-Goals
- No replacement of existing `MultiChainIndexer` logic.
- No real puller/processor/storage integration in this phase.
- No microservice entrypoint wiring in this phase.
- No new external dependencies or deployment configuration changes.

## Selected Approach
- Keep the shared runtime wiring additive and isolated from existing runtime
  behavior.
- Use in-memory/no-op runtime ports so the monolith can host the shared runtime
  lifecycle without forcing external integration changes yet.
- Keep bootstrap responsible for object construction only; keep lifecycle calls
  in `cmd/monolithic`.

## Data / Contract Impact
- Internal monolithic startup contract expands to include shared indexing
  runtime lifecycle management.
- No API, storage, or external deployment contract changes in this phase.

## Observability
- Emit startup/shutdown logs with shared runtime state and chain list.
- Record simple lifecycle gauges so operators can see whether additive runtime
  wiring is active in monolith mode.
- SLI intent for later phases: monolithic shared runtime should stay in
  `running` state while service process is healthy.

## Risks
- Low risk because the shared runtime uses isolated no-op/in-memory ports.
- Main risk is startup noise or shutdown ordering confusion; keep wiring
  explicit and lightweight.

## Rollback Plan
- Remove monolithic shared runtime bootstrap helper and related `main.go`
  lifecycle calls.
- Monolith falls back to current behavior with legacy runtime only.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase108-monolithic-shared-runtime-additive-wiring.md`
- `go test ./pkg/application/bootstrap/...`
- `go test ./pkg/application/indexing/...`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase108-monolithic-shared-runtime-additive-wiring.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first entrypoint-level adoption step after shared runtime
  contract scaffolding.

## Implementation Summary
- Added monolithic bootstrap helper for additive shared indexing runtime
  construction using no-op/in-memory ports.
- Wired shared indexing runtime lifecycle into monolithic startup and shutdown.
- Added startup/shutdown metrics and structured logs for runtime state.
- Added focused bootstrap tests for validation, constructor error propagation,
  and lifecycle-ready success path.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase108-monolithic-shared-runtime-additive-wiring.md`
- `go test ./pkg/application/bootstrap/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
