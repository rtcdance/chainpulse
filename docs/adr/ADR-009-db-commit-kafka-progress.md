# ADR-009: Database Commit and Kafka Progress Coordination

**Date**: 2026-10-13

### Status

Accepted

### Context

Indexing is a two-system write: events and the chain checkpoint go to
PostgreSQL, and the work is driven by Kafka (or an equivalent queue). The
question this ADR answers is **when a batch may be considered done**, so that a
crash at any point neither loses events nor leaves the checkpoint ahead of the
data.

Before this decision the two writes were separate: "store the events" and
"advance the checkpoint" were independent steps, and nothing tied them to the
queue's progress or to chain ownership. That produced three concrete failures
(findings F04, F05, F28):

- **F05 (split commit)** — a crash between "events written" and "checkpoint
  advanced" either lost the batch (checkpoint behind, events present but
  unreachable) or replayed it.
- **F04 (two writers)** — a Kafka rebalance could hand a chain to a second
  worker while the first was still committing, so two writers could interleave
  on one chain.
- **F28 (partition key drift)** — nothing froze the rule that a chain's messages
  land on one partition, so ordering was accidental rather than guaranteed.

### Options considered

1. **Commit the DB, then ack Kafka.** Crash between commit and ack → the batch
   is redelivered. Correct only if the write path is idempotent. This is the
   shape chosen, *conditional* on idempotency and on the checkpoint advancing
   in the same transaction as the events.
2. **Ack Kafka, then commit the DB.** Crash between ack and commit → the batch
   is gone forever. Rejected: it trades a duplicate for a loss, and a loss is
   not recoverable.
3. **Two-phase commit across PostgreSQL and Kafka.** Kafka has no XA
   participant; the operational cost is high and the failure modes are worse
   than at-least-once. Rejected.
4. **Exactly-once via Kafka transactions only.** Does not cover the database
   write, which is the thing that must be atomic with the checkpoint. Rejected.

### Decision

**A batch commits as one fenced transaction, and the queue is allowed to
redeliver.**

- **Atomic commit** — `processor.BatchCommitter.CommitBlock`
  (`pkg/services/processor/batch_committer.go`) writes the block's events and
  advances the checkpoint to `(block_number, block_hash)` inside a single
  transaction. They commit or roll back together (I05).
- **Fencing** — the transaction begins under a chain lease
  (`database.AcquireLease` → `BeginFencedTx`). The lease carries an owner token
  and a **monotonic generation** (`pkg/plugins/database/postgres_lease.go`); a
  commit that starts after another owner took the lease is rejected with
  `ErrLeaseLost`. A chain therefore has exactly one writer.
- **Partition key = chain key** — frozen in `mq.PartitionKeyForChain`
  (`pkg/plugins/mq/partition_contract.go`). Every message for a chain lands on
  one partition and is consumed in offset order, so chain-internal ordering does
  not depend on the topic's partition count. `TestPartitionKeyContract` fails if
  a caller sets the key by hand.
- **At-least-once with idempotent dedup** — redelivery is expected. The
  natural-key unique constraint plus `ON CONFLICT DO NOTHING` in the database is
  the dedup mechanism (ADR-008 identity), not a pre-check in the committer.
- **Outbox for side effects** — reorg notifications are written to a
  transactional outbox and published by a drainer
  (`pkg/plugins/database/postgres_reorg_outbox.go`), so a notification is never
  emitted for a transaction that did not commit.
- **Drainable** — `BatchCommitter` consults a work gate
  (`bootstrap.DrainController`). Once the gate closes, no new batch starts; the
  batch already running is allowed to finish (SIGTERM handling).

The commit order is therefore: **begin fenced tx → write events → advance
checkpoint → commit → (later) ack the queue**. A crash before commit replays
from the checkpoint; a crash after commit before ack redelivers, and dedup makes
that a no-op.

### Failure semantics

- **Crash before commit** — nothing is visible; the batch replays from the last
  checkpoint. Safe.
- **Crash after commit, before ack** — redelivery; the second attempt is a
  no-op because the identities already exist. Safe.
- **Lease lost mid-flight** (rebalance) — `BeginFencedTx` returns
  `ErrLeaseLost`; the caller must stop committing for that chain. The new owner
  resumes from the checkpoint.
- **Empty or all-duplicate block** — the checkpoint **still advances**, because
  otherwise the chain would stall at that height forever. This is deliberate and
  tested (`EmptyBlockStillAdvancesCheckpoint`,
  `DuplicateOnlyBatchStillAdvancesCheckpoint`).

### Tests

- `pkg/services/processor/batch_committer_test.go` —
  `CommitsEventsAndCheckpointTogether`, `RollsBackOnCheckpointFailure`,
  `RollsBackOnStoreFailure`, `SurfacesLeaseLoss`, `RenewsLeaseLazily`,
  `RejectsNewWorkWhileDraining`.
- `pkg/services/processor/idempotency_test.go`,
  `idempotency_toctou_test.go` — `CheckAndMark_ConcurrentSingleWinner` (one
  winner under concurrency), multi-chain isolation.
- `pkg/plugins/mq/partition_contract_test.go` — `TestPartitionKeyContract`.
- T16 rehearsals — real DB outage, Kafka redelivery, kill-and-resume.

### Consequences

- **Positive** — the checkpoint can never be ahead of the data (I05), and the
  queue may be at-least-once without risking loss. One writer per chain makes
  reasoning about ordering local to a chain. The partition contract is frozen in
  code and test, so ordering is a property, not a coincidence.
- **Negative** — every consumer of the queue must be idempotent; correctness now
  depends on the ADR-008 identity being used everywhere. The outbox adds a table
  and a drainer process.
- **Neutral** — per-chain parallelism is capped at one committer by design. That
  is a deliberate limit: throughput scales by adding *chains*, not by adding
  writers to one chain.

### Implementation acceptance note — 2026-10-15

The PostgreSQL native consumer now checks the RPC canonical header inside its
fenced transaction. Expired owners cannot begin transactions. The producer
compares durable checkpoint hashes every poll and uses an expected-checkpoint
rollback guard; rejected orphan frames are acknowledged only after a terminal
audit write, and never published as canonical commits. Actual four-service
reorg depths 1/2/12 and outbox publication have local process evidence in the
[execution record](../project/2026-10-14-live-delivery-execution.md).

Database owner fencing is implemented; binding ownership to Kafka group
revocation and proving the old worker cannot commit after an actual rebalance
remain required T05 work. The lease's stored generation alone does not prove
that broker lifecycle property. These local tests do not complete T18.
