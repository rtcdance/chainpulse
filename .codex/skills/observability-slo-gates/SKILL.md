---
name: "observability-slo-gates"
description: "Define metrics, health checks, and SLO-oriented alerts. Require actionable telemetry for chain-level operations. Invoke for production-facing feature changes, reliability changes, and performance-sensitive changes."
---

# Skill: observability-slo-gates

## Trigger

Use this skill for production-facing feature changes, reliability changes, and performance-sensitive changes.

## Must Do

1. Add actionable metrics, logs, and health checks for changed behavior.
2. Include core labels where applicable:
   - `chain_id`
   - `service`
   - `operation`
   - `block_height`
3. Define at least one SLI/SLO-aligned signal for the changed path.
4. Add alerting intent notes (what should page and why).
5. Update ops docs when runtime behavior changes.

### Web3-Specific Metrics

**Block Indexing Lag**
```go
var blockIndexingLag = prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: "chainpulse_block_indexing_lag_blocks",
        Help: "Blocks behind chain head",
    },
    []string{"chain_id"},
)

// SLO: <10 blocks lag for real-time chains
```

**RPC Success Rate**
```go
var rpcCallSuccess = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "chainpulse_rpc_calls_total",
    },
    []string{"chain_id", "method", "status"},
)

// SLO: >99.5% success rate (excluding rate limits)
```

**Event Processing Latency**
```go
var eventProcessingDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "chainpulse_event_processing_seconds",
        Buckets: []float64{0.1, 0.5, 1, 5, 10},
    },
    []string{"chain_id", "event_type"},
)

// SLO: p95 <5s for critical events
```

### Alert Fatigue Prevention

**Alert Grouping**
```yaml
# Group related alerts to prevent spam
groups:
  - name: chain_indexing
    interval: 5m
    rules:
      - alert: HighIndexingLag
        expr: chainpulse_block_indexing_lag_blocks > 100
        for: 10m  # Don't alert on transient spikes
        annotations:
          summary: "Chain {{ $labels.chain_id }} is {{ $value }} blocks behind"
```

**Actionable Alerts Only**
- ❌ Alert on every RPC timeout (too noisy)
- ✅ Alert when RPC error rate >5% for 5 minutes
- ❌ Alert on single block processing failure
- ✅ Alert when block processing stuck for 10+ minutes

## ChainPulse Pointers

- Observability code: `pkg/observability/*`
- Health checks: `pkg/infrastructure/health/*`
- Ops docs:
  - `docs/deployment/monitoring.md`
  - `docs/deployment/operations.md`
  - `docs/guides/OPERATIONS_GUIDE.md`

## Must Not

- No production changes without telemetry.
- No health endpoint updates without validation logic.
- No alerts without clear remediation steps.

## Exit Criteria

- New/changed behavior can be monitored and triaged.
- Web3-specific metrics added (lag, RPC success, latency).
- Alert grouping configured to prevent fatigue.
- Health/metrics docs updated for operators.

