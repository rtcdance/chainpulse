Title: Phase 426 Four-Service Security Posture Baseline
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: README.md, RUNNABLE_APP.md, docs/ARCHITECTURE.md, docs/INDEX.md, docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md

## Status

Status: Implemented

## Problem Statement

The repository now exposes optional, default-off security surfaces across the
four-service runnable baseline:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

Those controls are implemented and verified, but the current discoverability is
still fragmented across service-local quickstarts and docs. We need a single
repo-root statement that records the current security posture of the runnable
baseline without turning it into a new hardening program.

## Scope

- summarize the current opt-in security posture of the four-service baseline
- add a repo-root doc entry so the posture is discoverable from the top level
- keep the current runnable baseline unchanged

## Non-Goals

- no new security behavior
- no mandatory auth rollout
- no new deployment policy
- no cross-service control-plane redesign

## Selected Approach

Add a repo-root security posture baseline document that states:

- each service has an optional security surface
- the default runnable path remains open
- the security surfaces are opt-in and documented
- the security surfaces should not be forced on by default

Link the new document from the root README and docs index.

## Risks

- overclaiming that the repository is hardened when the controls are only
  opt-in
- making the root docs too long or redundant

## Rollback

- remove the repo-root security posture document
- remove the README and docs index references

## Test Strategy

- spec approval check
- no code path changes expected

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase426-four-service-security-posture-baseline.md`

## Decision

- Record the current four-service optional security posture at the repo root so
  the runnable baseline and its opt-in security direction are discoverable in
  one place.
