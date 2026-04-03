# Phase 403 - Execution Control Target Alignment

## Status
Status: Approved

## Summary

Align execution-control target metadata across `puller` and
`event-processor` so both writable control pilots expose a target-aware
contract instead of treating target metadata as an `event-processor`-only
detail.

## Problem

Phases 399 through 402 already aligned and shared:

- the execution-control envelope/core write path
- the execution-control envelope/core validator

But one important contract detail still drifts:

- `event-processor` exposes explicit `target` metadata
- `puller` still responds without target metadata

That means the current contract can say that both services are controllable,
but only one of them directly identifies the execution seam being controlled.

## Decision

Promote target metadata into the aligned execution-control contract by:

- defining shared target constants for the currently supported pilots
- switching `puller` runtime-control responses onto the target-aware envelope
- validating both services through the target-aware shared validator

Keep route naming and service-specific action semantics unchanged.

## Scope

In scope:

- shared runtime-control target constants
- `puller` target-aware runtime-control responses
- focused shared and service-local test updates
- documentation refresh for the aligned contract

Out of scope:

- route renaming
- new control actions
- full shared control normalization

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase403-execution-control-target-alignment.md`

## Exit Criteria

- Both writable execution-control pilots expose explicit target metadata.
- Shared tests and service tests validate target-aware envelopes for both
  services.
- The repository records target metadata as part of the aligned control
  contract rather than a service-specific exception.
