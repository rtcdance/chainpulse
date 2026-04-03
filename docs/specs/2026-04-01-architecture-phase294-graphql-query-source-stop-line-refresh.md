# Phase 294 - GraphQL Query Source Stop-Line Refresh

## Status
Status: Approved

## Summary

Refresh the GraphQL query-source-surfacing assessment after the current
mini-baseline expanded across:

- `event`
- `eventsByName`
- `eventsByAddress`
- `eventsByBlock`

## Problem

After phase 293, the GraphQL source-surfacing line was stronger than the
mini-baseline judgment recorded in phase 292. The repository needed an updated
stop-line so the current state is described accurately:

- stronger than an early mini-baseline
- still short of full GraphQL parity

Without that refresh, we would either understate the current GraphQL coverage
or keep extending it without a clear pause boundary.

## Decision

Refresh the GraphQL assessment and classify the current state as:

- **strong GraphQL source-surfacing mini-baseline**

That means:

- GraphQL now has a coherent set of source-surfacing reads across single-event
  and multiple list-style event queries
- the current line is strong enough to pause on
- any further GraphQL expansion should be treated as an explicit new objective
  rather than automatic continuation

## Scope

In scope:

- GraphQL query-source-surfacing assessment refresh
- updated stop-line guidance for the current GraphQL baseline

Out of scope:

- new GraphQL query meta envelopes
- full GraphQL parity claims
- additional resolver expansion

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase294-graphql-query-source-stop-line-refresh.md`

## Exit Criteria

- The repository contains an updated stop-line for the current GraphQL
  source-surfacing baseline.
- The assessment reflects the stronger current coverage without claiming full
  GraphQL parity.
