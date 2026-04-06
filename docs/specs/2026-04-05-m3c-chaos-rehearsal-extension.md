Title: M3c Chaos Rehearsal Extension
Type: feature
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: scripts, README.md, RUNNABLE_APP.md

## Status

Approved for implementation.

## Problem Statement

The repository now has a dedicated `chaos-test.sh`, but the top-level
production-readiness rehearsal still only sequences:

1. deployment smoke
2. observability baseline
3. alert-readiness baseline

That leaves chaos validation as a sidecar command instead of part of the main
rehearsal flow.

## Scope

This slice will:

1. extend the production-readiness rehearsal to run `scripts/chaos-test.sh`
2. update README and runnable docs to mention the new rehearsal scope

## Non-Goals

This slice will not:

1. change chaos-test assertions
2. add new live dependency checks beyond chaos-test itself
3. claim final production readiness

## Selected Approach

Keep the change thin and orchestration-only:

1. append chaos-test to the existing rehearsal sequence
2. update user-facing docs to reflect the widened rehearsal coverage

## Risks

1. the full rehearsal will now depend on Docker availability whenever it reaches
   the chaos step
2. runtime duration increases because chaos-test is live and restart-oriented

## Rollback Plan

1. remove chaos-test from the rehearsal script
2. restore prior docs wording

## Test Strategy

1. `bash -n scripts/run-production-readiness-rehearsal.sh`
2. `bash scripts/run-production-readiness-rehearsal.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3c-chaos-rehearsal-extension.md`

## Quality Gates

1. `bash -n scripts/run-production-readiness-rehearsal.sh`
2. `bash scripts/run-production-readiness-rehearsal.sh --help`
3. `./scripts/spec-approval-check.sh docs/specs/2026-04-05-m3c-chaos-rehearsal-extension.md`

## Review Notes

Approved as the smallest way to promote chaos validation into the main
production-readiness rehearsal path.
