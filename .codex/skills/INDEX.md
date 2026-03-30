# ChainPulse Skill Index

## Available Skills

0. `design-review-gate` (MANDATORY)
   - No coding before spec doc is approved.
   - Spec path: `docs/specs/<yyyy-mm-dd>-<topic>.md`

1. `web3-go-architecture-guardrails`
   - Keep domain/application/adapters/platform boundaries clean.
   - Preserve monolith debug and microservice deploy consistency.

2. `web3-reorg-idempotency`
   - Implement and verify reorg-safe indexing and idempotent writes.
   - Enforce rollback and replay correctness.

3. `adapter-contract-testing`
   - Add or update adapter contract tests for DB/MQ/Cache/RPC/API implementations.
   - Ensure behavior parity across in-memory and production adapters.

4. `observability-slo-gates`
   - Define metrics, health checks, and SLO-oriented alerts.
   - Require actionable telemetry for chain-level operations.

5. `micro-loop-delivery`
   - Execute spec-first, test-first micro-cycles with mandatory quality gates.
   - Run `scripts/dev-micro-loop.sh` in fast/full modes.

6. `security-compliance-baseline`
   - Prevent secret leakage and unsafe privilege patterns.
   - Enforce security review notes for sensitive changes.

7. `schema-migration-safety`
   - Keep schema evolution reversible and compatibility-safe.
   - Require migration + rollback + integrity verification.

8. `api-contract-compatibility`
   - Preserve API backward compatibility by default.
   - Require versioning/migration plan for breaking changes.

9. `performance-capacity-guardrails`
   - Measure hot-path changes and prevent unbounded resource patterns.
   - Track queue lag, indexing delay, and query latency.

10. `release-rollback-readiness`
   - Require explicit rollout and rollback steps.
   - Require release-window verification and alert watchpoints.

11. `incident-postmortem-learning`
   - Convert incidents into tests + telemetry + corrective actions.
   - Prevent recurrence, not just patch symptoms.

12. `deterministic-testing`
   - Keep tests reproducible and non-flaky.
   - Require seed/time/input determinism for replay.

13. `concurrency-safety`
   - Enforce bounded concurrency and lifecycle ownership.
   - Prevent races, leaks, and deadlocks in changed paths.

14. `event-ordering-finality`
   - Make ordering/finality semantics explicit per chain.
   - Test out-of-order, duplicate, and reorg scenarios.

15. `state-checkpoint-recovery`
   - Keep checkpoint/replay behavior deterministic and recoverable.
   - Define and validate restart/recovery path.

16. `dependency-upgrade-governance`
   - Control dependency upgrades with impact review and rollback plan.
   - Avoid broad or unverified dependency drift.

17. `chaos-resilience-testing`
   - Verify recovery under RPC failures, timeouts, and network partitions.
   - Inject failures and validate circuit breakers.

18. `gas-cost-optimization`
   - Minimize RPC call costs via batching, caching, and filtering.
   - Track cost metrics and enforce budgets.

19. `multi-chain-consistency`
   - Handle chain-specific finality, reorg depth, and consensus differences.
   - Use unified abstractions with chain-specific configs.

20. `data-retention-archival`
   - Define hot/warm/cold storage tiers for blockchain data growth.
   - Implement archival automation and query layer.

21. `rate-limiting-backpressure`
   - Handle RPC provider rate limits with backoff and flow control.
   - Prevent queue overflow and unbounded concurrency.

22. `code-organization-placement`
   - Enforce clean file organization and directory structure.
   - Prevent random file placement and maintain navigability.

## How To Apply

- Pick one or more skills before coding.
- Always activate `design-review-gate` first.
- Follow `Trigger -> Must Do -> Exit Criteria`.
- Do not mark task complete unless all active skill exit criteria pass.
