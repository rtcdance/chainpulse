# Phase 296 - GraphQL Event Query Source Baseline Refresh

## Status
Status: Approved

## Summary

Refresh the GraphQL query-source-surfacing assessment after the mini-baseline
expanded to include the root `events` connection in addition to the existing
single-event and filtered event read paths.

## Problem

After phase 295, GraphQL query-source surfacing now spans:

- `event`
- `events`
- `eventsByName`
- `eventsByAddress`
- `eventsByBlock`

The current documentation still described the GraphQL state as a strong
mini-baseline without explicitly capturing that the generic root list path has
joined the same source-surfacing surface.

## Decision

Refresh the GraphQL assessment and stop-line to treat the current state as a
broader event-query source baseline:

- single-event reads are covered
- root paginated event lists are covered
- key filtered event list reads are covered

Keep the boundary explicit:

- this is still not full GraphQL parity
- no broader GraphQL meta envelope is implied
- future expansion should be an explicit reopen, not default drift

## Scope

In scope:

- GraphQL event-query source baseline assessment refresh
- updated stop-line for the current GraphQL event family coverage
- architecture/index documentation updates

Out of scope:

- additional GraphQL resolver expansion
- full cross-protocol parity claims
- new GraphQL response metadata contracts

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase296-graphql-event-query-source-baseline-refresh.md`

## Exit Criteria

- The docs explicitly describe the current GraphQL event-query source coverage
  as a refreshed baseline rather than a narrow pilot slice.
- The stop-line clearly states that further expansion requires explicit reopen.
