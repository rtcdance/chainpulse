# ADR-010: Finality and Reorg Degradation

**Date**: 2026-10-13

### Status

Accepted

### Context

"Finalized" is not one thing across chains. Ethereum PoS finalizes through
epochs (32 slots ≈ 6.4 min); an L2 rollup may expose only a `safe` tag that
reflects the sequencer's view, with true finality gated on an L1 challenge
window (Optimism, ~7 days) or an L1 proof (ZK rollups); some endpoints support
neither tag and silently fall back to `latest`.

The indexer must answer "is this block final?" and must be able to say **how
sure it is**, because the answer gates whether derived state may be promoted and
whether a reorg can still rewrite the block. A single boolean cannot express
this, and a boolean plus a `degraded` flag was not enough either (finding F08):

- **F08 (false finality)** — an L2 `safe` answer reported `Degraded=false` while
  carrying **no** finality guarantee, so a caller treating `IsFinalized` as
  authoritative promoted events that were still reorg-able.

### Options considered

1. **A single `confirmed` boolean.** Cannot express "finalized" vs "safe" vs
   "we fell back to latest". Every caller would have to know the chain's
   semantics out of band. Rejected — it is the bug F08 describes.
2. **A block-depth heuristic only** (e.g. "12 blocks is final"). Ignores
   chain-specific finality and is wrong on both PoS (too shallow) and L2 (far too
   shallow). Rejected as the primary rule; depth remains a *fallback signal*, not
   the verdict.
3. **Trust the RPC `finalized` tag blindly.** L2 nodes frequently do not support
   it; a blind caller then treats a `latest` fallback as final. Rejected.
4. **An explicit confidence + source enum, fail-closed.** Only one value means
   "may be promoted", and every degraded path is visible and named. Chosen.

### Decision

**Finality is a confidence, not a boolean.**

- `finality.FinalityConfidence` — `unknown | degraded | safe | finalized`
  (`pkg/services/finality/finality_checker.go`). **Only `finalized` may be
  treated as final.**
- `finality.FinalitySource` — `finalized_tag | safe_tag | latest_fallback |
  unknown`, so a caller can distinguish a `safe` answer from a `latest` fallback
  for diagnostics and alerting.
- `IsBlockFinalized` returns `true` **only** when the height came from the
  `finalized` tag and the block is at or below it. A degraded or `safe` answer
  returns `false` even when the height suggests otherwise.
- **L2 finality discount** — for an L2 chain, the `safe` height is reduced by
  the rollup's L1 window (`chainid.GetL2ChainInfo(...).FinalityBlocks`) before it
  is used, so an L2 block is not promoted until its L1 backing is accounted for.
- **Degraded values expire faster** — a `latest` fallback is cached for a
  shorter TTL than a real tag (2s vs 12s), so a node that regains finality
  support stops reporting degraded quickly.

**Reorg handling is an ordered repair, not a delete.**

A reorg enqueues a persistent repair job
(`pkg/services/reorg/reorg_job.go`) whose runner (`reorg_repair.go`) executes
three steps **in this order**:

1. **Roll back events** for the range — so the data is consistent first;
2. **Rewind the checkpoint** — so a crash leaves a resume point at or below the
   repaired range (re-running is safe, skipping is not);
3. **Invalidate derived state** for the range, **scoped by generation** —
   anything cached under a lower generation is stale by definition.

The order is deliberate: events before the checkpoint, checkpoint before derived
state, because derived state is rebuildable from the other two and the other two
are not rebuildable from it.

### Failure semantics

- **RPC cannot answer finality** — the result is `degraded`, never `finalized`.
  Callers fail closed; queries may still serve the data with a lower confidence,
  but nothing is promoted as final.
- **L2 `safe`** — carries a finality guarantee of *none*; it is `safe`, not
  `finalized`, and is discounted by the L1 window.
- **Reorg deeper than the configured rollback bound** — the repair refuses and
  alerts rather than deleting an unbounded range.
- **Crash mid-repair** — the job stays pending and resumes; the resume point is
  at or below the repaired range, so re-running is idempotent.
- **Repair invalidation without a generation** — rejected: an invalidation must
  say which generation it is repairing, or it cannot distinguish stale from
  current state.

### Tests

- `pkg/services/finality/finality_checker_test.go` — `L1Finalized`, `L2Safe`,
  `FallbackToLatest`, `AllFallbacksFail`, `IsBlockFinalizedWithStatus_Degraded`.
- `pkg/services/finality/finality_promotion_test.go` — `RefusesDegraded`,
  `SafeIsNotFinalized`, `FinalizedTagPromotesOnlyBelowHeight`.
- `pkg/services/reorg/reorg_repair_test.go` — `RunsStepsInOrder`,
  `PassesGenerationToInvalidation`, `LeavesJobPendingOnFailure`,
  `RejectsInvertedRange`, `ResumeRepairsEveryPendingJob`.
- E09 rehearsals — degraded-source and reorg recovery paths.

### Consequences

- **Positive** — a caller can never mistake a `safe` or `latest` answer for
  finality (F08 closed). Degradation is named, observable, and expires quickly.
  Reorg repair is crash-safe and generation-scoped, so stale derived state cannot
  be served after a repair.
- **Negative** — there are more states to reason about, and every query must
  choose the confidence level it needs (a product decision, not a default).
  Chains whose endpoints never support a finality tag remain permanently
  `degraded`, which is honest but means their data is never promoted.
- **Neutral** — the `safe` tag is still used for L2 chains, but as an explicitly
  discounted, explicitly non-final input rather than as an approximation of
  finality.
