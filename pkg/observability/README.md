# Observability Package

Monitoring, metrics, tracing, and health checking for ChainPulse.

## Modules

### Metrics
- **indexer_metrics.go** - Indexer performance metrics
- **indexer_metrics_test.go** - Metrics tests

**Metrics Tracked**:
- Event collection rate
- Event processing latency
- Cache hit rate
- Query latency
- Error rate
- Block height

### Health Monitoring
- **indexer_health.go** - Indexer health status
- **indexer_health_test.go** - Health check tests

**Health Checks**:
- Database connectivity
- Cache connectivity
- Message queue connectivity
- Blockchain node connectivity
- Sync status

### Distributed Tracing
- **distributed_tracing.go** - Distributed tracing implementation
- **distributed_tracing_property_test.go** - Property-based tests

**Tracing Features**:
- Request tracing
- Span creation and management
- Trace context propagation
- Performance analysis

## Architecture

```
┌──────────────────────────────────────┐
│      Observability Manager           │
│  (Coordinates all monitoring)        │
└──────────────┬───────────────────────┘
               │
       ┌───────┴────────────────────────┐
       │                                │
   ┌───▼────┐  ┌────────┐  ┌────────┐  │
   │Metrics │  │Health  │  │Tracing │  │
   │Collector   │Monitor │  │Engine  │  │
   └────────┘  └────────┘  └────────┘  │
       │                                │
       └────────────────────────────────┘
               │
       ┌───────▼──────────────────────┐
       │  Export to Monitoring Stack  │
       │  (Prometheus, Jaeger, etc.)  │
       └──────────────────────────────┘
```

## Key Metrics

### Collection Metrics
- `chainpulse_events_collected_total` - Total events collected
- `chainpulse_events_collected_rate` - Events collected per second
- `chainpulse_collection_latency_ms` - Collection latency

### Processing Metrics
- `chainpulse_events_processed_total` - Total events processed
- `chainpulse_events_processed_rate` - Events processed per second
- `chainpulse_processing_latency_ms` - Processing latency

### Cache Metrics
- `chainpulse_cache_hits_total` - Total cache hits
- `chainpulse_cache_misses_total` - Total cache misses
- `chainpulse_cache_hit_rate` - Cache hit rate percentage

### Query Metrics
- `chainpulse_queries_total` - Total queries executed
- `chainpulse_query_latency_ms` - Query latency
- `chainpulse_query_errors_total` - Total query errors

### System Metrics
- `chainpulse_block_height` - Current block height
- `chainpulse_sync_lag_blocks` - Blocks behind latest
- `chainpulse_goroutines` - Active goroutines
- `chainpulse_memory_bytes` - Memory usage

## Health Status

### Healthy
All components operational:
- Database connected
- Cache connected
- Message queue connected
- Blockchain nodes connected
- Sync lag < 10 blocks

### Degraded
Some components operational:
- One or more connections failing
- Sync lag 10-100 blocks
- High error rate

### Unhealthy
Critical components down:
- Database disconnected
- Blockchain nodes disconnected
- Sync lag > 100 blocks

## Usage

### Collect Metrics

```go
import "chainpulse/pkg/observability"

metrics := observability.NewIndexerMetrics()

// Record event collection
metrics.RecordEventCollected()

// Record processing latency
metrics.RecordProcessingLatency(duration)

// Record cache hit
metrics.RecordCacheHit()
```

### Check Health

```go
health := observability.NewIndexerHealth(db, cache, mq, nodes)

// Get overall health
status := health.GetStatus(ctx)
if status.Status == "healthy" {
    log.Info("System is healthy")
}

// Get component health
dbHealth := health.CheckDatabase(ctx)
cacheHealth := health.CheckCache(ctx)
```

### Distributed Tracing

```go
tracer := observability.NewDistributedTracer()

// Start trace
span := tracer.StartSpan(ctx, "process_event")
defer span.End()

// Add attributes
span.SetAttribute("event_id", eventID)
span.SetAttribute("block_number", blockNumber)

// Create child span
childSpan := tracer.StartSpan(ctx, "decode_event")
defer childSpan.End()
```

## Configuration

Set environment variables:

```bash
# Metrics
export METRICS_ENABLED=true
export METRICS_PORT=9090
export METRICS_INTERVAL=60s

# Health Checks
export HEALTH_CHECK_ENABLED=true
export HEALTH_CHECK_INTERVAL=30s
export HEALTH_CHECK_TIMEOUT=10s

# Tracing
export TRACING_ENABLED=true
export TRACING_JAEGER_ENDPOINT=http://localhost:14268/api/traces
export TRACING_SAMPLE_RATE=0.1

# Logging
export LOG_LEVEL=info
export LOG_FORMAT=json
```

## Monitoring Stack Integration

### Prometheus

Export metrics to Prometheus:
```bash
curl http://localhost:9090/metrics
```

### Jaeger

View traces in Jaeger:
```
http://localhost:16686
```

### Grafana

Create dashboards in Grafana:
- Event collection rate
- Processing latency
- Cache hit rate
- Query latency
- System health

## Testing

Run observability tests:
```bash
go test ./pkg/observability/...
```

Run property-based tests:
```bash
go test ./pkg/observability/... -run Property
```

## Best Practices

1. **Record all operations** - Track metrics for all operations
2. **Use structured logging** - Include context in logs
3. **Trace critical paths** - Trace important operations
4. **Monitor health** - Check component health regularly
5. **Set alerts** - Alert on anomalies
6. **Aggregate metrics** - Use time-series database
7. **Visualize data** - Create dashboards
8. **Analyze trends** - Track performance over time
