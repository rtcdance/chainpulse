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
- **Monitoring**: `monitoring/README.md`
- **Debugging**: `docs/DEBUGGING.md`
- **Incident Response**: `.codex/skills/incident-postmortem-learning/SKILL.md`

## Key Metrics

| Metric | SLO | Alert Threshold |
|--------|-----|-----------------|
| Block indexing lag | <10 blocks | >100 blocks for 10m |
| RPC success rate | >99.5% | <95% for 5m |
| Event processing p95 | <5s | >10s for 5m |

## Runbooks

See `monitoring/runbooks/` for:
- High indexing lag
- RPC failures
- Database connection issues
- Memory leaks

---

**Note**: Original 1019-line guide archived to `docs/archive/OPERATIONS_GUIDE_v1.md`
