# Phase 112 Monolithic Runtime Legacy Sink Wiring

## Title
Phase 112 - Switch monolithic shared runtime sink to legacy-backed storage adapter

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
- `pkg/application/bootstrap/indexing_runtime.go`
- `pkg/application/bootstrap/indexing_runtime_test.go`
- `cmd/monolithic/chainpulse/main.go`
- `pkg/services/indexing/legacy_runtime_sink.go`
- `docs/specs/2026-03-30-architecture-phase111-monolithic-inmemory-indexing-storage.md`

## Context
Phases 109-111 established real monolithic event input, a legacy-backed sink
adapter, and real in-memory storage on the monolithic indexing path.

## Problem Statement
The shared runtime still uses a no-op sink in monolithic mode, so shadow batch
processing does not yet exercise real storage semantics through the shared
runtime contract.

## Scope
- Update monolithic shared runtime bootstrap wiring to require database and
  cache plugins.
- Use `LegacyRuntimeSink` as the shared runtime sink in monolithic mode.
- Update monolithic entrypoint wiring to pass the started indexing storage into
  shared runtime construction.
- Add focused tests proving bootstrap wiring no longer uses the no-op sink.

## Non-Goals
- No removal of the legacy indexing path.
- No microservice runtime changes in this phase.
- No reorg, replay, or DLQ behavior changes in this phase.

## Selected Approach
- Keep the sink switch inside bootstrap/runtime wiring only.
- Preserve shadow-mode behavior by continuing to let legacy indexing remain the
  active source of truth.
- Reuse the already-tested `LegacyRuntimeSink` adapter rather than adding new
  sink-specific behavior here.

## Data / Contract Impact
- Monolithic shared runtime now persists through the same in-memory storage
- semantics already active on the legacy indexing path.
- No external API or deployment contract changes.

## Observability
- Preserve existing runtime startup/shutdown logs.
- Continue recording lifecycle gauges; later phases can add dedicated sink-path
  counters once ownership moves further into shared runtime.

## Risks
- Moderate duplication risk because both shadow runtime and legacy path can
  write to the same in-memory storage during this phase.
- Acceptable for now because storage is in-memory and this phase is still
  additive/shadow-oriented.

## Rollback Plan
- Revert bootstrap runtime wiring back to the no-op sink.
- Keep monolithic indexing storage in place.

## Test and Verification Plan
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase112-monolithic-runtime-legacy-sink-wiring.md`
- `go test ./pkg/application/bootstrap/... ./pkg/services/indexing/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase112-monolithic-runtime-legacy-sink-wiring.md`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes
- Approved as the first monolithic runtime wiring step that exercises real
  storage semantics through the shared runtime contract.

## Implementation Summary
- Updated monolithic shared runtime bootstrap wiring to require indexing
  database and cache plugins.
- Switched monolithic shared runtime sink construction from the no-op sink to
  `LegacyRuntimeSink`.
- Updated monolithic entrypoint initialization order so indexing storage starts
  before shared runtime construction.
- Added bootstrap tests for required database validation and sink construction
  failure propagation.

## Final Verification
- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase112-monolithic-runtime-legacy-sink-wiring.md`
- `go test ./pkg/application/bootstrap/... ./pkg/services/indexing/... ./pkg/application/indexing/... ./cmd/monolithic/chainpulse/...`
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`
