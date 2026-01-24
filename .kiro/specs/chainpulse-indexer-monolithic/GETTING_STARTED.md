# ChainPulse Indexer Monolithic Service - Getting Started

## Quick Overview

This guide helps you understand and start working on the ChainPulse Indexer Monolithic Service spec.

## What is This Spec?

This spec defines how to build an integrated indexer service that:
- Collects blockchain events from multiple chains
- Processes them through a validated pipeline
- Stores them in PostgreSQL with Redis caching
- Exposes them via multi-protocol API (GraphQL, gRPC, HTTP, WebSocket)

## Key Documents

1. **requirements.md** - What we're building (10 requirements)
2. **design.md** - How we're building it (architecture, components, data models)
3. **tasks.md** - Implementation tasks (12 tasks across 5 phases)
4. **SUMMARY.md** - High-level overview
5. **GETTING_STARTED.md** - This document

## Understanding the System

### The Big Picture

```
Blockchains → Data Pullers → Kafka → Event Processor → Database + Cache → Query Service → API Gateway → Clients
```

### The 5 Phases

**Phase 1: Core Indexer Service** (8 hours)
- Create the main indexer service
- Build the event processing pipeline
- Connect data pullers to the pipeline

**Phase 2: Query Service Enhancement** (5 hours)
- Implement cache-first pattern
- Add filtering and pagination

**Phase 3: API Gateway Integration** (8 hours)
- Enhance GraphQL, gRPC, HTTP handlers
- Ensure all protocols work

**Phase 4: Monitoring** (4 hours)
- Health checks
- Metrics collection

**Phase 5: Testing & Documentation** (7 hours)
- Integration tests
- Documentation

## Key Concepts

### Cache-First Pattern
```
Query arrives
    ↓
Check Redis cache
    ├─ HIT → Return immediately (<10ms)
    └─ MISS → Query PostgreSQL
              ↓
              Cache result
              ↓
              Return (<500ms)
```

### Event Processing Pipeline
```
Raw Event (from blockchain)
    ↓
Validate schema
    ↓
Decode using ABI
    ↓
Enrich with metadata
    ↓
Write to PostgreSQL
    ↓
Update Redis cache
    ↓
Update metrics
```

### Multi-Protocol API
```
GraphQL Client → GraphQL Handler ↘
gRPC Client    → gRPC Handler    → Unified Query Service → Cache/Database
HTTP Client    → HTTP Handler    ↗
WebSocket      → WebSocket Handler
```

## Implementation Roadmap

### Week 1: Core Service (Phase 1)
- **Day 1-2**: Task 1 - Indexer skeleton
- **Day 3-4**: Task 2 - Event processing pipeline
- **Day 5**: Task 3 - Data puller integration

### Week 2: Query & API (Phases 2-3)
- **Day 1-2**: Task 4 - Cache-first pattern
- **Day 3**: Task 5 - Filtering & pagination
- **Day 4-5**: Task 6 - GraphQL enhancement

### Week 3: gRPC & HTTP (Phase 3 continued)
- **Day 1-2**: Task 7 - gRPC handler
- **Day 3**: Task 8 - HTTP REST handler
- **Day 4-5**: Task 9 - Health checks

### Week 4: Monitoring & Testing (Phases 4-5)
- **Day 1**: Task 10 - Metrics endpoint
- **Day 2-3**: Task 11 - Integration tests
- **Day 4-5**: Task 12 - Documentation

## Starting Task 1: Indexer Service Skeleton

### What You'll Create

File: `pkg/services/indexing/monolithic_indexer.go`

```go
type MonolithicIndexer struct {
    logger       core.Logger
    metrics      core.MetricsCollector
    registry     core.PluginRegistry
    cache        plugins.CachePlugin
    database     plugins.DatabasePlugin
    mq           plugins.MQPlugin
    pullers      plugins.DataPuller
    processor    *EventProcessor
    queryService query.QueryService
    gateway      *api.APIGatewayPlugin
}

func (m *MonolithicIndexer) Initialize(config *core.Config) error {
    // Initialize all components in order
}

func (m *MonolithicIndexer) Start() error {
    // Start all components
}

func (m *MonolithicIndexer) Stop() error {
    // Stop all components gracefully
}
```

### Initialization Order

1. Logger
2. Metrics
3. Registry
4. Cache
5. Database
6. Message Queue
7. Data Pullers
8. Event Processor
9. Query Service
10. API Gateway

### Key Methods

- `Initialize(config)` - Set up all components
- `Start()` - Start all components
- `Stop()` - Graceful shutdown
- `IsRunning()` - Check if service is running
- `GetStatus()` - Get component status

## Common Patterns

### Error Handling
```go
if err := component.Initialize(config); err != nil {
    logger.Error("Failed to initialize component", "error", err.Error())
    return fmt.Errorf("initialization failed: %w", err)
}
```

### Graceful Shutdown
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
<-sigChan
indexer.Stop()
```

### Metrics Tracking
```go
metrics.IncrementCounter("events_processed", 1)
metrics.RecordLatency("query_latency_ms", latency)
```

## Testing Strategy

### Unit Tests
- Test each component independently
- Mock dependencies
- Test error cases

### Integration Tests
- Test components working together
- Use Docker containers for dependencies
- Test end-to-end flows

### Performance Tests
- Measure latency
- Measure throughput
- Verify cache hit rate

## Debugging Tips

### Enable Debug Logging
```bash
LOG_LEVEL=debug ./chainpulse
```

### Check Component Status
```bash
curl http://localhost:8080/health
```

### View Metrics
```bash
curl http://localhost:8080/metrics
```

### Check Kafka Topics
```bash
kafka-topics --list --bootstrap-server localhost:9092
```

### Check PostgreSQL
```bash
psql postgres://localhost/chainpulse
SELECT COUNT(*) FROM events;
```

### Check Redis
```bash
redis-cli
> KEYS *
> GET event:123
```

## Common Issues

### Issue: Service won't start
**Solution**: Check logs with `LOG_LEVEL=debug`, verify all dependencies are running

### Issue: Events not being processed
**Solution**: Check Kafka is running, verify data puller is connected, check event processor logs

### Issue: Queries are slow
**Solution**: Check cache hit rate in metrics, verify database indexes exist, check query logs

### Issue: Out of memory
**Solution**: Reduce cache size, reduce batch size, increase worker pool size

## Resources

### Existing Code
- `pkg/plugins/cache/` - Cache implementations
- `pkg/plugins/database/` - Database implementations
- `pkg/plugins/mq/` - Message queue implementations
- `pkg/plugins/pullers/` - Data puller implementations
- `pkg/plugins/api/` - API gateway implementations
- `pkg/services/query/` - Query service interface

### Documentation
- `docs/progress/` - Previous work summaries
- `docs/guides/` - Implementation guides
- `docs/architecture/` - Architecture documentation

### Tests
- `test/integration/` - Integration tests
- `test/fixtures/` - Test fixtures

## Next Steps

1. **Read requirements.md** - Understand what we're building
2. **Read design.md** - Understand how we're building it
3. **Review tasks.md** - See the implementation plan
4. **Start Task 1** - Create the indexer service skeleton
5. **Ask questions** - If anything is unclear

## Questions?

If you have questions about:
- **Requirements**: See requirements.md
- **Architecture**: See design.md
- **Implementation**: See tasks.md
- **Existing code**: Check the files in `pkg/`
- **Previous work**: Check `docs/progress/`

## Success Indicators

You'll know you're on the right track when:

- ✅ Service starts without errors
- ✅ All components initialize in correct order
- ✅ Events flow from puller → Kafka → processor → database → cache
- ✅ Queries return results from cache
- ✅ All API protocols work
- ✅ Health check endpoint works
- ✅ Metrics are collected
- ✅ Graceful shutdown works

Good luck! 🚀

