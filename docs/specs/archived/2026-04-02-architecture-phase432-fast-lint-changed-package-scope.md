Title: Phase 432 Fast Lint Changed-Package Scope
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: scripts/dev-micro-loop.sh, Makefile, .github/workflows/ci.yml, docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md

## Status

Status: Approved

## Context

The full lint gate has been tightened to real source files and a workspace-safe
cache, but the fast developer micro-loop still uses `golangci-lint run --new
./...`, which can pull in unrelated packages and fail on existing test-only
noise outside the changed surface.

The fast gate should remain focused on the changed packages so it stays a quick
signal for the current diff rather than a broad repository scan.

## Problem Statement

The fast lint path is broader than the set of changed packages it is supposed
to guard. This makes the fast gate vulnerable to unrelated lint/typecheck
failures in untouched packages, which reduces its value as a tight micro-loop
signal.

## Scope

- narrow the fast lint invocation to the changed package set computed by the
  micro-loop
- keep the full lint gate source-root scoped and cache-normalized
- avoid changing lint rules or runtime behavior

## Non-Goals

- no linter upgrade
- no lint rule changes
- no repository-wide lint redesign
- no change to test execution order

## Options Considered

1. Leave the fast lint path broad and accept unrelated failures.
   - Rejected because it weakens the fast feedback loop.
2. Skip fast lint entirely.
   - Rejected because the fast gate should still catch issues in the changed
     packages.
3. Run fast lint only on the changed packages computed by the micro-loop.
   - Selected because it keeps the gate focused and stable.

## Selected Approach

Update `scripts/dev-micro-loop.sh` so the fast lint step runs
`golangci-lint run --new` against the changed package list instead of `./...`.
This keeps the fast gate aligned to the current diff while leaving the full gate
responsible for broader source-root analysis.

## Data / Contract Impact

No runtime contract changes are expected. This only affects local quality-gate
scope.

## Risks

- The changed-package list could miss a dependency edge if the diff touches a
  package boundary indirectly.
- The fast gate will still rely on the full gate for broader repository
  coverage.

## Rollback Plan

- restore the broader fast lint scope if changed-package scoping proves too
  narrow
- keep the full gate source-root scoped and cache-normalized

## Test and Verification Plan

- run `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- run `scripts/dev-micro-loop.sh --mode full --base HEAD`
- run the spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase432-fast-lint-changed-package-scope.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD`

## Review Notes

- Approved as the minimal way to keep fast lint focused on changed packages
  after the full-gate lint was normalized.

## Implementation Summary

The fast developer micro-loop now computes the changed package set for the
current diff and runs golangci-lint only against those changed packages.

This keeps the fast gate focused on the current change surface while the full
gate remains responsible for broader source-root analysis and cache-normalized
lint execution.

## Final Verification

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase432-fast-lint-changed-package-scope.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD`
