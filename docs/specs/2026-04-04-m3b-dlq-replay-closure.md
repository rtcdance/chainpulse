Title: M3b DLQ Replay Closure
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: pkg/application/indexing, pkg/application/bootstrap

## Status

Approved for implementation.

## Problem Statement

The shared indexing runtime can already route failed events into the monolithic
in-memory failure journal and reload them during startup recovery. However, the
current replay seam is still weaker than the blueprint's `§3.2 Indexer`
tolerance posture because it does not expose an explicit manual replay entry
point, does not support operator-scoped replay windows, and does not clear
successfully replayed events from the DLQ journal.

## Scope

This slice will:

1. add a manual shared-runtime replay entry point for one chain
2. support replaying a bounded DLQ window by checkpoint range
3. acknowledge and clear successfully replayed DLQ events when the replay
   source supports it
4. upgrade the monolithic in-memory failure journal to preserve replayable DLQ
   records instead of only append-only events
5. expose a monolithic in-process operator route for manual replay requests
6. add focused runtime, bootstrap, and route tests

## Non-Goals

This slice will not:

1. add a new persistent DLQ backend
2. introduce a new external multi-process CLI replay workflow
3. redesign the legacy multi-chain indexer path
4. change distributed microservice replay orchestration

## Selected Approach

Keep the change additive inside the shared indexing runtime contract:

1. introduce optional replay capabilities for range-based replay and replay
   acknowledgement
2. add `ReplayChainRange(...)` to the shared runtime so callers can trigger a
   manual replay for one chain and one checkpoint window
3. keep startup recovery on the existing `RecoverChain(...)` path, but
   acknowledge replayed events after successful processing when the source
   supports acknowledgement
4. upgrade the monolithic in-memory failure journal to store structured DLQ
   records, filter replay windows, and remove acknowledged entries
5. expose a write-scoped monolithic runtime route that can issue replay
   requests inside the same process, which is necessary because the current
   DLQ journal is intentionally in-memory and process-local

## Risks

1. replay acknowledgement failure could leave already-processed events in the
   in-memory DLQ journal
2. range filtering must stay deterministic for same-block events with cursor
   ordering

## Rollback Plan

1. remove the new manual replay method from the shared runtime
2. fall back to the existing append-only in-memory failure journal behavior
3. keep startup recovery on the previous replay-only flow

## Test Strategy

1. unit test manual range replay success and acknowledgement
2. unit test that acknowledged replayed events are removed from the DLQ source
3. unit test that failed replay acknowledgement is surfaced as an error
4. unit test the monolithic in-memory failure journal range filtering and ack
   behavior
5. unit test the monolithic runtime replay route request/response behavior

## Quality Gates

1. `go test -short ./pkg/application/indexing/... ./pkg/application/bootstrap/...`
2. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m3b-dlq-replay-closure.md`

## Review Notes

Approved as the smallest blueprint-aligned DLQ replay step that materially
improves `§3.2 Indexer` failure tolerance without opening a larger operator
surface project.
