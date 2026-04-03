Title: Phase 427 Four-Service Security Rollout Readiness
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: SECURITY_BASELINE.md, RUNNABLE_APP.md, README.md, docs/ARCHITECTURE.md, docs/INDEX.md, docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md

## Status

Status: Implemented

## Problem Statement

The repository now documents an opt-in security posture across the four-service
runnable baseline. The controls exist, but there is still no single repo-root
rollout and rollback guide that tells operators the safest sequence for enabling
the security surfaces without accidentally disrupting the runnable baseline.

## Scope

- add a repo-root rollout/rollback guide for the current security posture
- recommend an explicit enablement order for the four services
- keep the current runnable baseline unchanged
- document the expected default-off rollback path

## Non-Goals

- no new security behavior
- no mandatory auth rollout
- no deployment platform redesign
- no cross-service control-plane redesign

## Selected Approach

Add a repo-root document that explains:

- the default-off expectation
- the recommended rollout order:
  1. `api-gateway`
  2. `api-service`
  3. `puller`
  4. `event-processor`
- the rollback sequence in reverse order
- the verification steps after each enablement step

## Risks

- operators could misread the guide as a hardening mandate rather than an
  optional posture baseline
- the guide could become stale if it repeats implementation details rather than
  documenting the high-level sequence

## Rollback

- remove the rollout guide
- remove the README and docs index references

## Test Strategy

- spec approval check
- no code path changes expected

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase427-four-service-security-rollout-readiness.md`

## Decision

- Record the safest opt-in rollout/rollback sequence for the four-service
  security posture so the baseline can be enabled incrementally without
  changing the default runnable path.
