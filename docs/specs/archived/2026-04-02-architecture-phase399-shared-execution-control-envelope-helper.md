# Phase 399 - Shared Execution Control Envelope Helper

## Status
Status: Approved

## Summary

Extract a shared helper for the already-aligned execution-control envelope and
core control fields without forcing route, target, or service semantics into a
premature shared abstraction.

## Problem

Phase 398 recorded that the two execution-control pilots already align on:

- envelope fields:
  - `service`
  - `timestamp`
  - `control`
- core control facts:
  - `paused`
  - `state`
  - `reason`
  - `last_action`
  - `updated_unix`

But that aligned layer is still duplicated in service-local response code.

## Decision

Add a shared helper that covers only:

- the control core field set
- the common runtime-control envelope writer

Keep the following service-local:

- route naming
- action target semantics
- optional extra envelope metadata such as `target`

## Scope

In scope:

- shared control core type/helper
- shared runtime-control response writer
- adoption in `puller` and `event-processor`
- focused tests for the shared helper and both service integrations

Out of scope:

- shared writable-control contract redesign
- route renaming
- target-semantics normalization

## Validation

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api/... ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase399-shared-execution-control-envelope-helper.md`

## Exit Criteria

- The already-aligned execution-control envelope is shared in code.
- `puller` and `event-processor` still preserve their intentional route/target
  differences.
- Future shared-control work can build on one stable envelope layer instead of
  duplicating response assembly.

## Verification Summary

- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test ./pkg/plugins/api ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...` passed
- `GOROOT= GOCACHE=/tmp/chainpulse-go-build-cache go test -race ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...` passed
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase399-shared-execution-control-envelope-helper.md` passed
