# Architecture Decision Records

Each record states the alternatives considered, the decision, the failure
semantics, the tests that hold it, and what it costs. They are ordered by date;
the number is the identity.

| ADR | Title | Date | Status |
| --- | --- | --- | --- |
| [ADR-001](ADR-001-plugin-architecture.md) | Plugin Architecture — Monolith vs Microservices | 2026-03-15 | Accepted |
| [ADR-002](ADR-002-event-bus-pubsub.md) | Event Bus Publish/Subscribe Pattern | 2026-03-20 | Accepted |
| [ADR-003](ADR-003-reorg-rollback-reindex.md) | Reorg Handling Strategy — Rollback and Reindex | 2026-03-25 | Accepted |
| [ADR-004](ADR-004-env-var-naming-convention.md) | Environment Variable Naming Convention | 2026-05-05 | Accepted |
| [ADR-005](ADR-005-error-code-taxonomy.md) | Error Code Taxonomy | 2026-05-13 | Accepted |
| [ADR-006](ADR-006-api-versioning-strategy.md) | API Versioning Strategy | 2026-05-13 | Accepted |
| [ADR-007](ADR-007-ethclient-migration.md) | ethclient 迁移 — 手写 JSON-RPC → go-ethereum | 2026-05-13 | Accepted |
| [ADR-008](ADR-008-event-identity-canonical-digest.md) | Event Identity and the Canonical Digest | 2026-10-13 | Accepted |
| [ADR-009](ADR-009-db-commit-kafka-progress.md) | Database Commit and Kafka Progress Coordination | 2026-10-13 | Accepted |
| [ADR-010](ADR-010-finality-reorg-degradation.md) | Finality and Reorg Degradation | 2026-10-13 | Accepted |

## The three core decisions

ADR-008, ADR-009 and ADR-010 are the decisions the indexing correctness rests
on, and they reference each other:

- **ADR-008** defines what an event *is* (physical identity) and how to summarise
  what was indexed (canonical digest).
- **ADR-009** defines when a batch is *done* (atomic fenced commit) and how the
  queue's progress is coordinated with it (partition key = chain, at-least-once
  with dedup on the ADR-008 identity).
- **ADR-010** defines how *sure* the system is that a block will not change
  (finality confidence) and what happens when it does (ordered, generation-scoped
  reorg repair).

## Numbering note

`ADR-007` was originally written as a second `ADR-004`. It was renumbered to
resolve the conflict; [ADR-004](ADR-004-env-var-naming-convention.md) (2026-05-05)
predates it and keeps the number.
