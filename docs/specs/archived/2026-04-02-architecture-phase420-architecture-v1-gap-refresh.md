Title: Phase 420 Architecture v1 Gap Refresh
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: docs/archive/ARCHITECTURE_v1.md, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor, scripts

## Status

Status: Approved

## Problem Statement

The repository now has a credible minimal runnable-app baseline, but the user
goal is still anchored to `docs/archive/ARCHITECTURE_v1.md`. We need an honest
refresh that compares the current state against the archived architecture
blueprint and identifies the smallest remaining gaps to reach a lowest
acceptable blueprint-aligned completion state.

## Scope

- compare the current runnable baseline against the most relevant `ARCHITECTURE_v1.md` goals
- record what already satisfies the blueprint at a minimum viable level
- identify the one to two highest-value remaining gaps
- define what should count as a lowest acceptable blueprint-aligned completion state

## Non-Goals

- no new runtime implementation
- no new deployment platform work
- no attempt to satisfy the full long-term `ARCHITECTURE_v1.md` vision in this phase

## Selected Approach

Record the current state as already meeting the minimum viable baseline for:

- execution services
- query entrypoint
- local/dev runnable workflow

Then explicitly identify the smallest remaining blueprint-aligned gaps as:

1. one repository-root dev runbook that ties the local startup and verification
   path together
2. one clear statement that the current app is a minimum viable subset of the
   larger blueprint, not full parity

## Risks

- understating progress if the comparison is too strict
- overstating completion if the comparison implies full `ARCHITECTURE_v1.md` parity

## Rollback

- remove the assessment entry
- continue treating blueprint progress as implicit rather than explicitly bounded

## Test Strategy

- spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase420-architecture-v1-gap-refresh.md`

## Decision

- Treat the current repository as very close to the lowest acceptable
  blueprint-aligned runnable app state.
- Prioritize one repository-root runbook/entrypoint summary as the next
  highest-value gap instead of opening broader new capability fronts.
