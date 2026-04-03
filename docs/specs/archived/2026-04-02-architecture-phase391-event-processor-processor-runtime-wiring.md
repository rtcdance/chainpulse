# Phase 391 - Event Processor Processor Runtime Wiring

## Status
Status: Approved

## Summary

Wire a real processor runtime lifecycle into the `event-processor`
microservice entrypoint without pretending that the service already owns a
fully wired consume/process execution loop.

## Problem

The current `event-processor` service owns:

- Kafka runtime lifecycle
- event-store and metadata-store lifecycle
- rollout health and runtime summary surfaces

But it still does not own an actual processor runtime lifecycle in the
microservice entrypoint. That leaves a gap between:

- the service's execution-oriented identity
- the runtime surfaces it exposes to operators

At the same time, `Phase 389` explicitly recorded that writable control should
not be copied into `event-processor` until execution ownership becomes more
real.

## Decision

Add a small and honest processor-runtime slice that:

- initializes and starts a local processor runtime lifecycle
- initializes and starts the idempotency service that the processor depends on
- stops both cleanly during shutdown
- exposes processor runtime health and counters through runtime summary and
  readiness/component details

Do **not** claim that this means `event-processor` already owns a full
consume/process loop. The slice is lifecycle-first and visibility-first.

## Scope

In scope:

- processor lifecycle wiring in `cmd/microservices/event-processor/main.go`
- runtime-state integration for processor health and counters
- runtime summary exposure for processor lifecycle state
- focused tests for the new runtime wiring

Out of scope:

- writable control routes for `event-processor`
- Kafka consume-to-processor execution-loop redesign
- durable or distributed processor coordination

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase391-event-processor-processor-runtime-wiring.md`

## Exit Criteria

- `event-processor` owns a real processor runtime lifecycle in its entrypoint.
- runtime surfaces can report processor lifecycle state without overstating
  execution ownership.
- the service is better positioned for a future control-plane reopen without
  violating the `Phase 389` no-go boundary.

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...` passed
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase391-event-processor-processor-runtime-wiring.md` passed
