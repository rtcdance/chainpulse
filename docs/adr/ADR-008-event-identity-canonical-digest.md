# ADR-008: Event Identity and the Canonical Digest

**Date**: 2026-10-13

### Status

Accepted

### Context

The indexer needs two things that look similar but are not:

1. **A physical identity for an event** — one stable key that says "this is the
   same log", used for duplicate suppression on redelivery and for telling two
   chains apart.
2. **A reproducible summary of "what was indexed"** — a single value that two
   runs, two nodes, or two processes can compare to decide whether they agree,
   used by the acceptance scenarios (E01–E13) and by recovery checks.

The second is harder than it looks. Comparing raw row dumps is brittle: SQL
gives no row order, Kafka redelivery duplicates rows, and a reorg rewrites
history. A summary that changes when the input order changes is not a summary
of *what was indexed*, it is a summary of *how the rows happened to arrive*.

Findings that motivated this decision:

- **F06** — event identity was derived from `block_number`, so a reorg that
  replaced a block at the same height produced the *same* identity for two
  different events. The old and new logs collided.
- **F21** — the acceptance toolchain referenced a "canonical digest oracle" in
  §8 and E01–E15, but no task implemented it, so T18 would have hand-assembled
  unverifiable evidence.

### Options considered

**For the event identity:**

1. **`(block_number, tx_hash, log_index)`** — simple, but a reorg replaces a
   block at the same height, so a replaced log keeps its old identity and
   collides with its replacement (F06). Rejected.
2. **Auto-increment / row ID** — not stable across replay, not comparable
   between two databases, and not known before insert. Rejected.
3. **`(chain_id, block_hash, tx_hash, log_index)`** — the *physical* identity.
   A block hash changes when the block is replaced, so a reorg naturally
   produces a new identity and the old one is distinct. Chain id is included so
   the same transaction on two chains never collides (I01). Chosen.

**For the digest:**

1. **Hash of rows in query order** — order is undefined, so the digest is not
   stable. Rejected.
2. **Hash of concatenated field values** — ambiguous: `"ab"+"c"` and `"a"+"bc"`
   produce the same bytes. Rejected.
3. **Hash of a stable-sorted, deduplicated, length-prefixed canonical form** —
   order-independent, idempotent under redelivery, and unambiguous. Chosen.

### Decision

**Event identity** is `(chain_id, block_hash, tx_hash, log_index)`, computed by
`core.ComputeEventHash` (`pkg/core/event_hash.go`) as
`sha256("chain:block_hash:tx_hash:log_index")`. `block_hash` is part of the
identity, not a derived field; `block_number` is deliberately excluded because
it is implied by the block hash and is only used for ordering and range scans.

**The canonical digest** is defined once in `test/acceptance/tools/digest`
(`CanonicalEvents`, `CanonicalBlocks`):

- deduplicate by identity, and treat two records with the same identity but a
  different payload as a **conflict** (an error), never silently resolved;
- sort the surviving identities (stable order);
- hash length-prefixed fields, so concatenation cannot be ambiguous;
- hash the block set by `(chain, height)` and reject two canonical blocks at the
  same height with different hashes (I03).

The digest is a **verification oracle**, not a storage format. Nothing in the
write path depends on it; it exists so assertions compare like with like.

### Failure semantics

- **Duplicate redelivery** — collapses to one record; the digest is unchanged.
  This is what makes at-least-once delivery (ADR-009) safe to verify.
- **Same identity, different payload** — a hard error. The system fails closed
  rather than picking one arbitrarily, because picking one would hide a
  producer bug.
- **Empty chain key** — an error (I01): an event that cannot be attributed to a
  chain is a bug, not a record.
- **Two canonical blocks at one `(chain, height)`** — an error (I03).

### Tests

- `pkg/core/event_hash_test.go` — determinism, `DistinguishesBlockHash`,
  `IgnoresBlockNumber`, `IgnoresDerivedFields`, nil/zero handling.
- `test/acceptance/tools/digest/digest_test.go` — `OrderIndependent`,
  `DuplicateRedeliveryIsIdempotent`, `ConflictingPayloadFails`,
  `ChainIsolation`, `DistinguishesLogIndex`, `DistinguishesBlockHash`,
  `CanonicalBlocksRejectsTwoCanonicalBlocksAtSameHeight`.

### Consequences

- **Positive** — one identity function is used by every component (pullers,
  idempotency, processor, database dedup), so duplicate detection is consistent
  by construction. Reorgs are handled by identity change rather than by
  special-casing. The digest makes acceptance assertions reproducible.
- **Negative** — `block_hash` must be stored on every event, and it changes on
  reorg, which means a reorg rewrites the identity of the affected events. That
  is the intended behaviour, but it means any external consumer that cached an
  identity across a reorg must refresh.
- **Neutral** — `block_number` is no longer part of identity but remains an
  ordering/scan column, so the schema keeps both.
