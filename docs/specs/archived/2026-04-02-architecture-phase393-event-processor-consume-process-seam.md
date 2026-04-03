# Phase 393 - Event Processor Consume Process Seam

## Status
Status: Approved

## Summary

Add a minimal consume/process ownership seam to the `event-processor`
microservice by wiring Kafka message consumption into the local processor
runtime without overstating the resulting control-plane maturity.

## Problem

After phase 391 and phase 392, `event-processor` now owns:

- processor lifecycle
- idempotency lifecycle
- runtime visibility for processor health and counters

But it still does not own a real consume/process bridge in the microservice
entrypoint. That keeps execution ownership stronger than before, yet still too
abstract:

- Kafka lifecycle exists
- processor lifecycle exists
- the seam between them is still missing

Without that seam, future control-plane work remains blocked on an execution
boundary that is more documented than implemented.

## Decision

Add a small consume/process seam that:

- starts topic-scoped consume loops from the microservice entrypoint
- decodes consumed payloads into `core.BlockchainEvent`
- passes decoded events into the local processor runtime
- records compact runtime facts about configured topics, active topics, and
  last consume-loop error

Do **not** treat this as full execution-control readiness. This is ownership
and visibility work first.

## Scope

In scope:

- event-processor consume/process bridge wiring
- compact consume-loop runtime state
- runtime summary/readiness integration for consume-loop ownership facts
- focused tests for the bridge and runtime state

Out of scope:

- writable control routes for event-processor
- Kafka rebalance redesign
- durable offset orchestration redesign
- processed-event publish pipeline redesign

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...`
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase393-event-processor-consume-process-seam.md`

## Exit Criteria

- `event-processor` owns a real consume-to-processor seam in its entrypoint.
- runtime surfaces can report consume-loop ownership facts honestly.
- the service is better positioned for a future writable-control decision
  without claiming that decision has already been made.

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./cmd/microservices/event-processor/...` passed
