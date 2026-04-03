# Phase 401 - Shared Execution Control Validator

## Status
Status: Approved

## Summary

Add a shared validator for the already-aligned execution-control envelope and
core control fields so `puller` and `event-processor` no longer lock that
shape with service-local assertions only.

## Problem

Phase 399 extracted a shared envelope/helper writer for the aligned
execution-control response shape.

But test coverage still mostly validates that shape in service-local ways.
That leaves a gap:

- the write path is shared
- the contract verification is still partly duplicated

## Decision

Add a shared validator layer for:

- `RuntimeControlEnvelope`
- `RuntimeControlEnvelopeWithTarget`
- the common control-core fields

Adopt the validator in:

- `puller` runtime-control tests
- `event-processor` runtime-control tests

Keep service-specific route naming and target semantics outside the shared
validator unless they are already part of the explicit envelope contract.

## Scope

In scope:

- shared execution-control validator helpers
- focused validator tests
- puller and event-processor test adoption

Out of scope:

- runtime-control route redesign
- new control actions
- target-semantics normalization

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase401-shared-execution-control-validator.md`

## Exit Criteria

- The aligned execution-control contract is validated through shared helpers.
- `puller` and `event-processor` still preserve their intentional differences
  while relying on one common contract validator for the aligned layer.

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...` passed
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...` passed
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase401-shared-execution-control-validator.md` passed
