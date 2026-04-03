# Phase 276 - Event Query Metadata Coverage Posture

## Status
Status: Approved

## Summary

Compress raw metadata coverage counts into a compact posture so callers can scan
query result quality without interpreting counts first.

## Problem

After phase 275, event query responses exposed concrete metadata coverage
counts, but callers still needed to interpret those counts themselves to judge
coverage quality.

That meant the data plane exposed facts, but not a compact conclusion.

## Decision

Extend event query response `meta` with:

- `metadataCoveragePosture`

Current posture values:

- `coverage-empty`
- `coverage-missing`
- `coverage-partial`
- `coverage-complete`

## Scope

In scope:

- event query response meta enrichment
- handler-level tests
- gateway runtime route coverage

Out of scope:

- query execution changes
- stronger consistency semantics
- new query endpoints

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase276-event-query-metadata-coverage-posture.md`

## Exit Criteria

- Event query responses expose a compact metadata coverage posture.
- The posture is validated across domain-first and retrieval-backed paths.
- Gateway runtime route coverage proves the posture survives composed routing.
