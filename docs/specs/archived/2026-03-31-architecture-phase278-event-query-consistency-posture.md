# Phase 278 - Event Query Consistency Posture

## Status
Status: Approved

## Summary

Add a compact consistency posture to event query response `meta` so callers can
scan query trust posture without interpreting path, fallback, and coverage
fields individually.

## Problem

After phase 277, event query responses exposed a compact execution summary, but
clients still lacked a dedicated field that answered the more direct question:

- what consistency/trust posture should I infer from this response path?

## Decision

Extend event query response `meta` with:

- `consistencyPosture`

Current posture values include:

- `domain-direct`
- `fallback-served`
- `retrieval-partial`
- `retrieval-complete`
- `retrieval-metadata-missing`
- `empty-result`

## Scope

In scope:

- event query response meta enrichment
- handler-level tests
- gateway runtime route coverage

Out of scope:

- query execution changes
- new query endpoints
- stronger storage-level consistency guarantees

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-03-31-architecture-phase278-event-query-consistency-posture.md`

## Exit Criteria

- Event query responses expose a compact consistency posture.
- The posture is validated across domain-first and retrieval-backed paths.
- Gateway runtime route coverage proves the posture survives composed routing.
