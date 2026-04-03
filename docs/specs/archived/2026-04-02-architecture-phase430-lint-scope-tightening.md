Title: Phase 430 Lint Scope Tightening
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: Makefile, .github/workflows/ci.yml, scripts/dev-micro-loop.sh, docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md

## Status

Status: Approved

## Context

The repository already has a working runnable-app baseline, a repo-root
security posture baseline, and a shared verification flow. The developer
micro-loop full gate now fails in the lint step because the repository-wide
lint scope includes test-only directories that contain no non-test Go files
after golangci-lint excludes `_test.go` files.

That makes the lint gate broader than the actual source roots and causes
unhelpful "no go files to analyze" failures even though the runnable baseline
remains healthy.

## Problem Statement

The current lint scope is too broad for the repository layout. It includes
directories that only contain tests, so golangci-lint can end up with no
analyzable Go files in those paths once test files are excluded. This breaks
the full quality gate without improving signal on the actual source roots.

## Scope

- tighten lint targets to the repository's real source roots
- preserve the existing lint tooling and workflow structure
- keep the change minimal so the runnable-app and security baselines are not
  affected

## Non-Goals

- no new lint rules
- no linter version upgrade
- no disabling of useful lint checks
- no test behavior changes
- no workflow redesign beyond narrowing the lint scope

## Options Considered

1. Leave lint scope unchanged and accept the noisy failure.
   - Rejected because it blocks the full gate and does not improve source
     coverage.
2. Disable the test-file exclusion in golangci-lint.
   - Rejected because it would flood the lint run with test-only noise and does
     not match the desired source-quality gate.
3. Restrict lint scope to actual source roots.
   - Selected because it preserves lint signal while avoiding test-only
     directories that have no non-test Go files.

## Selected Approach

Update the repository lint entry points to run golangci-lint only against the
real Go source roots:

- `./cmd/...`
- `./pkg/...`
- `./services/...`

If needed, keep test-only trees out of the lint scope while leaving tests
themselves fully covered by `go test` in the existing gates.

## Data / Contract Impact

No runtime contract changes are expected. The change only affects static
analysis scope.

## Risks

- Some source files outside the selected roots could be missed if the repo
  later adds new production code outside those paths.
- The lint scope will need to be revisited if the source tree expands.

## Rollback Plan

- restore the broader lint target if the repository layout changes
- keep the existing test and runnable-app gates unchanged

## Test and Verification Plan

- run `scripts/dev-micro-loop.sh --mode full --base HEAD`
- run `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- run the spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase430-lint-scope-tightening.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD`

## Review Notes

- Approved as a small lint-scope correction to keep the full quality gate
  aligned with the source tree.

## Implementation Summary

The full developer micro-loop and CI lint entrypoints now scope golangci-lint to
the repository's actual source roots instead of broad parent paths that can
collapse to test-only trees:

- `./cmd/...`
- `./pkg/...`
- `./services/...`

The lint entrypoints continue to exclude `_test.go` files from lint analysis,
while tests remain covered by the existing `go test` gates.

## Final Verification

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase430-lint-scope-tightening.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD`
