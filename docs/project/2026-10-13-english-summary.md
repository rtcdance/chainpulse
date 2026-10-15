# ChainPulse — English project summary

> T17 deliverable (plan §11): an English summary positioned as **Web3 Go
> Backend** work. It does **not** claim production Web3 architect experience —
> see *Scope and honesty* at the end.

## Positioning

A Web3 Go backend: an **EVM event indexing service** with PostgreSQL
persistence, Kafka redelivery, a Redis cache and a query API. The interesting
part is not "read logs and store them"; it is doing that **correctly** when the
chain rewrites history, the queue redelivers, and "finalized" means different
things on different chains.

## The problem it solves

Indexing EVM logs is a correctness problem, not a CRUD problem:

- a **reorg** replaces a block at the same height, so an indexer keyed on block
  number collides old and new events;
- a **queue** delivers at-least-once, so the same log arrives twice and must not
  double-count;
- **finality** differs by chain — an L2 `safe` tag carries no finality
  guarantee, and a node that lacks the tag silently falls back to `latest`;
- a **crash** between "events written" and "checkpoint advanced" either loses a
  batch or replays it, and a rebalance can leave two writers on one chain.

Each of those is a decision with a trade-off, not a bug to patch — which is why
they are written down as ADRs.

## What is built (the real architecture)

The pipeline, as the code actually implements it (diagram in the
[README](../../README.md#-architecture)):

| Stage | Package | What it does |
| --- | --- | --- |
| Ingest | `pkg/plugins/pullers` | `eth_getLogs` with bounded complete-block ranges; provider-limit acceptance remains pending |
| Decode | `pkg/evm` | Typed events (ERC20/721/1155, Uniswap V3) with shape validation |
| Commit | `pkg/services/processor` | Fenced **atomic** commit of events + checkpoint |
| Store | `pkg/plugins/database` | PostgreSQL: natural-key uniqueness, lease/generation fencing, reorg outbox |
| Queue | `pkg/plugins/mq` | Kafka, partition key = chain key, at-least-once + idempotent dedup |
| Cache | `pkg/plugins/cache` | Redis adapters and shared SIWE nonces; canonical hot-cache generation acceptance remains pending |
| Serve | `pkg/plugins/api` | Canonical REST / GraphQL; WebSocket/SSE components, with live canonical subscription wiring pending |
| Safety | `pkg/services/finality`, `pkg/services/reorg` | Finality confidence; ordered, generation-scoped reorg repair |
| Observability | `pkg/observability` | Bounded metric labels and progress wiring; committed/finalized lag and cross-process trace acceptance pending |

One codebase, two deployment modes (monolithic PostgreSQL vs four native services; a separate in-memory playground is retained) —
[ADR-001](../adr/ADR-001-plugin-architecture.md).

The three correctness decisions are
[ADR-008](../adr/ADR-008-event-identity-canonical-digest.md) (event identity and
the canonical digest), [ADR-009](../adr/ADR-009-db-commit-kafka-progress.md)
(DB commit and Kafka progress), and
[ADR-010](../adr/ADR-010-finality-reorg-degradation.md) (finality and reorg
degradation).

## Evidence — what is actually covered today

The repository has runnable tests for the following. **No performance numbers
are published yet**: the benchmark gate and the 30-minute soak are pending, so
no `events/s` or P99 figure is claimed, and no mainnet capacity is extrapolated.

1. **Chain isolation** — `test/contracts/postgres_chain_isolation_test.go`: a
   rollback on one chain leaves another chain unchanged (invariant I01).
2. **Atomic commit and graceful shutdown** —
   `test/contracts/postgres_atomic_test.go` and
   `test/acceptance/sigterm_test.go`: events and the checkpoint commit together
   or not at all, and SIGTERM drains in-flight work instead of cutting it off.
3. **Redelivery idempotency and fault rehearsals** —
   `test/acceptance/kafka_redelivery_test.go`,
   `test/acceptance/postgres_fault_test.go`,
   `test/acceptance/redis_fault_test.go`: duplicate delivery is a no-op, and a
   DB or Redis outage has a rehearsed, asserted behaviour.

The current local execution record additionally covers two independent EVM
chains in both deployment modes, exact raw-log oracle digests, a real signed
Swap fixture and max-uint256 API values. Native four-service scenarios exercise
reorg depths 1/2/12, transaction re-inclusion, terminal orphan audit and durable
reorg notification publication. The repair-crash rehearsal additionally
verifies failed publication, SIGKILL, persisted replay with old-branch backlog,
terminal audit and actual owed notices after restart. See the named JSONL artifacts in the
[execution record](2026-10-14-live-delivery-execution.md).

These are dirty-working-tree results. The acceptance manifest still has pending
scenarios, including browser flow, sustained performance and real
release rollback. Kafka rebalance/revocation fencing, generation-safe hot caching
and canonical subscription recovery also need acceptance. Remote same-SHA checks,
immutable image digests and the final evidence pack are not complete. The
[SUPPORT_MATRIX](SUPPORT_MATRIX.md) separates local verification from release
acceptance.

## Support boundary — what this is *not*

- **Not a bridge** and does not custody assets or attest cross-chain messages.
- **Not a multi-chain production deployment.** The widest thing exercised is a
  two-independent-chain normal pipeline in both modes and isolated fault tests;
  production capacity is unmeasured.
- **Not an MEV mitigation** and **not an account-abstraction implementation**
  (see the security discussion outline for where the code is merely *adjacent*).
- **L2 finality is discounted, not guaranteed.**
- A chain whose endpoints never expose a finality tag stays permanently
  `degraded` — honest, but never promoted to final.

## Skills this demonstrates

- **Go** — `context` and goroutine lifecycle (drain), channel ownership,
  atomic check-and-mark under concurrency, error taxonomy for metrics,
  allocation-aware benchmarks.
- **EVM / Web3** — logs and topics, ABI decoding with shape validation, reorg
  identity, safe vs finalized, SIWE nonce replay.
- **PostgreSQL** — transactional atomicity, lease/generation fencing,
  natural-key dedup, transactional outbox.
- **Kafka** — partition-key design, at-least-once semantics, redelivery.
- **Redis** — shared atomic nonce consumption, fail-closed vs degrade policies.
- **Architecture** — hexagonal ports/adapters, ADR-driven decisions, CI gating
  that refuses false-green runs.

## Resume entry (template — filled only after T18)

> Built and verified an EVM event indexing backend in Go, covering PostgreSQL
> persistence, Kafka redelivery, a Redis cache and a query API; reached
> `[measured events/s and P99 on a stated machine and dataset]`; passed
> `[two-chain isolation, 12-block reorg, commit-crash]` fault acceptance; and
> shipped via `[same-SHA CI/CD and a rollback run]`.

The bracketed values stay empty until measured — filling them now would be a
claim without evidence.

## Scope and honesty

This is a **personal project**. It demonstrates the ability to design, build and
verify the system above. It is **not** a claim of production Web3 architect
experience, and its numbers are only meaningful once measured on a stated
machine and dataset.
