# ChainPulse Indexer Monolithic Service - Requirements

## Introduction

Create an integrated monolithic indexer service that combines all verified plugins (Cache, Database, Message Queue, Data Puller, API Gateway) into a single cohesive system. The service will collect blockchain events from multiple chains, process them through a unified pipeline, persist to PostgreSQL with Redis caching, and expose a multi-protocol query interface (GraphQL, gRPC, HTTP, WebSocket).

## Glossary

- **Data Puller**: Component that collects blockchain events from multiple chains via gRPC/HTTPS/WebSocket
- **Event Pipeline**: Processing flow from collection → validation → storage → caching
- **Message Queue**: Kafka-based event broker for asynchronous processing
- **Query Service**: Unified interface for querying indexed events across all chains
- **API Gateway**: Multi-protocol endpoint (GraphQL, gRPC, HTTP, WebSocket)
- **Monolithic Service**: Single deployable unit with all components integrated
- **Chain**: Individual blockchain network (Ethereum, Polygon, Arbitrum, etc.)

## Requirements

### Requirement 1: Multi-Chain Data Collection

**User Story:** As an indexer operator, I want to collect blockchain events from multiple chains simultaneously, so that I can index events across the entire Web3 ecosystem.

#### Acceptance Criteria

1. WHEN the indexer starts, IT SHALL initialize data pullers for all configured chains
2. WHEN a data puller connects to a blockchain node, IT SHALL support gRPC, HTTPS JSON-RPC, and WebSocket JSON-RPC protocols
3. WHEN events are collected from a chain, THEY SHALL be tagged with the chain ID and network name
4. WHEN a blockchain node becomes unavailable, THE data puller SHALL retry with exponential backoff
5. WHEN multiple chains are configured, THEY SHALL be indexed in parallel without blocking each other
6. WHEN an event is collected, IT SHALL include: block number, transaction hash, log index, contract address, event signature, topics, and data

### Requirement 2: Event Processing Pipeline

**User Story:** As a system architect, I want events to flow through a validated processing pipeline, so that only valid events are persisted and indexed.

#### Acceptance Criteria

1. WHEN an event is collected, IT SHALL be validated against the event schema
2. WHEN an event is validated, IT SHALL be published to the Kafka message queue
3. WHEN an event is in the message queue, IT SHALL be processed asynchronously by workers
4. WHEN an event is processed, IT SHALL be decoded using the contract ABI
5. WHEN an event is decoded, IT SHALL be enriched with metadata (token info, pool info, etc.)
6. WHEN an event is enriched, IT SHALL be persisted to PostgreSQL
7. WHEN an event is persisted, IT SHALL be cached in Redis for fast retrieval
8. WHEN processing fails, THE event SHALL be moved to a dead-letter queue for manual review

### Requirement 3: Unified Query Service

**User Story:** As an API consumer, I want a single query interface that works across all indexed chains, so that I can query events uniformly regardless of source chain.

#### Acceptance Criteria

1. WHEN a query is executed, IT SHALL support filtering by: contract address, event name, block range, chain ID, timestamp range
2. WHEN a query is executed, IT SHALL check Redis cache first (cache-first pattern)
3. WHEN cache misses, THE query SHALL fall back to PostgreSQL
4. WHEN results are retrieved from PostgreSQL, THEY SHALL be cached in Redis with appropriate TTL
5. WHEN a query returns results, THEY SHALL be paginated with limit and offset
6. WHEN a query is executed, IT SHALL return results in <100ms for cached queries and <500ms for database queries
7. WHEN multiple queries are executed, THEY SHALL be batched for efficiency

### Requirement 4: Multi-Protocol API Gateway

**User Story:** As an API consumer, I want to query indexed events using my preferred protocol (GraphQL, gRPC, HTTP, WebSocket), so that I can integrate with my existing infrastructure.

#### Acceptance Criteria

1. WHEN the indexer starts, IT SHALL expose GraphQL endpoint at `/graphql`
2. WHEN the indexer starts, IT SHALL expose gRPC endpoint at `:50051`
3. WHEN the indexer starts, IT SHALL expose HTTP REST endpoint at `/api/v1`
4. WHEN the indexer starts, IT SHALL expose WebSocket endpoint at `/ws`
5. WHEN a client connects via WebSocket, IT SHALL support real-time event subscriptions
6. WHEN a GraphQL query is executed, IT SHALL return results in GraphQL format
7. WHEN a gRPC request is made, IT SHALL return results in Protocol Buffer format
8. WHEN an HTTP request is made, IT SHALL return results in JSON format
9. ALL protocols SHALL use the same underlying query service (zero code duplication)

### Requirement 5: Data Persistence and Caching

**User Story:** As a system operator, I want reliable data persistence with fast caching, so that indexed events are durable and queries are performant.

#### Acceptance Criteria

1. WHEN an event is persisted, IT SHALL be written to PostgreSQL with ACID guarantees
2. WHEN an event is persisted, IT SHALL be indexed by: contract address, event name, block number, timestamp
3. WHEN an event is persisted, IT SHALL be cached in Redis with 1-hour TTL by default
4. WHEN Redis cache is full, OLD entries SHALL be evicted using LRU policy
5. WHEN PostgreSQL is unavailable, THE system SHALL queue events in Kafka until database is available
6. WHEN cache hit rate is tracked, IT SHALL be reported in metrics (target: 70-85%)
7. WHEN database queries are slow, THEY SHALL be logged for optimization

### Requirement 6: System Initialization and Lifecycle

**User Story:** As an operator, I want the system to initialize all components in correct order and shut down gracefully, so that the service is reliable and data-safe.

#### Acceptance Criteria

1. WHEN the service starts, IT SHALL initialize in this order: Logger → Metrics → Registry → Cache → Database → Message Queue → Data Pullers → Query Service → API Gateway
2. WHEN initialization fails at any step, THE service SHALL log the error and exit with non-zero status
3. WHEN the service receives SIGTERM or SIGINT, IT SHALL initiate graceful shutdown
4. DURING graceful shutdown, IT SHALL: stop accepting new requests → wait for in-flight requests → flush message queue → close database connections → stop data pullers
5. WHEN graceful shutdown completes, THE service SHALL exit with status 0
6. WHEN the service is running, IT SHALL expose health check endpoint at `/health`
7. WHEN health check is called, IT SHALL return status of all components

### Requirement 7: Monitoring and Observability

**User Story:** As an operator, I want comprehensive monitoring of the indexer, so that I can detect issues and optimize performance.

#### Acceptance Criteria

1. WHEN the service is running, IT SHALL collect metrics: events collected, events processed, events cached, query latency, cache hit rate, error rate
2. WHEN metrics are collected, THEY SHALL be exposed at `/metrics` in Prometheus format
3. WHEN an error occurs, IT SHALL be logged with: timestamp, error message, stack trace, component name
4. WHEN a query is executed, IT SHALL be logged with: query type, execution time, result count, cache hit/miss
5. WHEN data puller connects to a chain, IT SHALL log: chain ID, node URL, connection status
6. WHEN the service is running, IT SHALL track uptime and report in health check

### Requirement 8: Configuration Management

**User Story:** As an operator, I want to configure the indexer via environment variables, so that I can deploy to different environments without code changes.

#### Acceptance Criteria

1. WHEN the service starts, IT SHALL read configuration from environment variables
2. WHEN configuration is missing, IT SHALL use sensible defaults
3. WHEN configuration is invalid, IT SHALL log error and exit
4. THE following configuration options SHALL be supported:
   - `CHAINS`: Comma-separated list of chains to index (e.g., "ethereum,polygon,arbitrum")
   - `BLOCKCHAIN_NODE_URLS`: Comma-separated list of node URLs (one per chain)
   - `DATA_PULLER_TYPE`: Type of data puller (grpc, https-jsonrpc, websocket-jsonrpc)
   - `MQ_TYPE`: Message queue type (kafka)
   - `MQ_CONNECTION_URL`: Message queue connection string
   - `CACHE_TYPE`: Cache type (redis, in-memory)
   - `CACHE_CONNECTION_URL`: Cache connection string
   - `DATABASE_TYPE`: Database type (postgres)
   - `DATABASE_URL`: Database connection string
   - `API_PORT`: API gateway port (default: 8080)
   - `WORKER_POOL_SIZE`: Number of event processing workers (default: 8)
   - `BATCH_SIZE`: Event batch size for processing (default: 100)
   - `LOG_LEVEL`: Logging level (debug, info, warn, error)

### Requirement 9: Error Handling and Resilience

**User Story:** As a system operator, I want the indexer to handle errors gracefully and recover automatically, so that the service is resilient to failures.

#### Acceptance Criteria

1. WHEN a data puller fails to connect, IT SHALL retry with exponential backoff (max 5 retries)
2. WHEN a database query fails, IT SHALL retry up to 3 times with exponential backoff
3. WHEN a message queue operation fails, IT SHALL retry with exponential backoff
4. WHEN an event processing fails, IT SHALL be moved to dead-letter queue for manual review
5. WHEN a component fails, THE system SHALL continue operating with remaining components
6. WHEN a component recovers, IT SHALL automatically reconnect and resume processing
7. WHEN an error occurs, IT SHALL be logged with full context for debugging

### Requirement 10: Performance Targets

**User Story:** As a performance engineer, I want the indexer to meet specific performance targets, so that it can handle production workloads.

#### Acceptance Criteria

1. WHEN events are collected, THE throughput SHALL be at least 1,000 events/sec per chain
2. WHEN events are processed, THE latency SHALL be <1ms per event
3. WHEN events are persisted, THE write latency SHALL be 1-5ms per event
4. WHEN events are queried, THE read latency SHALL be 0.5-2ms per event
5. WHEN cache is hit, THE query latency SHALL be <10ms
6. WHEN cache is missed, THE query latency SHALL be <500ms
7. WHEN message queue is used, THE throughput SHALL be at least 10,000 messages/sec
8. WHEN the system is running, THE cache hit rate SHALL be 70-85%

## Architecture Principles

1. **Integration**: All plugins work together seamlessly in a single service
2. **Unified Pipeline**: Events flow through a consistent processing pipeline
3. **Multi-Chain Support**: Simultaneous indexing of multiple blockchains
4. **Protocol Agnostic**: Query service is independent of API protocols
5. **Cache-First**: Redis cache is checked before database for performance
6. **Graceful Degradation**: System continues operating even if components fail
7. **Observable**: Comprehensive logging and metrics for monitoring
8. **Configurable**: All behavior controlled via environment variables

## Success Criteria

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

