# Phase 288 - Cross-Protocol Query Source Assessment

## Status
Status: Approved

## Summary

Record a small cross-protocol assessment after extending query-source
surfacing from the HTTP event/query data plane into the GraphQL `eventsByName`
read path.

## Problem

After phase 287, query-source surfacing now exists in two protocol surfaces:

- HTTP event query routes
- GraphQL `eventsByName`

That is enough to prove the idea is portable, but not yet enough to claim full
cross-protocol parity. Without an explicit assessment, it would be too easy to
either:

- overstate the current GraphQL coverage
- or keep expanding the line without a clear stop point

## Decision

Classify the current state as:

- **cross-protocol source surfacing proven at pilot scope**

That means:

- HTTP remains the strong baseline
- GraphQL now has a credible pilot slice
- further GraphQL or protocol-wide expansion should be treated as an explicit
  new objective rather than automatic continuation

## Scope

In scope:

- cross-protocol assessment for query-source surfacing
- stop-line guidance for the current GraphQL pilot

Out of scope:

- new GraphQL result envelopes
- broad protocol parity claims
- new query execution work

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase288-cross-protocol-query-source-assessment.md`

## Exit Criteria

- The repository contains an explicit assessment for the current
  cross-protocol query-source surfacing state.
- The assessment makes the current GraphQL slice clearly readable as a pilot,
  not as full cross-protocol parity.
