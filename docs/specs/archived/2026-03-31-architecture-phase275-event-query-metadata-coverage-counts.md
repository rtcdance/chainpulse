# Phase 275 - Event Query Metadata Coverage Counts

## Status
Status: Approved

## Summary

Strengthen event query response `meta` with concrete metadata coverage counts so
callers can judge response quality without inferring from a coarse label alone.

## Problem

After phase 274, event query responses exposed source, query path, and fallback
behavior, but callers still only saw metadata completeness as a coarse label.

That meant a caller could know a result was `partial`, but not immediately know
how partial it was.

## Decision

Extend event query response `meta` with:

- `metadataAttachedCount`
- `metadataMissingCount`

Use these counts across:

- domain-first single-event reads
- retrieval fallback reads
- retrieval-backed list reads

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
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase275-event-query-metadata-coverage-counts.md`

## Exit Criteria

- Event query responses expose concrete metadata coverage counts.
- Single-event and list responses both report metadata coverage honestly.
- Gateway runtime route coverage proves the counts survive the composed route.
