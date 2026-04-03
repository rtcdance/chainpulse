# Phase 292 - GraphQL Query Source Baseline Refresh

## Status
Status: Approved

## Summary

Refresh the GraphQL query-source-surfacing assessment after expanding the pilot
across the single-event `event` path and the two list-style event paths
`eventsByName` and `eventsByAddress`.

## Problem

After phase 291, the GraphQL query-source-surfacing pilot now covers:

- `event`
- `eventsByName`
- `eventsByAddress`

That is stronger than the original pilot boundary captured in phase 290. The
repository needed an updated judgment so the current state is not understated
as a narrow experiment nor overstated as full GraphQL parity.

## Decision

Refresh the assessment and classify the current GraphQL state as:

- **mini-baseline established for GraphQL query source surfacing**

That means:

- GraphQL now has a small but coherent source-surfacing baseline
- the baseline spans both single-item and list-style event reads
- further expansion should still be treated as an explicit new objective rather
  than automatic continuation toward full GraphQL parity

## Scope

In scope:

- GraphQL query-source-surfacing assessment refresh
- stop-line guidance for the current GraphQL mini-baseline

Out of scope:

- new GraphQL query meta envelopes
- full GraphQL parity claims
- additional resolver expansion

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase292-graphql-query-source-baseline-refresh.md`

## Exit Criteria

- The repository contains an updated assessment for the current GraphQL
  source-surfacing state.
- The assessment reflects the current mini-baseline without claiming full
  GraphQL parity.
