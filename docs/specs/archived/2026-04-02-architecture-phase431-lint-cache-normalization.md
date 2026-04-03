Title: Phase 431 Lint Cache Normalization
Type: architecture
Status: Implemented
Owner: Codex
Reviewers: Platform Team
Related Modules: Makefile, .github/workflows/ci.yml, scripts/dev-micro-loop.sh, docs/architecture/MICROSERVICE_ROLLOUT_PRODUCER_COVERAGE.md

## Status

Status: Approved

## Context

The lint scope has already been narrowed to the repository's real source
directories, but the full developer micro-loop still encounters lint failures
when golangci-lint uses the default Go build cache location. In this workspace,
that default cache path is not reliable for the full-gate execution.

The most useful next step is to keep the lint gate source-root focused while
also normalizing the cache location so the same lint command behaves
consistently in local development and CI.

## Problem Statement

The full lint gate is still sensitive to the default Go build cache location.
That creates a brittle developer experience even after the lint target scope
was corrected. The lint gate should be able to analyze the repository's source
files without depending on the default host cache path.

## Scope

- keep the lint scope focused on real source files
- normalize the Go build cache location for lint execution
- keep the change minimal and aligned with the existing local/full gate flow

## Non-Goals

- no linter upgrade
- no lint rule changes
- no workflow redesign
- no new runtime behavior

## Options Considered

1. Leave cache behavior unchanged and accept the brittle full-gate failure.
   - Rejected because it leaves the developer gate unstable.
2. Disable golangci-lint caching entirely.
   - Rejected because the gate should stay fast enough for repeated use.
3. Pin golangci-lint to a workspace-safe cache location while keeping the
   source-file lint scope.
   - Selected because it preserves speed and avoids the host-cache failure.

## Selected Approach

Update the lint entrypoints to:

- keep analyzing only real source files
- run golangci-lint with a workspace-safe `GOCACHE` value

This should keep both local and CI lint execution reproducible without
changing the lint rules themselves.

## Data / Contract Impact

No runtime contract changes are expected. This only affects lint execution
environment and static analysis scope.

## Risks

- The fixed cache location may need to be kept writable in other environments.
- The file-level lint invocation could be slower than a package-root invocation.

## Rollback Plan

- restore the previous lint invocation if the cache-normalized path proves too
  slow or brittle
- keep the source-root scope correction in place if it still improves signal

## Test and Verification Plan

- run `scripts/dev-micro-loop.sh --mode fast --base HEAD`
- run `scripts/dev-micro-loop.sh --mode full --base HEAD`
- run the spec approval check

## Quality Gates

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase431-lint-cache-normalization.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD`

## Review Notes

- Approved as the smallest next step to make the full lint gate stable after
  the source-root scope correction.

## Implementation Summary

The lint entrypoints now pin `GOCACHE` to a workspace-safe path when they run
golangci-lint so full-gate execution no longer depends on the host's default
Go build cache location.

The source-root lint scope remains narrowed to the actual production roots, so
the cache normalization improves reproducibility without changing lint rules or
runtime behavior.

## Final Verification

- `./scripts/spec-approval-check.sh docs/specs/2026-04-02-architecture-phase431-lint-cache-normalization.md`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode fast --base HEAD`
- `unset GOROOT; export PATH="/Users/mingo/Applications/workspace/goproject/bin:/opt/homebrew/opt/go@1.24/bin:$PATH"; export GOPROXY="https://proxy.golang.org,direct"; scripts/dev-micro-loop.sh --mode full --base HEAD`
