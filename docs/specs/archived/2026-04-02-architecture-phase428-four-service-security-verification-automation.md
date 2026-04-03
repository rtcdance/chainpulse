Title: Phase 428 Four-Service Security Verification Automation
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: scripts/verify-local-runnable-app.sh, SECURITY_BASELINE.md, SECURITY_ROLLOUT.md, RUNNABLE_APP.md, README.md

## Status

Status: Implemented

## Problem Statement

The four-service security posture is now documented and the rollout sequence is
clear, but the current verification entry still only checks service availability
and a few runtime surfaces. To keep the security baseline honest and repeatable,
the verification path should also assert that the security surfaces remain
disabled by default unless explicitly enabled.

## Scope

- extend the local runnable-app verification script to assert default-off
  security posture for the four services
- keep the current runnable baseline unchanged
- keep the verification path light and shell-based

## Non-Goals

- no new security behavior
- no mandatory auth rollout
- no deployment platform redesign
- no new runtime control actions

## Selected Approach

Update the existing verification script so it asserts:

- the runtime summaries expose the security posture section
- the default posture is `*-security-unconfigured` for each service
- the default security flags remain false

Keep the checks simple, textual, and aligned to the existing shell-based
verification flow.

## Risks

- overfitting the verification to exact JSON formatting if the checks are too
  brittle
- making the runnable verification too strict if the optional security surface
  changes in the future

## Rollback

- remove the security posture assertions from the verification script
- keep the documentation references intact if the runtime behavior remains
  unchanged

## Test Strategy

- run the local runnable-app verification script in the current default-off
  configuration
- run spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase428-four-service-security-verification-automation.md`

## Decision

- Add default-off security posture assertions to the existing verification flow
  so the four-service baseline is checked as a runnable app and as an opt-in
  security posture baseline.
