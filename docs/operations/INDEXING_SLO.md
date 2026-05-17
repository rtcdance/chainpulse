# Indexing Service SLO

**Status**: Active | **Last Updated**: 2026-05-13

## Scope

This SLO set governs the core indexing pipeline: block pulling, event decoding, persistence, and API query availability. These SLIs are instrumented via the RED metrics framework (`pkg/observability/red_metrics.go`).

## Service Level Indicators (SLI)

### 1. Indexing Latency (Freshness)

Time from a block being produced on chain to being indexed by ChainPulse.

- **Definition**: `chainpulse_puller_events_total` timestamps vs block timestamps
- **Measurement**: `current_chain_block - last_indexed_block` (block gap)
- **Source metric**: `chainpulse_indexer_blocks_total` with `status=ok`
- **Ideal**: < 30s behind chain head (P99)

### 2. Event Pull Success Rate

Ratio of successful pull cycles vs total pull cycles.

- **Definition**: `chainpulse_puller_requests_total / (puller_requests + puller_errors)`
- **Source metric**: `chainpulse_puller_requests_total{status=ok}` and `chainpulse_puller_errors_total`
- **Ideal**: >= 99.5% over 1h rolling window

### 3. RPC Call Success Rate

Ratio of successful Ethereum RPC calls.

- **Definition**: `chainpulse_rpc_calls_total / (rpc_calls + rpc_errors)`
- **Source metric**: `chainpulse_rpc_calls_total{status=ok}` and `chainpulse_rpc_errors_total`
- **Error breakdown**: tagged by `error_code` (RPC_RATE_LIMITED, TIMEOUT, NETWORK_ERROR...)

### 4. API Availability

Ratio of successful API responses.

- **Definition**: `chainpulse_api_requests_total` with `status=ok`
- **Source metric**: `chainpulse_api_requests_total{status=ok}`
- **Ideal**: >= 99.9% over 24h rolling window

### 5. Data Consistency

Ratio of events that pass reorg consistency checks.

- **Definition**: `reorg_detected / total_blocks_indexed`
- **Source metric**: reorg handler counters
- **Ideal**: reorg rate < 0.1% (for Ethereum PoS with 32-slot safe window)

## SLO Targets

| SLI | Target | Window | Error Budget | Severity |
|-----|--------|--------|-------------|----------|
| Indexing Latency | P99 < 30s behind chain head | 1h rolling | 30s | Critical |
| Pull Success Rate | >= 99.5% | 1h rolling | 0.5% | High |
| RPC Success Rate | >= 99% | 1h rolling | 1% | High |
| API Availability | >= 99.9% | 24h rolling | 0.1% | Critical |
| Data Consistency | reorgs < 0.1% of blocks | 24h rolling | 0.1% | Medium |

## Burn-Rate Alert Rules

### Critical (1h window, 5x burn rate)

```
- alert: IndexingLatencyCritical
  expr: time() - chainpulse_indexer_blocks_total{status="ok"} offset 1h > 150
  for: 5m
  labels: { severity: critical }
  annotations: { summary: "Indexing lag > 150 blocks for 5 minutes" }

- alert: APIAvailabilityCritical
  expr: rate(chainpulse_api_requests_total{status="error"}[5m]) / rate(chainpulse_api_requests_total[5m]) > 0.05
  for: 5m
  labels: { severity: critical }
```

### Warning (6h window, 2x burn rate)

```
- alert: PullSuccessRateWarning
  expr: rate(chainpulse_puller_errors_total[30m]) / rate(chainpulse_puller_requests_total[30m]) > 0.01
  for: 10m
  labels: { severity: warning }
```

## Owner

indexer-team

## Delivery Status

Implemented (metrics instrumented), alert rules need Prometheus deployment
