# ChainPulse Indexer Monolithic Service - Design

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    ChainPulse Indexer Service                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              API Gateway (Multi-Protocol)                │  │
│  │  ┌──────────┬──────────┬──────────┬──────────────────┐  │  │
│  │  │ GraphQL  │  gRPC    │   HTTP   │   WebSocket      │  │  │
│  │  └──────────┴──────────┴──────────┴──────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                           ▲                                     │
│                           │                                     │
│  ┌────────────────────────┴────────────────────────────────┐   │
│  │           Unified Query Service                         │   │
│  │  ┌──────────────────────────────────────────────────┐  │   │
│  │  │  Cache-First Pattern (Redis → PostgreSQL)       │  │   │
│  │  └──────────────────────────────────────────────────┘  │   │
│  └────────────────────────────────────────────────────────┘   │
│           ▲                                    ▲                │
│           │                                    │                │
│  ┌────────┴──────────┐              ┌─────────┴──────────┐    │
│  │  Redis Cache      │              │  PostgreSQL DB     │    │
│  │  (Hot Data)       │              │  (Persistent)      │    │
│  └───────────────────┘              └────────────────────┘    │
│           ▲                                    ▲                │
│           │                                    │                │
│  ┌────────┴──────────────────────────────────┴──────────┐     │
│  │         Event Processing Pipeline                    │     │
│  │  ┌──────────────────────────────────────────────┐   │     │
│  │  │  Validation → Decoding → Enrichment → Store │   │     │
│  │  └──────────────────────────────────────────────┘   │     │
│  └────────────────────────────────────────────────────┘     │
│           ▲                                                   │
│           │                                                   │
│  ┌────────┴──────────────────────────────────────────────┐   │
│  │         Kafka Message Queue                           │   │
│  │  (Async Event Processing, Dead-Letter Queue)         │   │
│  └────────────────────────────────────────────────────┘   │
│           ▲                                                   │
│           │                                                   │
│  ┌────────┴──────────────────────────────────────────────┐   │
│  │         Data Collection Layer                         │   │
│  │  ┌──────────────┬──────────────┬──────────────────┐  │   │
│  │  │ gRPC Puller  │ HTTPS Puller │ WebSocket Puller │  │   │
│  │  └──────────────┴──────────────┴──────────────────┘  │   │
│  └────────────────────────────────────────────────────┘   │
│           ▲                                                   │
│           │                                                   │
│  ┌────────┴──────────────────────────────────────────────┐   │
│  │         Blockchain Nodes (Multiple Chains)           │   │
│  │  ┌──────────┬──────────┬──────────┬──────────────┐   │   │
│  │  │ Ethereum │ Polygon  │ Arbitrum │ Optimism ... │   │   │
│  │  └──────────┴──────────┴──────────┴──────────────┘   │   │
│  └────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Core Services (Shared)                      │  │
│  │  ┌──────────┬──────────┬──────────┬──────────────────┐  │  │
│  │  │ Logger   │ Metrics  │ Registry │ Event Bus        │  │  │
│  │  └──────────┴──────────┴──────────┴──────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Component Interactions

### 1. Data Collection Flow

```
Blockchain Node
    ↓
Data Puller (gRPC/HTTPS/WebSocket)
    ↓
Event Validation
    ↓
Kafka Message Queue
    ↓
Event Processing Workers
```

### 2. Event Processing Flow

```
Raw Event (from Kafka)
    ↓
Schema Validation
    ↓
ABI Decoding
    ↓
Metadata Enrichment
    ↓
PostgreSQL Write
    ↓
Redis Cache Update
    ↓
Metrics Update
```

### 3. Query Flow

```
API Request (GraphQL/gRPC/HTTP/WebSocket)
    ↓
Query Service
    ↓
Redis Cache Check
    ├─ HIT → Return Cached Result
    └─ MISS → Query PostgreSQL
              ↓
              Cache Result
              ↓
              Return Result
```

## Component Specifications

### 1. Data Puller Component

**Location**: `pkg/plugins/pullers/`

**Responsibilities**:
- Connect to blockchain nodes via gRPC/HTTPS/WebSocket
- Collect events from specified block ranges
- Handle chain reorganizations (reorgs)
- Retry failed connections with exponential backoff
- Tag events with chain ID and network name

**Key Classes**:
- `MultiChainDataPuller`: Orchestrates pullers for multiple chains
- `GRPCPuller`: gRPC protocol implementation
- `HTTPSJSONRPCPuller`: HTTPS JSON-RPC implementation
- `WebSocketJSONRPCPuller`: WebSocket JSON-RPC implementation

**Configuration**:
```go
type PullerConfig struct {
    ChainID          string
    NodeURL          string
    Protocol         string // "grpc", "https-jsonrpc", "websocket-jsonrpc"
    StartBlock       uint64
    BatchSize        uint64
    RetryMaxAttempts int
    RetryBackoff     time.Duration
}
```

### 2. Message Queue Component

**Location**: `pkg/plugins/mq/`

**Responsibilities**:
- Publish collected events to Kafka topics
- Consume events for processing
- Handle acknowledgments and retries
- Manage dead-letter queue for failed events
- Track offset per topic and consumer group

**Key Classes**:
- `KafkaMQPlugin`: Kafka implementation
- `MessageBroker`: Interface for message queue operations

**Topics**:
- `chainpulse-events`: Main event topic
- `chainpulse-events-dlq`: Dead-letter queue for failed events

**Configuration**:
```go
type MQConfig struct {
    BrokerURL        string
    Topic            string
    ConsumerGroup    string
    BatchSize        int
    RetryMaxAttempts int
    RetryBackoff     time.Duration
}
```

### 3. Event Processing Pipeline

**Location**: `pkg/services/indexing/`

**Responsibilities**:
- Consume events from Kafka
- Validate event schema
- Decode events using contract ABI
- Enrich events with metadata
- Persist to database
- Update cache

**Key Classes**:
- `EventProcessor`: Main processing orchestrator
- `EventValidator`: Schema validation
- `EventDecoder`: ABI decoding
- `EventEnricher`: Metadata enrichment

**Processing Steps**:
1. Consume from Kafka
2. Validate against schema
3. Decode using ABI
4. Enrich with token/pool metadata
5. Write to PostgreSQL
6. Update Redis cache
7. Publish metrics

### 4. Query Service Component

**Location**: `pkg/services/query/`

**Responsibilities**:
- Provide unified query interface
- Implement cache-first pattern
- Support filtering and pagination
- Return consistent results across protocols

**Key Classes**:
- `QueryService`: Interface definition
- `DefaultQueryService`: Implementation with cache-first pattern
- `QueryCache`: Redis-backed cache layer
- `QueryExecutor`: Database query execution

**Query Types**:
- `GetEvent(id)`: Get single event by ID
- `QueryEvents(filter, limit, offset)`: Query with filtering
- `GetTokenMetadata(token)`: Get token info
- `GetPoolMetadata(pool)`: Get pool info
- `GetContractMetadata(address)`: Get contract info

### 5. API Gateway Component

**Location**: `pkg/plugins/api/`

**Responsibilities**:
- Route requests to appropriate protocol handler
- Expose multi-protocol endpoints
- Implement health checks
- Collect metrics

**Protocol Handlers**:
- `GraphQL Handler`: GraphQL queries and subscriptions
- `gRPC Handler`: Protocol Buffer RPC
- `HTTP Handler`: REST API
- `WebSocket Handler`: Real-time subscriptions

**Endpoints**:
- `GET /health`: Health check
- `GET /metrics`: Prometheus metrics
- `POST /graphql`: GraphQL queries
- `WS /graphql`: GraphQL subscriptions
- `POST /api/v1/events`: HTTP REST API
- `:50051`: gRPC endpoint

### 6. Cache Component

**Location**: `pkg/plugins/cache/`

**Responsibilities**:
- Store frequently accessed data
- Implement LRU eviction policy
- Track cache statistics
- Support TTL-based expiration

**Key Classes**:
- `CachePlugin`: Interface
- `RedisCachePlugin`: Redis implementation
- `InMemoryCachePlugin`: In-memory fallback

**Cache Keys**:
- `event:{id}`: Single event
- `events:{filter_hash}`: Query results
- `token:{address}`: Token metadata
- `pool:{address}`: Pool metadata

### 7. Database Component

**Location**: `pkg/plugins/database/`

**Responsibilities**:
- Persist events durably
- Create and maintain indexes
- Support transactions
- Handle connection pooling

**Key Classes**:
- `DatabasePlugin`: Interface
- `PostgreSQLDatabase`: PostgreSQL implementation

**Tables**:
- `events`: Main event table with indexes on contract_address, event_name, block_number, timestamp
- `tokens`: Token metadata cache
- `pools`: Pool metadata cache
- `contracts`: Contract metadata cache

**Indexes**:
- `idx_events_contract_address`: For contract filtering
- `idx_events_event_name`: For event name filtering
- `idx_events_block_number`: For block range queries
- `idx_events_timestamp`: For time-based queries

## Initialization Sequence

```
1. Load Configuration
   ↓
2. Initialize Logger
   ↓
3. Initialize Metrics Collector
   ↓
4. Initialize Plugin Registry
   ↓
5. Initialize Cache Plugin
   ↓
6. Initialize Database Plugin
   ↓
7. Initialize Message Queue Plugin
   ↓
8. Initialize Data Pullers (per chain)
   ↓
9. Initialize Event Processor
   ↓
10. Initialize Query Service
    ↓
11. Initialize API Gateway
    ↓
12. Start Data Pullers
    ↓
13. Start Event Processor
    ↓
14. Start API Gateway
    ↓
15. Ready to serve requests
```

## Graceful Shutdown Sequence

```
1. Receive SIGTERM/SIGINT
   ↓
2. Stop accepting new requests
   ↓
3. Wait for in-flight requests (timeout: 30s)
   ↓
4. Stop API Gateway
   ↓
5. Stop Event Processor
   ↓
6. Stop Data Pullers
   ↓
7. Flush Message Queue
   ↓
8. Close Database Connections
   ↓
9. Close Cache Connections
   ↓
10. Exit with status 0
```

## Data Models

### Event Model

```go
type Event struct {
    ID               string
    EventHash        string
    EventSignature   string
    BlockNumber      uint64
    BlockHash        string
    BlockTimestamp   uint64
    TransactionHash  string
    TransactionIndex uint64
    LogIndex         uint64
    ContractAddress  string
    EventName        string
    EventTopics      []string
    EventData        []byte
    DecodedData      map[string]interface{}
    ChainID          string
    Network          string
    Status           string
    CreatedAt        time.Time
    ProcessedAt      time.Time
    IndexedAt        time.Time
}
```

### Query Filter Model

```go
type EventFilter struct {
    ContractAddress []string
    EventName       string
    FromBlock       *big.Int
    ToBlock         *big.Int
    Network         string
    FromTimestamp   time.Time
    ToTimestamp     time.Time
}
```

## Performance Considerations

### Cache Strategy
- **Cache-First Pattern**: Check Redis before PostgreSQL
- **TTL**: 1 hour for most queries, 24 hours for metadata
- **Eviction**: LRU policy when cache is full
- **Hit Rate Target**: 70-85%

### Database Optimization
- **Connection Pooling**: 25 max connections, 12 idle
- **Query Timeout**: 30 seconds default
- **Batch Writes**: 100 events per batch
- **Indexes**: On contract_address, event_name, block_number, timestamp

### Message Queue Optimization
- **Batch Size**: 100 events per batch
- **Worker Pool**: 8 workers by default (configurable)
- **Retry Policy**: Exponential backoff, max 5 retries
- **Dead-Letter Queue**: For failed events

### API Performance
- **Request Timeout**: 30 seconds
- **Response Compression**: gzip for HTTP
- **Connection Pooling**: For upstream services
- **Rate Limiting**: Per IP address (optional)

## Error Handling Strategy

### Retry Policy
- **Data Puller**: Exponential backoff, max 5 retries
- **Database**: Exponential backoff, max 3 retries
- **Message Queue**: Exponential backoff, max 5 retries
- **API Requests**: Exponential backoff, max 3 retries

### Dead-Letter Queue
- Failed events moved to DLQ after max retries
- Manual review and reprocessing capability
- Metrics tracking for DLQ size

### Graceful Degradation
- If cache unavailable: Query database directly
- If database unavailable: Queue events in Kafka
- If message queue unavailable: Buffer in memory (limited)
- If data puller fails: Retry and continue with other chains

## Monitoring and Observability

### Metrics
- `events_collected_total`: Total events collected per chain
- `events_processed_total`: Total events processed
- `events_cached_total`: Total events cached
- `query_latency_ms`: Query execution latency
- `cache_hit_rate`: Cache hit percentage
- `error_rate`: Error rate per component
- `uptime_seconds`: Service uptime

### Logging
- **Level**: DEBUG, INFO, WARN, ERROR
- **Format**: JSON with timestamp, level, component, message
- **Destinations**: stdout, file (optional)

### Health Checks
- `/health`: Returns status of all components
- Component status: healthy, degraded, unhealthy
- Response time: <100ms

## Deployment Considerations

### Docker
- Single container with all components
- Environment variables for configuration
- Health check endpoint for orchestration

### Kubernetes
- Single pod with all containers
- ConfigMap for configuration
- Service for API exposure
- PersistentVolume for data (if needed)

### Scaling
- Horizontal: Multiple instances with shared database/cache
- Vertical: Increase worker pool size, batch size
- Database: Connection pooling, read replicas

