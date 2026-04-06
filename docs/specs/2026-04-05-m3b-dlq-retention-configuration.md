Title: M3b DLQ Retention Configuration
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/application/bootstrap, pkg/application/indexing, cmd/monolithic/chainpulse

## Status

Approved for implementation.

## Problem Statement

The monolithic shared indexing runtime now supports in-memory DLQ replay, but
the failure journal has no retention policy. Failed events can accumulate
indefinitely in process memory, which is below the intended operational
baseline for a bounded DLQ surface.

## Scope

This slice will:

1. add retention support to the monolithic in-memory DLQ journal
2. make retention configurable from the monolithic startup environment
3. expire DLQ records based on journal insertion time rather than event
   timestamps
4. add focused tests for expiry and config parsing
5. update operator-facing docs for the new retention setting

## Non-Goals

This slice will not:

1. add durable DLQ storage
2. implement background sweeper goroutines
3. add retention configuration to every runtime embedding path
4. change replay route semantics beyond expiring old records before replay

## Selected Approach

1. extend the monolithic in-memory failure journal with an optional retention
   duration
2. stamp each DLQ record with the journal insertion time and lazily evict
   expired records during route/replay/ack/size operations
3. add additive builder options for monolithic and in-memory shared runtime
   construction while preserving existing defaults
4. expose a monolithic env var for retention duration parsing using Go
   `time.ParseDuration` semantics
5. keep the default retention enabled with a bounded but operator-overridable
   window

## Risks

1. operators may unexpectedly lose old in-memory DLQ entries if retention is
   configured too aggressively
2. lazy cleanup means expired entries persist until the next journal access
3. invalid duration parsing could block startup if not validated clearly

## Rollback Plan

1. remove retention-aware journal cleanup
2. revert monolithic startup wiring to the previous builder
3. remove the env parsing and docs for DLQ retention

## Test Strategy

1. unit test journal expiry removes stale records before replay
2. unit test journal size reflects retention cleanup
3. unit test monolithic builder forwards configured retention
4. unit test monolithic env parsing accepts valid durations and rejects invalid
   values

## Quality Gates

1. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3b-dlq-retention-configuration.md`
2. `go test -short ./pkg/application/indexing/... ./pkg/application/bootstrap/... ./cmd/monolithic/chainpulse/...`
3. `scripts/dev-micro-loop.sh --mode fast --base HEAD`

## Review Notes

Approved as the smallest bounded-memory improvement for the current in-process
DLQ design without opening a durable DLQ project.
