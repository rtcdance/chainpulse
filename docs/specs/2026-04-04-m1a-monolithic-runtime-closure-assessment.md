Title: M1a Monolithic Runtime Closure Assessment
Type: architecture
Status: Approved
Owner: Codex
Reviewers: Mingo
Related Modules: cmd/monolithic/chainpulse, pkg/adapters/indexing, pkg/plugins/api

## Status

Implemented. Header remains `Status: Approved` for repository checker compatibility.

## Problem Statement

`M1a` was defined as the milestone that closes the monolithic foundational data path described by `ARCHITECTURE_v1.md`: pullers must exist per chain, events must move through `EventBus` into indexers, the main query surface must read from the same indexing data plane, and the monolithic runtime must expose a truthful minimal operational surface. After five focused M1a slices, the repository now needs a milestone-level assessment to determine whether this baseline is actually complete enough to stop adding more M1a functionality and move into `M1b`.

## Assessment

The monolithic runtime now has all core closure points required by the M1a blueprint baseline:

1. real per-chain HTTPS pullers owned by monolithic bootstrap
2. a real `EventBus -> MultiChainIndexer` execution path
3. indexing-backed `/events` read paths over the same monolithic data plane
4. string chain-route alignment for `/events/chain/{chainId}`
5. a minimal per-chain in-memory reorg rollback seam
6. compact puller runtime posture plus a read-only `/runtime/control` surface

These changes move monolithic mode from passive wiring into a runnable, inspectable indexing/query loop that matches the minimum acceptable M1a boundary.

## Remaining Gaps

The following items still exist, but they no longer justify keeping the work inside M1a:

1. deeper fault-tolerance and resilience tuning for pull loops
2. richer operational semantics beyond read-only monolithic control status
3. broader observability and gateway hardening

Those belong to `M1b` and `M1c`, not to the foundational runtime-closure milestone.

## Decision

`M1a` is now **stage-complete for the monolithic foundational runtime baseline**.

The repository should stop adding new M1a execution slices and switch the active implementation focus to `M1b`.

## Verification Summary

This assessment is based on the focused M1a verification chain already completed across the five implementation slices, including:

1. `go test -short ./cmd/monolithic/chainpulse/...`
2. `go test -short ./pkg/adapters/indexing/...`
3. `go test -short ./pkg/plugins/api/...`
4. `./scripts/spec-approval-check.sh docs/specs/2026-04-03-m1a-monolithic-eventbus-puller-indexer-wiring.md`
5. `./scripts/spec-approval-check.sh docs/specs/2026-04-03-m1a-monolithic-indexing-backed-query-wiring.md`
6. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-chain-route-contract-alignment.md`
7. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-reorg-handler-wiring.md`
8. `./scripts/spec-approval-check.sh docs/specs/2026-04-04-m1a-monolithic-puller-runtime-surface.md`
