# Engineering Constraint Framework

## Purpose

This framework defines how ChainPulse work is delivered with enterprise-level quality while supporting:

- Monolithic debug mode for fast local iteration.
- Microservice deployment mode for production.
- Web3 + Go backend upskilling through real project work.

The process is mandatory for all architecture and feature changes.

## 1) Spec Framework

Every work item must start from a lightweight spec (`Spec Card`) with the fields below.

- `Context`: why this change is needed now.
- `Scope`: exact modules and boundaries touched.
- `Non-Goals`: what is explicitly out of scope.
- `Acceptance`: observable outcomes and success criteria.
- `Risks`: failure modes and rollback strategy.
- `Verification`: required checks and test plan.

`Spec Card` is created in PR description or issue before coding starts.

### Mandatory Approval Gate

- Spec document path: `docs/specs/<yyyy-mm-dd>-<short-topic>.md`
- Template: `docs/specs/TEMPLATE.md`
- Required status flow: `Draft -> In Review -> Approved -> Implemented`
- Hard rule: coding starts only when spec status is `Approved`

## 2) Skills Framework

The project uses capability tracks so learning and delivery move together.

- `Architecture`: DDD boundaries, ports/adapters, mono+micro dual-mode composition.
- `Web3 Data`: chain finality, reorg rollback, event idempotency, RPC failover.
- `Go Backend`: concurrency, context propagation, error classification, resilience patterns.
- `Quality`: static analysis, lint discipline, deterministic tests, CI parity.
- `Operations`: metrics, tracing, health checks, SLO/SLI thinking, runbook basics.

For each feature, record at least one skill gain in PR notes:

- `What was learned`
- `Where it was applied in code`
- `How it was validated`

## 3) Workflow Framework

Use micro-cycles until completion. One cycle = one small, verifiable increment.

1. Write or update `Spec Card`.
2. Slice the change into a small step (single behavior or single module boundary).
3. Implement the minimum change.
4. Add/update unit tests for the changed behavior.
5. Run `fast gate` checks locally.
6. Fix issues immediately.
7. Repeat cycle until acceptance criteria are met.
8. Run `full gate` before merge.

## 4) Quality Gates (Mandatory)

### Fast Gate (each micro-cycle)

- Format and style checks.
- Static checks on changed scope.
- Unit tests on changed packages.

Recommended command:

```bash
scripts/dev-micro-loop.sh --mode fast
```

### Full Gate (before merge)

- Format check.
- `golangci-lint`.
- `go vet`.
- `staticcheck`.
- Unit tests and relevant integration tests.

Recommended command:

```bash
scripts/dev-micro-loop.sh --mode full
```

## 5) Test Policy

- Unit tests are mandatory for all changed logic.
- Bug fixes must include regression tests.
- New adapters require contract tests.
- Reorg, retry, and idempotency paths must have failure-path tests.
- If tests cannot be added, the PR must explain why and list follow-up actions.

## 6) Definition of Done

A change is complete only when all are true:

- Spec acceptance criteria are satisfied.
- Fast gate passed in each iteration.
- Full gate passed before merge.
- Unit tests were added/updated.
- Docs updated for architecture or behavior changes.
- Operational impact is documented (metrics, alerts, rollback).

## 7) PR Checklist

- `Spec Card` included and scoped correctly.
- Tests cover happy path and at least one failure path.
- No business logic leaked into bootstrap/platform/adapters boundaries.
- Metrics/logging include `chain_id`, `service`, `operation` where applicable.
- Risk and rollback notes are present.

## 8) Escalation Rules

Stop and escalate design review when any of the following occurs:

- Domain boundary changes across services.
- Data model or schema changes impacting compatibility.
- Reorg/finality behavior changes.
- SLO-impacting performance regressions.
- Security-sensitive configuration or credential handling updates.

## 9) Single Source of Truth

- Architecture prompt: `docs/ARCHITECTURE_PROMPT.md`
- Unit test standards: `docs/guides/UNIT_TEST_STANDARDS.md`
- Error handling patterns: `docs/archive/ERROR_HANDLING_GUIDE.md`
- This workflow contract: `docs/guides/ENGINEERING_CONSTRAINT_FRAMEWORK.md`
