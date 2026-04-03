# Phase 274 - Event Query Path Meta

## Status
Status: Approved

## Summary

Strengthen the new event query `meta` block by making query path and fallback
behavior explicit to callers.

## Problem

After phase 273, event query responses exposed source and metadata
completeness, but callers still could not clearly tell:

- which path handled the query
- whether a domain-first read had to fall back to retrieval

That left an important piece of query-path observability implicit.

## Decision

Extend event query response `meta` with:

- `queryPath`
- `fallbackUsed`

Use the new fields to distinguish:

- domain-first success
- domain-first fallback to retrieval
- retrieval-backed list reads

## Scope

In scope:

- event query response meta enrichment
- handler-level tests
- gateway runtime route coverage

Out of scope:

- storage/query execution changes
- new query endpoints
- broader query consistency semantics

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase274-event-query-path-meta.md`

## Exit Criteria

- Event query responses expose query path semantics.
- Domain-first fallback is explicitly visible to callers.
- Runtime route coverage proves the new meta shape survives the gateway path.
