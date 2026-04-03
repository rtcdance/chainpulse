# Operations Guide

**Status**: Active | **Last Updated**: 2026-03-30

## Quick Start

### Health Checks
```bash
# Check service health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:8080/metrics
```

### Common Operations

**Start Services**
```bash
make run-monolith    # Single process (dev)
make run-services    # Microservices (prod)
```

**Monitor Indexing**
```bash
# Check indexing lag
curl http://localhost:8080/metrics | grep block_indexing_lag

# View logs
tail -f logs/indexer.log
```

**Handle Incidents**
```bash
# Restart indexer
systemctl restart chainpulse-indexer

# Check circuit breaker status
curl http://localhost:8080/health/circuit-breaker
```

## Detailed Guides

- **Deployment**: `docs/guides/DEPLOYMENT_GUIDE.md`
- **Policy SLO**: `docs/operations/POLICY_ROLLOUT_SLO.md`
- **Policy Dashboard Queries**: `docs/operations/POLICY_DASHBOARD_QUERIES.md`
- **Policy Runbook**: `docs/operations/POLICY_ROLLOUT_RUNBOOK.md`
- **Policy Metric Versioning**: `docs/operations/POLICY_METRIC_VERSIONING.md`
- **Migration Manifest Governance**: `docs/operations/MIGRATION_MANIFEST.md`
- **Migration Governance Queries**: `docs/operations/MIGRATION_GOVERNANCE_DASHBOARD_QUERIES.md`
- **Migration Governance Changelog**: `docs/operations/MIGRATION_GOVERNANCE_CHANGELOG.md`
- **Migration Ticket Registry**: `docs/operations/MIGRATION_TICKET_REGISTRY.txt`
- **Migration Ticket Registry Health Baseline**: `docs/operations/MIGRATION_TICKET_REGISTRY_HEALTH_BASELINE.prom`
- **Migration Resolver Test Baseline**: `docs/operations/MIGRATION_BASELINE_RESOLVER_TEST_BASELINE.prom`
- **Ticket Registry Health Report (generated)**: `build/migration-governance/ticket-registry-health.md`
- **Ticket Registry Health Delta (generated)**: `build/migration-governance/ticket-registry-health-delta.md`
- **Baseline Governance Scope Smoke Test**: `scripts/smoke-baseline-governance-scope.sh`
- **Baseline Governance Scope Smoke Report (generated)**: `build/migration-governance/baseline-scope-smoke.md`
- **Baseline Governance Scope Smoke Delta (generated)**: `build/migration-governance/baseline-scope-smoke-delta.md`
- **Baseline Update Template Preview (generated)**: `build/migration-governance/baseline-update-template.md`
- **Baseline Update Preflight (generated)**: `build/migration-governance/baseline-update-preflight.md`
- **Baseline Resolver Test Report (generated)**: `build/migration-governance/baseline-resolver-test.md`
- **Baseline Resolver Test Delta (generated)**: `build/migration-governance/baseline-resolver-test-delta.md`
- **Debugging**: `docs/DEBUGGING.md`
- **Incident Response**: `.codex/skills/incident-postmortem-learning/SKILL.md`

## Key Metrics

| Metric | SLO | Alert Threshold |
|--------|-----|-----------------|
| Block indexing lag | <10 blocks | >100 blocks for 10m |
| RPC success rate | >99.5% | <95% for 5m |
| Event processing p95 | <5s | >10s for 5m |

## Runbooks

See:
- `docs/operations/POLICY_ROLLOUT_RUNBOOK.md`
- `docs/guides/DEPLOYMENT_GUIDE.md`
- `docs/DEBUGGING.md`

---

**Note**: Original 1019-line guide archived to `docs/archive/OPERATIONS_GUIDE_v1.md`
