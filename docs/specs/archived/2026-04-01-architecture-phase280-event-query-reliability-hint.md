# Phase 280 - Event Query Reliability Hint

## Status
Status: Approved

## Summary

Expose a compact reliability hint so event query responses can summarize how
safe it is to rely on the returned result path, source posture, and metadata
coverage without forcing callers to interpret multiple low-level `meta` fields.

## Problem

After phase 279, event query responses exposed:

- query source posture
- query path
- fallback usage
- metadata coverage posture
- consistency posture

This was much more transparent than before, but callers still had to manually
interpret several fields to answer a simple question:

- how cautious should I be when relying on this response?

That gap was especially visible for:

- domain-first fallback responses
- query-service cache-backed list responses
- retrieval responses with partial or missing metadata

## Decision

Extend event query response `meta` with:

- `queryReliabilityHint`

Populate it from the already exposed source and consistency semantics so
callers receive a compact recommendation such as:

- served directly from domain query path without fallback
- served from query-service cache; verify freshness expectations before
  treating as latest
- served through fallback path; verify query-service availability if this
  persists
- served with partial metadata coverage; verify metadata completeness before
  relying on full event context

## Scope

In scope:

- event query response meta enrichment
- handler-level tests
- gateway runtime route coverage

Out of scope:

- query execution rewrites
- new query endpoints
- storage-level consistency guarantees

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase280-event-query-reliability-hint.md`

## Exit Criteria

- Event query responses expose a compact reliability hint.
- Domain-first success, domain-first fallback, retrieval-backed responses, and
  query-service-backed list responses are covered by tests.
- The reliability hint is derived from already exposed source and consistency
  semantics instead of introducing a separate hidden execution path.
