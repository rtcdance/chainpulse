# Phase 104 Indexer Runtime Closure Slice

## Title
Phase 104 - Define shared indexer runtime closure slice for monolith and microservices

## Type
- architecture
- indexing
- operations

## Status
- Draft | In Review | Approved | Implemented
Status: Approved

## Delivery Status
Planned

## Owner
platform-team

## Reviewers
- Product Owner (chat request)
- Architecture Lead

## Date
2026-03-30

## Related Modules
- `docs/archive/ARCHITECTURE_v1.md`
- `docs/architecture/ENTERPRISE_INDEXER_GAP_ANALYSIS.md`
- `pkg/services/indexing/`
- `pkg/infrastructure/processing/`
- `pkg/infrastructure/data/`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/microservices/puller/main.go`
- `cmd/microservices/event-processor/main.go`
- `test/integration/`

## Context
The repository already has:

- monolithic and microservice entrypoints
- a strong governance and testing layer
- a partially migrated layered architecture

But the indexing runtime is still not closed around one shared contract. Query
and API migration are ahead of puller/processor/indexer runtime unification.

## Problem Statement
ChainPulse currently supports monolithic debug mode and microservice deployment
mode structurally, but it does not yet prove that both modes share the same
indexing lifecycle and operational guarantees.

This creates four concrete problems:

1. puller and event-processor own too much runtime setup logic directly
2. checkpoint, idempotency, replay, and DLQ concerns are not closed as one
   explicit runtime contract
3. monolith and microservices are not yet behaviorally equivalent by design
4. the project is stronger in governance than in core enterprise indexer flow

## Scope
This slice defines the next implementation batch for **indexer runtime closure**.

Included:

- define a shared indexing runtime contract
- define a shared indexing lifecycle for startup/run/shutdown
- define explicit checkpoint, idempotency, DLQ, and replay boundaries
- define monolith/microservice runtime parity target
- define first integration test target for end-to-end closure

Excluded for this slice:

- full query-service redesign
- broad deployment packaging refactor
- Helm/service-mesh rollout work
- broad observability rewrite outside indexing-critical signals

## Current State Evidence

### Shared API/query bootstrap exists
- `pkg/application/bootstrap/runtime_wiring.go`
- `cmd/monolithic/chainpulse/main.go`
- `cmd/microservices/api-service/main.go`

### Indexing runtime is still split
- `pkg/services/indexing/`
- `pkg/infrastructure/data/data_puller.go`
- `pkg/infrastructure/processing/event_processor.go`
- `cmd/microservices/puller/main.go`
- `cmd/microservices/event-processor/main.go`

### New architecture layers are still thin outside query
- `pkg/domain/`
- `pkg/application/`
- `pkg/adapters/`

### Integration tests exist but are not yet a parity proof
- `test/integration/deployment_mode_integration_test.go`
- `test/integration/monolithic_upstream_downstream_test.go`
- `test/integration/multi_chain_indexing_test.go`

## Objectives

### Objective 1
Create one runtime contract that both monolithic and microservice indexing flows
can compose.

### Objective 2
Make checkpoint, idempotency, DLQ, and replay explicit runtime responsibilities
instead of scattered concerns.

### Objective 3
Add one integration proof that monolith and microservice paths produce
equivalent indexing results for the same input shape.

## Target Runtime Model

The intended flow after this slice:

1. **Source**
   - puller or replay source emits chain events
2. **Normalize**
   - event envelope normalization and validation
3. **Idempotency Gate**
   - duplicate detection by stable event key
4. **Persist**
   - write event store + metadata/checkpoint
5. **Failure Routing**
   - classify retryable vs terminal failures
   - route terminal failures to DLQ
6. **Recovery**
   - replay from checkpoint or DLQ
7. **Expose**
   - query/API reads from the same persisted truth

## Proposed Implementation Structure

### A. Shared indexing runtime package

Introduce a shared runtime orchestration slice under application-facing or
indexing-focused package boundaries that can own:

- lifecycle
- event envelope intake
- idempotency decision
- persistence coordination
- checkpoint advancement
- failure routing

Candidate placement:

- `pkg/application/indexing/`
or
- `pkg/application/runtime/indexing/`

Decision rule:

- prefer application-layer orchestration
- keep domain-neutral contracts inward
- keep infrastructure/plugin implementations outward

### B. Explicit runtime ports

Define interfaces for:

- event source
- event sink / event store writer
- checkpoint store
- idempotency store
- failure router / DLQ writer
- replay source

These should be composable by:

- monolith main
- puller service main
- event-processor main

### C. Runtime lifecycle contract

Each runtime should support:

- `Initialize(ctx)`
- `Start(ctx)`
- `Stop(ctx)`
- `Health()`
- `Status()`

And operational hooks for:

- checkpoint snapshot
- replay trigger
- DLQ depth / failure counters

## Delivery Plan

### Step 1
Design shared runtime interfaces and event envelope shape.

Deliverables:

- runtime contract spec in code
- narrow package boundary proposal
- no behavior change yet

### Step 2
Move idempotency/checkpoint/failure routing decisions behind shared orchestration.

Deliverables:

- shared runtime constructor
- adapter seams for event source/store/checkpoint/DLQ

### Step 3
Wire monolith to the shared indexing runtime.

Deliverables:

- monolith indexing path uses shared orchestration
- existing debug workflow preserved

### Step 4
Wire puller/event-processor to the same shared runtime seams.

Deliverables:

- microservice indexing path uses shared orchestration
- startup code becomes composition-only

### Step 5
Add parity integration proof.

Deliverables:

- one integration test proving equal persisted/indexed outcomes for equivalent
  synthetic input

## Acceptance Criteria

The slice is considered complete when:

1. monolith and microservice indexing paths both instantiate the same shared
   indexing runtime contract
2. checkpoint progression is explicit and testable
3. idempotency behavior is explicit and testable
4. failure classification and DLQ routing are explicit and testable
5. replay entrypoint exists or is stubbed behind the same runtime contract
6. at least one integration test proves mode parity for a representative flow

## Risks

- Risk: over-refactoring package structure before runtime behavior is stabilized
  - Mitigation: keep the first slice orchestration-first, not file-move-first

- Risk: monolith and microservice requirements diverge at operational edges
  - Mitigation: define ports around runtime lifecycle, not process topology

- Risk: DLQ/replay design becomes too broad for one slice
  - Mitigation: allow replay implementation to start as minimal but explicit

## Rollback Plan

If the shared runtime extraction proves too disruptive:

- keep the runtime contract package additive
- wire only monolith first
- leave puller/event-processor on legacy path temporarily
- retain existing integration tests while parity path stabilizes

## Test and Verification Plan

Planning-phase verification:

- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase104-indexer-runtime-closure-slice.md`

Implementation-phase target gates:

- focused unit tests for shared indexing runtime package
- targeted integration test for monolith/microservice parity
- `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Quality Gates

Planning phase:

- `scripts/spec-approval-check.sh docs/specs/2026-03-30-architecture-phase104-indexer-runtime-closure-slice.md`

Implementation phase:

- shared runtime unit tests green
- parity integration test green
- no regression in existing indexing/query integration suite

## Review Notes

- Approved as the next highest-value slice after governance harness closure.
- This phase intentionally shifts focus from shell/governance refinement to
  enterprise indexer runtime closure.
