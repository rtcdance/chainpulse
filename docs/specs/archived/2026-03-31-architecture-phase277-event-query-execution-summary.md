# Phase 277 - Event Query Execution Summary

## Status
Status: Approved

## Summary

Compress event query source/path/fallback/coverage facts into a compact
execution summary so callers can scan query behavior without inspecting every
meta field individually.

## Problem

After phase 276, event query responses exposed:

- source
- query path
- fallback behavior
- metadata coverage posture
- raw coverage counts

That was useful, but still verbose for clients that only needed a compact
execution conclusion.

## Decision

Extend event query response `meta` with:

- `queryExecutionSummary`

Use it to compress the current query path facts into one stable summary string.

## Scope

In scope:

- event query response meta enrichment
- handler-level tests
- gateway runtime route coverage

Out of scope:

- query execution changes
- new query endpoints
- stronger consistency semantics

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase277-event-query-execution-summary.md`

## Exit Criteria

- Event query responses expose a compact execution summary.
- The summary is validated across domain-first and retrieval-backed paths.
- Gateway runtime route coverage proves the summary survives composed routing.
