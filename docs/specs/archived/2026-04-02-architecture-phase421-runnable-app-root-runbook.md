Title: Phase 421 Runnable App Root Runbook
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: README.md, RUNNABLE_APP.md, scripts, docs/archive/ARCHITECTURE_v1.md

## Status

Status: Approved

## Problem Statement

The repository now has a minimum viable blueprint-aligned runnable app slice,
but the startup and verification guidance is still split across service-local
quickstarts and scripts. The highest-value remaining gap is one repository-root
runbook that explains the current runnable app boundary in one place.

## Scope

- Add one repository-root runbook for the current runnable app.
- Describe:
  - what the runnable app currently includes
  - required local dependencies
  - shared startup entry
  - shared verification entry
  - current service boundaries
- Update the root README so the runnable path is discoverable immediately.

## Non-Goals

- No new runtime behavior
- No new service implementation
- No deployment-platform redesign
- No claim of full `ARCHITECTURE_v1.md` parity

## Selected Approach

Create `RUNNABLE_APP.md` at the repository root and position it as the current
best entry for the minimum viable blueprint-aligned app. Then update `README.md`
to point its quick-start flow at that runbook instead of the older generic
docker-first path.

## Risks

- Overstating completion if the runbook implies full blueprint completion
- Confusion if the root README and service quickstarts disagree

## Rollback

- Remove `RUNNABLE_APP.md`
- Restore the older root README quick-start wording

## Test Strategy

- spec approval check
- manual doc consistency review

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase421-runnable-app-root-runbook.md`

## Decision

- Treat a repository-root runnable-app runbook as the final highest-value doc
  gap for the current minimum viable blueprint-aligned app state.
