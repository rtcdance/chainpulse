# Phase 300 - GraphQL Event Query Source Plane Assessment

## Status
Status: Approved

## Summary

Assess the current GraphQL event-family query-source work after resolver and
schema-builder paths both reached cache-aware source parity across the core
event read family.

## Problem

After phase 299, the GraphQL event family now has:

- single-event source surfacing
- root-list source surfacing
- filtered-list source surfacing
- resolver/schema-builder source parity
- cache-hit versus live event-store semantics on the core cached paths

The repository needs an explicit statement of whether this is still only a
mini-baseline or whether it has now become a stronger GraphQL event-query
source plane with a real stop-line.

## Decision

Classify the current GraphQL event family as:

- `stage-complete for the GraphQL event-query source plane baseline`

This means:

- the core GraphQL event family now exposes a stable source plane
- resolver and schema-builder implementations are aligned on that plane
- the baseline is strong enough to pause by default

It does **not** mean:

- full GraphQL parity
- a broader GraphQL response-meta contract
- source surfacing for every future GraphQL event query variant

## Scope

In scope:

- GraphQL event-family source-plane assessment
- explicit stop-line for the current GraphQL source contract
- architecture/index documentation updates

Out of scope:

- new GraphQL event read paths
- broader cross-protocol parity claims
- GraphQL control-plane or meta-envelope redesign

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase300-graphql-event-query-source-plane-assessment.md`

## Exit Criteria

- The docs explicitly describe the current GraphQL event-family source work as
  a stable baseline with a stop-line.
- The stop-line makes future GraphQL expansion an explicit reopen rather than
  default continuation.
