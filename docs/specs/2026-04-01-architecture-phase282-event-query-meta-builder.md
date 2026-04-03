# Phase 282 - Event Query Meta Builder

## Status
Status: Approved

## Summary

Extract a shared event query meta input/builder path so single-event, retrieval
list, and domain-list responses derive their externally visible query `meta`
through one stable assembly layer instead of repeating the same derivation
logic across several helper functions.

## Problem

After phase 281, the event/query data plane already had shared response
builders, but query `meta` assembly was still split across:

- single-event meta assembly
- retrieval-list meta assembly
- domain-list meta assembly

Those paths were converging on the same contract fields:

- source posture
- coverage posture
- consistency posture
- reliability hint
- execution summary

But they still re-derived that contract through multiple helper chains.

## Decision

Add a shared event query meta input model and builder, then route the existing
single-event, retrieval-list, and domain-list helpers through it.

Keep domain-list-specific source posture overrides and total-count adjustments,
but move the common contract derivation into one shared meta assembly path.

## Scope

In scope:

- shared event query meta input/builder extraction
- migration of current helper paths onto the builder
- focused builder tests

Out of scope:

- query execution rewrites
- new event query endpoints
- changing externally visible query `meta` semantics

## Validation

- `go test ./pkg/plugins/api/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase282-event-query-meta-builder.md`

## Exit Criteria

- Single-event, retrieval-list, and domain-list query meta paths all flow
  through one shared builder for common contract derivation.
- Package tests remain green without changing existing externally visible query
  semantics.
- Domain-list-specific source posture and total adjustments remain explicit and
  additive on top of the shared builder.
