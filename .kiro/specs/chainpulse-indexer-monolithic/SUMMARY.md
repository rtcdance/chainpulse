# ChainPulse Indexer Monolithic Service - Spec Summary

## Overview

This spec defines the creation of an integrated monolithic indexer service that combines all verified plugins (Cache, Database, Message Queue, Data Puller, API Gateway) into a single cohesive system for indexing blockchain events across multiple chains.

## What We're Building

A production-ready Web3 event indexing service that:

1. **Collects** blockchain events from multiple chains (Ethereum, Polygon, Arbitrum, etc.)
2. **Processes** events through a validated pipeline (validation → decoding → enrichment → storage)
3. **Caches** frequently accessed data in Redis for performance
4. **Persists** events durably to PostgreSQL
5. **Queries** indexed events via unified interface with cache-first pattern
6. **Exposes** multi-protocol API (GraphQL, gRPC, HTTP, WebSocket)
7. **Monitors** system health and performance with comprehensive metrics

## Key Features

### Multi-Chain Support
- Simultaneous indexing of multiple blockchains
- Support for gRPC, HTTPS JSON-RPC, and WebSocket JSON-RPC protocols
- Automatic retry with exponential backoff
- Chain reorganization (reorg) handling

### Event Processing Pipeline
- Schema validation
- ABI-based event decoding
- Metadata enrichment (token info, pool info, etc.)
- Asynchronous processing via Kafka
- Dead-letter queue for failed events

### High-Performance Query Service
- Cache-first pattern (Redis → PostgreSQL)
- Target cache hit rate: 70-85%
- Query latency: <100ms (cache), <500ms (database)
- Comprehensive filtering and pagination
- Consistent results across all protocols

### Multi-Protocol API Gateway
- **GraphQL**: Full query language with subscriptions
- **gRPC**: Protocol Buffer RPC with streaming
- **HTTP**: REST API with JSON responses
- **WebSocket**: Real-time event subscriptions

### Observability
- Health check endpoint (`/health`)
- Prometheus metrics endpoint (`/metrics`)
- Comprehensive logging with structured format
- Performance tracking and alerting

## Architecture Highlights

```
Blockchain Nodes (Multiple Chains)
    ↓
Data Pullers (gRPC/HTTPS/WebSocket)
    ↓
Kafka Message Queue
    ↓
Event Processing Pipeline (Validation → Decoding → Enrichment)
    ↓
PostgreSQL Database + Redis Cache
    ↓
Unified Query Service (Cache-First Pattern)
    ↓
Multi-Protocol API Gateway (GraphQL/gRPC/HTTP/WebSocket)
    ↓
API Consumers
```

## Implementation Phases

### Phase 1: Core Indexer Service (8 hours)
- Create indexer service skeleton
- Implement event processing pipeline
- Integrate data pullers with event pipeline

### Phase 2: Query Service Enhancement (5 hours)
- Implement cache-first query pattern
- Add query filtering and pagination

### Phase 3: API Gateway Integration (8 hours)
- Enhance GraphQL handler
- Implement gRPC handler
- Implement HTTP REST handler

### Phase 4: Monitoring and Observability (4 hours)
- Implement health check endpoint
- Implement metrics endpoint

### Phase 5: Testing and Documentation (7 hours)
- Create integration tests
- Create documentation and quick start guide

**Total Estimated Effort**: 32 hours

## Performance Targets

- **Event Collection**: 1,000+ events/sec per chain
- **Event Processing**: <1ms per event
- **Database Write**: 1-5ms per event
- **Database Read**: 0.5-2ms per event
- **Cache Hit**: <10ms
- **Cache Miss**: <500ms
- **Message Queue**: 10,000+ messages/sec
- **Cache Hit Rate**: 70-85%

## Configuration

All behavior is controlled via environment variables:

```bash
# Chains to index
CHAINS=ethereum,polygon,arbitrum

# Blockchain node URLs
BLOCKCHAIN_NODE_URLS=http://eth-node:8545,http://poly-node:8545,http://arb-node:8545

# Data collection
DATA_PULLER_TYPE=https-jsonrpc  # grpc, https-jsonrpc, websocket-jsonrpc

# Message queue
MQ_TYPE=kafka
MQ_CONNECTION_URL=localhost:9092

# Cache
CACHE_TYPE=redis
CACHE_CONNECTION_URL=localhost:6379

# Database
DATABASE_TYPE=postgres
DATABASE_URL=postgres://localhost/chainpulse

# API
API_PORT=8080

# Processing
WORKER_POOL_SIZE=8
BATCH_SIZE=100

# Logging
LOG_LEVEL=info
```

## Success Criteria

After implementation:

- ✅ Service starts successfully with all components initialized
- ✅ Events are collected from multiple chains in parallel
- ✅ Events flow through complete pipeline
- ✅ Query service returns results from cache (70-85% hit rate)
- ✅ All 4 API protocols work correctly
- ✅ Health check endpoint reports component status
- ✅ Metrics endpoint exposes performance data
- ✅ Graceful shutdown completes without data loss
- ✅ Performance targets are met
- ✅ Error handling and recovery work as expected
- ✅ Integration tests pass
- ✅ Documentation is complete

## Files to Create/Modify

### New Files
- `pkg/services/indexing/indexer.go`
- `pkg/services/indexing/monolithic_indexer.go`
- `pkg/services/indexing/event_processor.go`
- `pkg/services/indexing/event_validator.go`
- `pkg/services/indexing/event_decoder.go`
- `pkg/services/indexing/event_enricher.go`
- `pkg/services/query/query_cache.go`
- `pkg/services/health/health_checker.go`
- `pkg/services/metrics/metrics_exporter.go`
- `pkg/plugins/api/http/rest_handler.go`
- `test/integration/indexer_integration_test.go`
- `test/integration/event_pipeline_test.go`
- `test/integration/query_service_test.go`
- `test/integration/api_gateway_test.go`
- `docs/indexer/README.md`
- `docs/indexer/QUICKSTART.md`
- `docs/indexer/ARCHITECTURE.md`
- `docs/indexer/API.md`
- `docs/indexer/CONFIGURATION.md`
- `docs/indexer/TROUBLESHOOTING.md`

### Modified Files
- `pkg/plugins/pullers/data_puller.go` - Add Kafka publishing
- `pkg/services/query/query_service.go` - Add filter types
- `pkg/plugins/api/business/query_service.go` - Implement cache-first logic
- `pkg/plugins/database/postgres_database.go` - Add filter support
- `pkg/plugins/api/graphql/plugin.go` - Enhance schema
- `pkg/plugins/api/graphql_handler.go` - Add subscriptions
- `pkg/plugins/api/graphql/adapter.go` - Update resolvers
- `pkg/plugins/api/grpc/plugin.go` - gRPC implementation
- `pkg/plugins/api/grpc/adapter.go` - gRPC handlers
- `pkg/plugins/api/proto/chainpulse.proto` - Update schema
- `pkg/plugins/api/http/plugin.go` - HTTP plugin
- `pkg/plugins/api/shared/health.go` - Health endpoint
- `pkg/plugins/api/shared/monitoring.go` - Metrics endpoint

## Next Steps

1. **Review this spec** with the team
2. **Confirm requirements** - Are all requirements clear and acceptable?
3. **Approve design** - Does the architecture make sense?
4. **Begin Phase 1** - Start with Task 1: Create Indexer Service Skeleton
5. **Execute tasks sequentially** - Follow the implementation order
6. **Test continuously** - Run integration tests after each phase
7. **Document as you go** - Keep documentation up-to-date

## Questions to Confirm

Before starting implementation:

1. Are all 10 requirements acceptable?
2. Is the architecture design clear?
3. Are the performance targets achievable?
4. Should we add any additional features?
5. Are there any constraints we should consider?
6. What's the priority order if we need to cut scope?

