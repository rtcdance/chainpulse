Title: Phase 429 Four-Service Security CI Check
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Platform Team
Related Modules: .github/workflows/ci.yml, cmd/microservices/api-gateway, cmd/microservices/api-service, cmd/microservices/puller, cmd/microservices/event-processor, scripts/verify-local-runnable-app.sh, SECURITY_BASELINE.md, SECURITY_ROLLOUT.md

## Status

Status: Approved

## Context

The repository now has a documented four-service security baseline and a
shared runnable-app verification flow that asserts the default-off security
posture for the gateway, query service, puller, and event processor.

That verification currently lives in a local shell script and is exercised
manually or through local developer workflows. The next useful step is to make
the security posture checks part of CI without expanding the runtime
environment or introducing a full service orchestration dependency into the
workflow.

## Problem Statement

The four-service security baseline is opt-in and default-off, but the CI
pipeline does not yet execute any automated check that covers the runnable-app
security posture surfaces. This leaves a gap between the documented baseline
and the verification path used by contributors.

## Scope

- add a focused CI check for the four service command packages that exercise
  the runnable-app and security posture tests
- keep the existing build and test pipeline structure intact
- avoid introducing external service orchestration or additional deployment
  dependencies in CI

## Non-Goals

- no new security features
- no change to the default-off posture
- no mandatory auth rollout
- no full end-to-end local orchestration in CI
- no workflow redesign beyond the minimal verification addition

## Options Considered

1. Run the full local runnable-app shell script in CI.
   - Rejected for now because it would require external dependencies and a
     heavier orchestration surface than the current CI should own.
2. Add a focused `go test` check for the four command packages.
   - Selected because it exercises the security posture assertions already
     encoded in the microservice tests without introducing new runtime
     dependencies.
3. Create a separate workflow for runnable-app verification.
   - Rejected for now because it would spread the verification logic across
     multiple CI entry points.

## Selected Approach

Add a minimal CI step to the existing unit test job that runs the four command
package test suites:

- `./cmd/microservices/api-gateway/...`
- `./cmd/microservices/api-service/...`
- `./cmd/microservices/puller/...`
- `./cmd/microservices/event-processor/...`

This keeps the gate lightweight, uses the existing tests that already assert
the security posture surfaces, and makes CI fail if the default-off security
baseline regresses.

## Data / Contract Impact

No runtime contract changes are expected. The change only adds a CI execution
path that validates the existing runtime summary and security posture behavior.

## Risks

- The additional command-package tests could extend CI time modestly.
- If command-package tests start relying on external dependencies later, this
  CI step may need to stay focused on the runnable-app smoke surface rather
  than broad integration behavior.

## Rollback Plan

- remove the command-package CI step from `.github/workflows/ci.yml`
- keep the existing runnable-app shell verification and unit tests unchanged

## Test and Verification Plan

- run the updated CI workflow locally by executing the new command-package test
  step
- run the relevant `go test` command directly for the four command packages
- run the spec approval check

## Quality Gates

- `go test -short ./cmd/microservices/api-gateway/... ./cmd/microservices/api-service/... ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase429-four-service-security-ci-check.md`

## Review Notes

- Approved by the architecture owner as a minimal CI hardening step for the
  existing four-service security baseline.

## Implementation Summary

Added a CI step to the existing unit-test workflow so GitHub Actions now runs
the four runnable-app command package test suites for:

- `api-gateway`
- `api-service`
- `puller`
- `event-processor`

This keeps the four-service security posture baseline covered by automated CI
without introducing external orchestration or changing the default-off
security behavior.

## Final Verification

- `unset GOROOT; export GOCACHE=/tmp/chainpulse-go-build-cache; /opt/homebrew/opt/go@1.24/bin/go test -short ./cmd/microservices/api-gateway/... ./cmd/microservices/api-service/... ./cmd/microservices/puller/... ./cmd/microservices/event-processor/...`
- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase429-four-service-security-ci-check.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD` passed
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD` failed in the lint step with `context loading failed: no go files to analyze`
