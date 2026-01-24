# ChainPulse Indexer Monolithic Service - Implementation Tasks

## Phase 1: Core Indexer Service (Tasks 1-3)

### Task 1: Create Indexer Service Skeleton

**Objective**: Create the main indexer service that orchestrates all components.

**Acceptance Criteria**:
- ✅ `pkg/services/indexing/indexer.go` created with `Indexer` interface
- ✅ `pkg/services/indexing/monolithic_indexer.go` created with `MonolithicIndexer` implementation
- ✅ Indexer initializes all plugins in correct order
- ✅ Indexer implements Start/Stop lifecycle methods
- ✅ Indexer tracks initialization state and prevents double-initialization

**Files to Create**:
- `pkg/services/indexing/indexer.go` - Interface definition
- `pkg/services/indexing/monolithic_indexer.go` - Implementation

**Dependencies**:
- `pkg/core/logger.go`
- `pkg/core/metrics.go`
- `pkg/core/registry.go`
- `pkg/plugins/cache/cache_plugin.go`
- `pkg/plugins/database/database_plugin.go`
- `pkg/plugins/mq/kafka_mq.go`
- `pkg/plugins/pullers/data_puller.go`
- `pkg/services/query/query_service.go`
- `pkg/plugins/api/gateway.go`

**Estimated Effort**: 2 hours

---

### Task 2: Create Event Processing Pipeline

**Objective**: Implement the event processing pipeline that validates, decodes, enriches, and persists events.

**Acceptance Criteria**:
- ✅ `pkg/services/indexing/event_processor.go` created
- ✅ Event processor consumes from Kafka
- ✅ Event processor validates events against schema
- ✅ Event processor decodes events using ABI
- ✅ Event processor enriches events with metadata
- ✅ Event processor persists to database
- ✅ Event processor updates cache
- ✅ Event processor handles errors and moves failed events to DLQ
- ✅ Event processor tracks metrics

**Files to Create**:
- `pkg/services/indexing/event_processor.go` - Main processor
- `pkg/services/indexing/event_validator.go` - Schema validation
- `pkg/services/indexing/event_decoder.go` - ABI decoding
- `pkg/services/indexing/event_enricher.go` - Metadata enrichment

**Dependencies**:
- `pkg/plugins/mq/kafka_mq.go`
- `pkg/plugins/database/database_plugin.go`
- `pkg/plugins/cache/cache_plugin.go`
- `pkg/services/decoder/event_decoder.go`
- `pkg/core/logger.go`
- `pkg/core/metrics.go`

**Estimated Effort**: 4 hours

---

### Task 3: Integrate Data Pullers with Event Pipeline

**Objective**: Connect data pullers to the event processing pipeline.

**Acceptance Criteria**:
- ✅ Data pullers publish events to Kafka
- ✅ Event processor consumes from Kafka
- ✅ Events flow from puller → Kafka → processor → database → cache
- ✅ Chain ID and network name are preserved through pipeline
- ✅ Error handling works end-to-end
- ✅ Metrics are collected at each stage

**Files to Modify**:
- `pkg/plugins/pullers/data_puller.go` - Add Kafka publishing
- `pkg/services/indexing/monolithic_indexer.go` - Wire components together

**Dependencies**:
- `pkg/plugins/pullers/multi_chain_puller.go`
- `pkg/plugins/mq/kafka_mq.go`
- `pkg/services/indexing/event_processor.go`

**Estimated Effort**: 2 hours

---

## Phase 2: Query Service Enhancement (Tasks 4-5)

### Task 4: Implement Cache-First Query Pattern

**Objective**: Enhance query service to implement cache-first pattern with automatic fallback to database.

**Acceptance Criteria**:
- ✅ Query service checks Redis cache first
- ✅ Cache miss triggers database query
- ✅ Database results are cached with appropriate TTL
- ✅ Cache hit rate is tracked in metrics
- ✅ Query latency is <100ms for cache hits, <500ms for database hits
- ✅ Cache invalidation works correctly

**Files to Create/Modify**:
- `pkg/services/query/query_cache.go` - Cache layer (new)
- `pkg/services/query/query_service.go` - Add cache-first logic (modify)
- `pkg/plugins/api/business/query_service.go` - Update implementation (modify)

**Dependencies**:
- `pkg/plugins/cache/cache_plugin.go`
- `pkg/plugins/database/database_plugin.go`
- `pkg/core/logger.go`
- `pkg/core/metrics.go`

**Estimated Effort**: 3 hours

---

### Task 5: Add Query Filtering and Pagination

**Objective**: Implement comprehensive query filtering and pagination support.

**Acceptance Criteria**:
- ✅ Query service supports filtering by: contract address, event name, block range, chain ID, timestamp range
- ✅ Query service supports pagination with limit and offset
- ✅ Query service returns total count for pagination
- ✅ Filters are applied efficiently (use database indexes)
- ✅ Pagination works with cache and database

**Files to Modify**:
- `pkg/services/query/query_service.go` - Add filter types (modify)
- `pkg/plugins/api/business/query_service.go` - Implement filtering (modify)
- `pkg/plugins/database/postgres_database.go` - Add filter support (modify)

**Dependencies**:
- `pkg/plugins/database/database_plugin.go`
- `pkg/core/logger.go`

**Estimated Effort**: 2 hours

---

## Phase 3: API Gateway Integration (Tasks 6-8)

### Task 6: Enhance GraphQL Handler

**Objective**: Enhance GraphQL handler to support all query types and subscriptions.

**Acceptance Criteria**:
- ✅ GraphQL schema includes all query types
- ✅ GraphQL schema includes subscription types
- ✅ GraphQL resolvers use query service
- ✅ GraphQL subscriptions work via WebSocket
- ✅ GraphQL Playground is available
- ✅ Query results are returned in GraphQL format

**Files to Modify**:
- `pkg/plugins/api/graphql/plugin.go` - Enhance schema (modify)
- `pkg/plugins/api/graphql_handler.go` - Add subscriptions (modify)
- `pkg/plugins/api/graphql/adapter.go` - Update resolvers (modify)

**Dependencies**:
- `pkg/services/query/query_service.go`
- `pkg/plugins/api/websocket_handler.go`
- `pkg/core/logger.go`

**Estimated Effort**: 3 hours

---

### Task 7: Implement gRPC Handler

**Objective**: Implement gRPC handler for Protocol Buffer RPC support.

**Acceptance Criteria**:
- ✅ gRPC service definition includes all query types
- ✅ gRPC server listens on port 50051
- ✅ gRPC handlers use query service
- ✅ gRPC streaming is supported
- ✅ Query results are returned in Protocol Buffer format

**Files to Create/Modify**:
- `pkg/plugins/api/grpc/plugin.go` - gRPC plugin (modify)
- `pkg/plugins/api/grpc/adapter.go` - gRPC handlers (modify)
- `pkg/plugins/api/proto/chainpulse.proto` - Update schema (modify)

**Dependencies**:
- `pkg/services/query/query_service.go`
- `pkg/core/logger.go`
- `google.golang.org/grpc`

**Estimated Effort**: 3 hours

---

### Task 8: Implement HTTP REST Handler

**Objective**: Implement HTTP REST handler for standard REST API support.

**Acceptance Criteria**:
- ✅ HTTP endpoints: GET /api/v1/events, GET /api/v1/events/{id}, GET /api/v1/tokens/{address}
- ✅ HTTP handlers use query service
- ✅ Query results are returned in JSON format
- ✅ Pagination works via query parameters
- ✅ Error responses follow standard format

**Files to Create/Modify**:
- `pkg/plugins/api/http/plugin.go` - HTTP plugin (modify)
- `pkg/plugins/api/http/rest_handler.go` - REST handlers (new)

**Dependencies**:
- `pkg/services/query/query_service.go`
- `pkg/core/logger.go`

**Estimated Effort**: 2 hours

---

## Phase 4: Monitoring and Observability (Tasks 9-10)

### Task 9: Implement Health Check Endpoint

**Objective**: Create health check endpoint that reports status of all components.

**Acceptance Criteria**:
- ✅ Health check endpoint at GET /health
- ✅ Returns status of: cache, database, message queue, data pullers, query service, API gateway
- ✅ Each component status: healthy, degraded, unhealthy
- ✅ Response time: <100ms
- ✅ Response format: JSON with component details

**Files to Create/Modify**:
- `pkg/services/health/health_checker.go` - Health check logic (new)
- `pkg/plugins/api/shared/health.go` - Health endpoint (modify)

**Dependencies**:
- `pkg/plugins/cache/cache_plugin.go`
- `pkg/plugins/database/database_plugin.go`
- `pkg/plugins/mq/kafka_mq.go`
- `pkg/plugins/pullers/data_puller.go`
- `pkg/core/logger.go`

**Estimated Effort**: 2 hours

---

### Task 10: Implement Metrics Endpoint

**Objective**: Create metrics endpoint that exposes Prometheus metrics.

**Acceptance Criteria**:
- ✅ Metrics endpoint at GET /metrics
- ✅ Metrics include: events_collected_total, events_processed_total, events_cached_total, query_latency_ms, cache_hit_rate, error_rate, uptime_seconds
- ✅ Metrics are in Prometheus format
- ✅ Metrics are updated in real-time
- ✅ Response time: <100ms

**Files to Create/Modify**:
- `pkg/services/metrics/metrics_exporter.go` - Metrics export logic (new)
- `pkg/plugins/api/shared/monitoring.go` - Metrics endpoint (modify)

**Dependencies**:
- `pkg/core/metrics.go`
- `prometheus/client_golang`

**Estimated Effort**: 2 hours

---

## Phase 5: Testing and Documentation (Tasks 11-12)

### Task 11: Create Integration Tests

**Objective**: Create comprehensive integration tests for the entire system.

**Acceptance Criteria**:
- ✅ Test: Data collection from multiple chains
- ✅ Test: Event processing pipeline end-to-end
- ✅ Test: Query service with cache-first pattern
- ✅ Test: All API protocols (GraphQL, gRPC, HTTP, WebSocket)
- ✅ Test: Error handling and recovery
- ✅ Test: Graceful shutdown
- ✅ All tests pass

**Files to Create**:
- `test/integration/indexer_integration_test.go` - Main integration tests
- `test/integration/event_pipeline_test.go` - Event pipeline tests
- `test/integration/query_service_test.go` - Query service tests
- `test/integration/api_gateway_test.go` - API gateway tests

**Dependencies**:
- All service components
- `testing` package
- Docker containers for dependencies (Kafka, PostgreSQL, Redis)

**Estimated Effort**: 4 hours

---

### Task 12: Create Documentation and Quick Start Guide

**Objective**: Create comprehensive documentation and quick start guide.

**Acceptance Criteria**:
- ✅ README.md with system overview
- ✅ QUICKSTART.md with setup instructions
- ✅ ARCHITECTURE.md with detailed architecture
- ✅ API.md with endpoint documentation
- ✅ CONFIGURATION.md with environment variables
- ✅ TROUBLESHOOTING.md with common issues
- ✅ All documentation is clear and complete

**Files to Create**:
- `docs/indexer/README.md` - System overview
- `docs/indexer/QUICKSTART.md` - Quick start guide
- `docs/indexer/ARCHITECTURE.md` - Architecture details
- `docs/indexer/API.md` - API documentation
- `docs/indexer/CONFIGURATION.md` - Configuration guide
- `docs/indexer/TROUBLESHOOTING.md` - Troubleshooting guide

**Estimated Effort**: 3 hours

---

## Implementation Order

**Recommended execution order** (respects dependencies):

1. **Task 1**: Create Indexer Service Skeleton (2h)
2. **Task 2**: Create Event Processing Pipeline (4h)
3. **Task 3**: Integrate Data Pullers with Event Pipeline (2h)
4. **Task 4**: Implement Cache-First Query Pattern (3h)
5. **Task 5**: Add Query Filtering and Pagination (2h)
6. **Task 6**: Enhance GraphQL Handler (3h)
7. **Task 7**: Implement gRPC Handler (3h)
8. **Task 8**: Implement HTTP REST Handler (2h)
9. **Task 9**: Implement Health Check Endpoint (2h)
10. **Task 10**: Implement Metrics Endpoint (2h)
11. **Task 11**: Create Integration Tests (4h)
12. **Task 12**: Create Documentation (3h)

**Total Estimated Effort**: 32 hours

**Phases**:
- **Phase 1** (Tasks 1-3): 8 hours - Core indexer service
- **Phase 2** (Tasks 4-5): 5 hours - Query service enhancement
- **Phase 3** (Tasks 6-8): 8 hours - API gateway integration
- **Phase 4** (Tasks 9-10): 4 hours - Monitoring and observability
- **Phase 5** (Tasks 11-12): 7 hours - Testing and documentation

## Success Criteria

After completing all tasks:

- ✅ Service starts successfully with all components initialized
- ✅ Events are collected from multiple chains in parallel
- ✅ Events flow through complete pipeline: collect → validate → queue → process → persist → cache
- ✅ Query service returns results from cache (70-85% hit rate)
- ✅ All 4 API protocols (GraphQL, gRPC, HTTP, WebSocket) work correctly
- ✅ Health check endpoint reports status of all components
- ✅ Metrics endpoint exposes performance data
- ✅ Graceful shutdown completes without data loss
- ✅ Performance targets are met (1000+ events/sec, <1ms latency)
- ✅ Error handling and recovery work as expected
- ✅ Integration tests pass
- ✅ Documentation is complete and clear

