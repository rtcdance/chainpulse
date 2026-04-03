Title: Phase 417 - Minimal Runnable App Assessment
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor, docs

## Problem Statement

The foreground work has recently converged on the smallest realistic
microservice app slice around:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

Before opening another implementation phase, the repository needs an explicit
assessment of whether that slice has already reached a credible runnable-app
baseline for the current `ARCHITECTURE_v1.md` push, and what the next highest
value gap really is.

## Scope

- assess the current minimal runnable-app baseline
- record what is now complete
- record what is intentionally still incomplete
- define the next highest-value reopen target

## Non-Goals

- no new runtime behavior
- no new protocol implementation
- no deployment-manifest redesign

## Selected Approach

Document the current four-service state as a minimal runnable-app baseline that
is strong enough to pause by default. Explicitly record that the remaining
highest-value gap is no longer gateway query bridging, but orchestration around
bringing the four-service slice up together in a repeatable local/dev workflow.

## Risks

- overstating completion if the document implies full `ARCHITECTURE_v1.md`
  parity

## Rollback

- remove the assessment doc entry
- treat the line as still actively open without a recorded stop line

## Test Strategy

- spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase417-minimal-runnable-app-assessment.md`

## Decision

- Record the current state as a minimal runnable-app baseline with a clear next
  reopen target.

## Status

Status: Approved
