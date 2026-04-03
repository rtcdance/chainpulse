# Phase 290 - GraphQL Query Source Pilot Assessment

## Status
Status: Approved

## Summary

Record an explicit stage assessment for the GraphQL query-source-surfacing
pilot after extending it across both the single-event `event` path and the
list-style `eventsByName` path.

## Problem

After phases 287 and 289, GraphQL query-source surfacing now covers:

- `event`
- `eventsByName`

This is enough to show a credible pilot, but still not enough to claim broad
GraphQL query parity or full cross-protocol source semantics. Without an
explicit assessment, it would be too easy to either:

- overstate the current GraphQL coverage
- or keep expanding the pilot without a clear stop point

## Decision

Classify the current state as:

- **GraphQL query-source surfacing pilot established**

That means:

- the pilot is no longer a one-resolver experiment
- GraphQL now has a small but real source-surfacing slice
- further GraphQL expansion should be treated as an explicit new objective
  rather than automatic continuation of the pilot

## Scope

In scope:

- GraphQL pilot stage assessment
- stop-line guidance for current source-surfacing coverage

Out of scope:

- new GraphQL query meta envelopes
- broad GraphQL parity claims
- additional resolver expansion

## Validation

- `./scripts/spec-approval-check.sh docs/specs/2026-04-01-architecture-phase290-graphql-query-source-pilot-assessment.md`

## Exit Criteria

- The repository contains an explicit stage judgment for the current GraphQL
  source-surfacing pilot.
- The judgment makes clear that current GraphQL coverage is a pilot boundary,
  not full GraphQL parity.
