# 10x / 50-chain scaling: a design sketch

> **This is a design document. Nothing in it is implemented.**
> F26 asked for a "10x/50 链扩展设计（**只设计不实现**）" as interview-preparation
> material, and T17's acceptance requires that it cross-reference the project's
> ADRs and **not imply capabilities the repository does not have**.
>
> Status: **proposal only.** Every section says what would have to change and what
> would have to be measured; none of it claims to exist today.

Related records: [ADR-001 plugin architecture](../adr/ADR-001-plugin-architecture.md),
[ADR-002 event bus](../adr/ADR-002-event-bus-pubsub.md),
[ADR-003 reorg rollback/reindex](../adr/ADR-003-reorg-rollback-reindex.md).

---

## 1. What exists today, stated plainly

So the gap is measurable rather than hand-waved:

| Concern | Current state | Where |
| --- | --- | --- |
| Chain isolation | Chain-scoped accessors exist; unscoped port methods are refused once an adapter declares more than one chain | `pkg/plugins/database/postgres_chain_scope.go` |
| Message ordering | Partition key is the chain key, frozen in code and test | `pkg/plugins/mq/partition_contract.go` |
| One writer per chain | A lease with an owner token and a monotonic generation; commits are fenced | `pkg/plugins/database/postgres_lease.go` |
| Atomic progress | Events and the checkpoint advance commit in one transaction | `pkg/services/processor/batch_committer.go` |
| Reorg recovery | A persistent repair job plus a transactional outbox | `pkg/plugins/database/postgres_reorg_outbox.go` |
| Cache correctness | Keys are versioned by chain, generation and schema | `pkg/plugins/cache/cache_scope.go` |
| Provider limits | Narrow providers are bisected instead of skipped | `pkg/plugins/pullers/range_splitter.go` |

What this is **not**: a deployment that has run at 50 chains. The largest thing
actually exercised in this repository is a **two-chain** isolation test and a
single-chain load path. Any number below is a target, not a measurement.

## 2. Where the current design stops scaling

The design is chain-scoped throughout, which is the right axis, but three
assumptions break as chain count grows.

**2.1 One lease row per chain is fine; one *worker pool* per process is not.**
`AcquireLease` gives exactly one writer per chain, which is what correctness
needs. But nothing today decides *which* process takes which chain — the
partition key makes Kafka deliver a chain to one consumer in a group, and the
lease backstops it. At 50 chains, a rebalance moves up to 50 partitions at once,
and the recovery path (`Repair.Resume`) walks pending jobs **sequentially**.

**2.2 The query path is per-chain-scoped but not per-chain-partitioned.**
`blockchain_events` is a single table. A chain-scoped index makes one chain's
queries fast; it does not stop 50 chains from sharing one table's bloat, vacuum
and index maintenance.

**2.3 Provider cost scales with chains, not with events.** Each chain needs its
own `eth_getLogs` budget. 50 chains × one provider is a rate-limit problem before
it is a throughput problem, and `RangeSplitter` already shows how a provider
degrades — more chains means more of them are the narrow kind.

## 3. The design

### 3.1 Assign chains to workers explicitly (extends ADR-001)

ADR-001 chose two deployment modes from one codebase. The 10x version keeps that
and adds a **chain→worker assignment** layer:

- a chain registry (which chains exist, their finality model, their provider set);
- a deterministic assignment (hash of chain key over the worker set) so a
  rebalance moves the minimum number of chains;
- the existing lease as the *authority* — the assignment is an optimisation, and a
  worker that lost its lease still cannot commit.

This is a scheduling layer on top of ADR-001, not a new architecture.

### 3.2 Partition storage by chain (extends ADR-003)

Reorg handling (ADR-003) already treats a chain's history as rewritable within a
depth. That makes chain a natural partition key:

- partition `blockchain_events` and `blocks` by chain key (or shard by a hash of
  it), so one chain's reorg rewrites only its own partition;
- keep `indexing_state` and `chain_leases` unpartitioned — they are tiny and must
  stay transactionally cheap;
- the repair job's range scan then touches one partition instead of the table.

The open question is whether to partition by `LIST (chain_id)` (predictable, but
requires a migration per new chain) or by `HASH` (no migration, but a chain's rows
spread across partitions, which hurts the range scans ADR-003 depends on). **That
trade-off needs measurement before it needs an opinion.**

### 3.3 Make reorg repair concurrent, not sequential (extends ADR-003)

`Repair.Resume` today repairs pending jobs one at a time. At 50 chains a single
deep reorg must not block the other 49:

- repair jobs are already per-chain and carry a generation, so they can run in
  parallel **as long as** a chain's own jobs stay ordered;
- the bound is per-chain, not global: one worker per chain, N chains in parallel.

This preserves ADR-003's contract (rollback → rewind → invalidate → complete) and
only changes how many chains run it at once.

### 3.4 Fan out the event bus by chain (extends ADR-002)

ADR-002's pub/sub is in-process. Across 50 chains and multiple processes the same
message shape works, but two things change:

- subscribers must declare **which chains** they care about, so a subscriber is
  not woken for 49 chains it ignores;
- the outbox drainer becomes per-chain-shard rather than one global loop.

### 3.5 Budget providers per chain, not globally

`RangeSplitter` already learns the widest range a provider accepts. The 10x
version records that **per chain per provider** and uses it to choose the initial
chunk size instead of always starting optimistic. The measurement that decides
this: how many chains a single provider can serve before 429s dominate.

## 4. What would have to be measured before believing any of it

None of the above is worth building without numbers. In order:

1. **Events per second per chain** — the 10x claim is about chains, not volume; if
   each of 50 chains carries 1/50th the volume, the bottleneck is elsewhere.
2. **Reorg frequency and depth per chain** — decides whether §3.3 matters at all.
3. **Provider 429 rate vs chain count** — decides §3.5 and whether the design is
   provider-bound rather than code-bound.
4. **Query latency at 50 chains on one table** — decides §3.2. This is the one
   most likely to change the design.

## 5. What this document deliberately does not claim

- No multi-chain deployment has been run here. The two-chain test proves
  *isolation*, not scale.
- No partitioning, chain registry, or per-chain repair concurrency exists in the
  repository.
- The numbers in §2 are the current *shape*, not measurements of the current
  *limits*.

The honest summary for an interview: the repository is built along the right axis
(chain-scoped everywhere, one writer per chain, atomic progress) and the scaling
work is about **scheduling, storage layout and provider budget** — not about
redesigning the correctness model.
