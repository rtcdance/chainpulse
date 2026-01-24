# Design Document: ChainPulse Distributed Architecture

**Date**: January 11, 2026  
**Status**: Draft  
**Version**: 1.0

## Overview

This design document specifies the transformation of ChainPulse from a monolithic architecture to an enterprise-grade distributed microservices architecture. The system will support both monolithic and microkernel deployment modes, enabling horizontal scalability, fault tolerance, and independent service management across multiple blockchain networks.

### Key Design Principles

1. **Microkernel Architecture**: Core system with pluggable services
2. **Stateless Services**: All services store state externally
3. **Event-Driven Communication**: Asynchronous messaging between services
4. **Idempotent Operations**: All operations can be safely retried
5. **Distributed Resilience**: Automatic failover and recovery
6. **Observable Systems**: Comprehensive logging, metrics, and tracing
7. **Configuration-Driven**: Centralized configuration management

## Architecture Overview

### Deployment Modes

#### Monolithic Mode
- All components run in a single process
- Suitable for development and small deployments
- Simplified operational management
- Shared resource pools

#### Microkernel Mode
- Components run as independent services
- Each service can be scaled independently
- Services communicate via message queues
- Distributed configuration management

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    API Cluster (1+ instances)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ REST API     │  │ gRPC API     │  │ WebSocket    │          │
│  │ Instance 1   │  │ Instance 1   │  │ Instance 1   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ REST API     │  │ gRPC API     │  │ WebSocket    │          │
│  │ Instance 2   │  │ Instance 2   │  │ Instance 2   │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
└────────────────────────┬─────────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼──────────┐ ┌──▼──────────┐ ┌──▼──────────┐
│ Data Puller      │ │ Event       │ │ Config      │
│ Cluster          │ │ Processor   │ │ Center      │
│ (EVM, Cosmos,    │ │ Cluster     │ │             │
│  Solana)         │ │ (1+ inst)   │ │ • Consul    │
│ (1+ inst/chain)  │ │             │ │ • etcd      │
└────────┬─────────┘ └──┬──────────┘ │ • Zookeeper│
         │              │            └─────────────┘
         │              │
         └──────┬───────┘
                │
        ┌───────▼──────────────┐
        │  Message Queue       │
        │  Cluster             │
        │  • Kafka             │
        │  • Redis             │
        │  • RabbitMQ          │
        └───────┬──────────────┘
                │
        ┌───────▼──────────────┐
        │  Cache Cluster       │
        │  • Redis             │
        │  • Memcached         │
        └───────┬──────────────┘
                │
        ┌───────▼──────────────┐
        │  Database Cluster    │
        │  • PostgreSQL        │
        │  • MongoDB           │
        └──────────────────────┘
```

## Components and Interfaces

### 1. API Gateway Cluster

**Responsibility**: Serve indexed blockchain data to clients

**Deployment**: 1+ instances behind load balancer

**Features**:
- Multi-protocol support (REST, gRPC, WebSocket)
- Cache-first query strategy
- Session management via distributed cache
- Rate limiting and authentication
- Request/response compression
- Connection pooling

**Interface**:
```go
type APIGateway interface {
    // Query operations
    GetEvents(ctx context.Context, filter EventFilter) ([]Event, error)
    GetEventByID(ctx context.Context, id string) (Event, error)
    QueryByContract(ctx context.Context, contract string) ([]Event, error)
    
    // Health and status
    Health() HealthStatus
    GetMetrics() APIMetrics
}
```

**Scaling Strategy**:
- Horizontal scaling: Add instances behind load balancer
- Session affinity: Optional for WebSocket connections
- Cache locality: Replicate cache across instances

### 2. Data Puller Cluster

**Responsibility**: Extract blockchain events from multiple chains

**Deployment**: 1+ instances per blockchain network

**Features**:
- Multi-protocol support (HTTPS-JSONRPC, WebSocket, gRPC)
- Block height tracking per chain
- Reorg detection and handling
- Connection pooling and retry logic
- Event batching for efficiency

**Interface**:
```go
type DataPuller interface {
    // Event pulling
    PullEvents(ctx context.Context, fromBlock, toBlock uint64) ([]Event, error)
    SubscribeToEvents(ctx context.Context, handler func(Event)) error
    
    // State management
    GetLatestBlock(ctx context.Context) (uint64, error)
    GetProcessedHeight(ctx context.Context) (uint64, error)
    
    // Health and status
    Health() HealthStatus
    GetMetrics() PullerMetrics
}
```

**Scaling Strategy**:
- Per-chain instances: Separate instances for each blockchain
- Partition by block range: Multiple instances per chain
- Load balancing: Distribute chains across instances

### 3. Event Processor Cluster

**Responsibility**: Process, validate, and store blockchain events

**Deployment**: 1+ instances

**Features**:
- Event validation and normalization
- Idempotency checking via event hashing
- Batch processing for efficiency
- Transaction management
- Dead letter queue for failed events
- Reorg handling

**Interface**:
```go
type EventProcessor interface {
    // Event processing
    ProcessEvent(ctx context.Context, event Event) error
    ProcessBatch(ctx context.Context, events []Event) error
    
    // Idempotency
    IsDuplicate(ctx context.Context, eventHash string) (bool, error)
    MarkProcessed(ctx context.Context, eventHash string) error
    
    // Health and status
    Health() HealthStatus
    GetMetrics() ProcessorMetrics
}
```

**Scaling Strategy**:
- Horizontal scaling: Add instances to increase throughput
- Partition by event type: Route different event types to different instances
- Load balancing: Distribute events across instances

### 4. Configuration Center

**Responsibility**: Centralized configuration management

**Deployment**: Highly available cluster (3+ nodes)

**Features**:
- Configuration versioning
- Hot reload capability
- Encryption for sensitive values
- Audit logging for changes
- Rollback support

**Technology Options**:
- Consul: Service discovery + configuration
- etcd: Distributed configuration
- Zookeeper: Distributed coordination

**Interface**:
```go
type ConfigCenter interface {
    // Configuration management
    GetConfig(ctx context.Context, key string) (string, error)
    SetConfig(ctx context.Context, key, value string) error
    WatchConfig(ctx context.Context, key string, handler func(string)) error
    
    // Service discovery
    RegisterService(ctx context.Context, service ServiceInfo) error
    DeregisterService(ctx context.Context, serviceID string) error
    DiscoverService(ctx context.Context, serviceName string) ([]ServiceInfo, error)
}
```

### 5. Message Queue Cluster

**Responsibility**: Temporary event storage and distribution

**Deployment**: Highly available cluster (3+ nodes)

**Features**:
- Partitioned topics for scalability
- Dead letter queue for failed events
- Message retention policies
- Offset tracking for resume capability
- Metrics collection

**Technology Options**:
- Kafka: High throughput, distributed
- Redis: Simple, fast, in-memory
- RabbitMQ: Reliable, feature-rich

### 6. Cache Cluster

**Responsibility**: Fast data access for frequently queried data

**Deployment**: Highly available cluster (3+ nodes)

**Features**:
- TTL-based expiration
- Cache invalidation strategies
- Hit/miss tracking
- Distributed cache coherence

**Technology Options**:
- Redis: High performance, distributed
- Memcached: Simple, fast
- Hazelcast: Java-based distributed cache

### 7. Database Cluster

**Responsibility**: Durable storage of indexed events

**Deployment**: Highly available cluster with replication

**Features**:
- Replication for high availability
- Backup and recovery
- Query optimization
- Connection pooling
- Transaction support

**Technology Options**:
- PostgreSQL: ACID, relational, proven
- MongoDB: Document-oriented, flexible schema

## Data Models

### Event Model
```go
type Event struct {
    ID              string                 // Unique identifier
    EventHash       string                 // Deterministic hash for idempotency
    BlockNumber     uint64                 // Block number
    TransactionHash string                 // Transaction hash
    LogIndex        uint                   // Log index in transaction
    ContractAddress string                 // Contract address
    EventName       string                 // Event name
    EventData       map[string]interface{} // Event data
    ChainID         string                 // Blockchain network identifier
    Timestamp       time.Time              // Event timestamp
    ProcessedAt     time.Time              // Processing timestamp
    Status          string                 // Status (pending, processed, failed)
}
```

### Service Registration Model
```go
type ServiceInfo struct {
    ID              string            // Service instance ID
    Name            string            // Service name
    Address         string            // Service address (host:port)
    Port            int               // Service port
    Tags            []string          // Service tags
    HealthCheckURL  string            // Health check endpoint
    Metadata        map[string]string // Additional metadata
    RegisteredAt    time.Time         // Registration timestamp
}
```

### Configuration Model
```go
type ServiceConfig struct {
    // Service identification
    ServiceName     string
    ServiceID       string
    Environment     string
    
    // Deployment mode
    DeploymentMode  string // "monolithic" or "microservice"
    
    // Data Puller configuration
    DataPullerType  string
    BlockchainNodes []string
    StartBlock      uint64
    
    // Message Queue configuration
    MQType          string
    MQBrokers       []string
    
    // Cache configuration
    CacheType       string
    CacheBrokers    []string
    CacheTTL        time.Duration
    
    // Database configuration
    DatabaseType    string
    DatabaseURL     string
    
    // API configuration
    APIPort         int
    APIProtocols    []string
    
    // Processing configuration
    WorkerPoolSize  int
    BatchSize       int
    MaxRetries      int
    RetryBackoff    time.Duration
}
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property-Based Testing Overview

Property-based testing (PBT) validates software correctness by testing universal properties across many generated inputs. Each property is a formal specification that should hold for all valid inputs.

### Core Principles

1. **Universal Quantification**: Every property must contain an explicit "for all" statement
2. **Requirements Traceability**: Each property must reference the requirements it validates
3. **Executable Specifications**: Properties must be implementable as automated tests
4. **Comprehensive Coverage**: Properties should cover all testable acceptance criteria


## Correctness Properties

### Property 1: Dual Deployment Mode Support
*For any* configuration specifying deployment mode, the system SHALL initialize and run in that mode (monolithic or microservice) without code changes.
**Validates: Requirements 1.1, 1.2, 1.3, 1.5**

### Property 2: API Cluster Load Distribution
*For any* API request arriving at the cluster, the load balancer SHALL distribute requests across healthy API instances.
**Validates: Requirements 2.1, 2.2, 2.4**

### Property 3: Session State in Distributed Cache
*For any* API session, the session state SHALL be stored in distributed cache, not in-process memory, enabling session migration across instances.
**Validates: Requirements 2.3**

### Property 4: Multi-Protocol API Support
*For any* API instance, the instance SHALL support all configured protocols (REST, gRPC, WebSocket, GraphQL) simultaneously.
**Validates: Requirements 2.5**

### Property 5: Data Puller Multi-Chain Support
*For any* blockchain network configured, the Data Puller Cluster SHALL pull events from that network in parallel with other networks.
**Validates: Requirements 3.1, 3.2**

### Property 6: Event Publishing to Message Queue
*For any* blockchain event detected by Data Puller, the event SHALL be published to the message queue partitioned by chain.
**Validates: Requirements 3.2**

### Property 7: WebSocket Event Subscription
*For any* WebSocket subscription to blockchain events, the Data Puller SHALL deliver events in real-time as they occur.
**Validates: Requirements 3.3**

### Property 8: Block Height Tracking
*For any* Data Puller instance, the instance SHALL track the last processed block height per chain and resume from that height after restart.
**Validates: Requirements 3.5**

### Property 9: Event Processor Idempotency
*For any* blockchain event, the Event Processor SHALL generate a deterministic hash that is identical when the same event is processed multiple times.
**Validates: Requirements 4.2**

### Property 10: Exactly-Once Event Processing
*For any* event in the message queue, the Event Processor SHALL process it exactly once, even if the processor crashes and restarts.
**Validates: Requirements 4.5**

### Property 11: Service Registration on Startup
*For any* service instance starting, the instance SHALL register itself with the Service Registry with health status.
**Validates: Requirements 5.1**

### Property 12: Service Deregistration on Shutdown
*For any* service instance shutting down, the instance SHALL deregister itself from the Service Registry.
**Validates: Requirements 5.2**

### Property 13: Health Status Updates
*For any* service instance becoming unhealthy, the Service Registry SHALL mark it as unavailable within 30 seconds.
**Validates: Requirements 5.3**

### Property 14: Service Discovery
*For any* client querying the Service Registry, the registry SHALL return only healthy service endpoints.
**Validates: Requirements 5.4**

### Property 15: Automatic Routing Updates
*For any* change in service endpoints, the system SHALL update client routing automatically without manual intervention.
**Validates: Requirements 5.5**

### Property 16: Stateless Service Design
*For any* service instance, the instance SHALL store all state in external systems (cache, database, message queue), not in-process memory.
**Validates: Requirements 6.1, 6.2**

### Property 17: Horizontal Scaling Without Downtime
*For any* service cluster, adding or removing instances SHALL not cause downtime or data loss.
**Validates: Requirements 6.3**

### Property 18: Consistent Results Across Instances
*For any* request processed by different service instances, the result SHALL be identical.
**Validates: Requirements 6.4**

### Property 19: Distributed Locking
*For any* operation requiring coordination across instances, the system SHALL use distributed locks to prevent conflicts.
**Validates: Requirements 6.5**

### Property 20: Configuration Propagation
*For any* configuration update in the Configuration Center, the update SHALL propagate to all affected services within 5 seconds.
**Validates: Requirements 7.2**

### Property 21: Configuration Versioning
*For any* configuration change, the Configuration Center SHALL maintain version history enabling rollback.
**Validates: Requirements 7.3**

### Property 22: Sensitive Configuration Encryption
*For any* sensitive configuration value (passwords, API keys), the value SHALL be encrypted at rest in the Configuration Center.
**Validates: Requirements 7.4**

### Property 23: Configuration Audit Logging
*For any* configuration change, the Configuration Center SHALL log the change with timestamp, user, and old/new values.
**Validates: Requirements 7.4**

### Property 24: Failure Detection
*For any* service instance failure, the system SHALL detect the failure within 30 seconds.
**Validates: Requirements 8.1**

### Property 25: Automatic Workload Migration
*For any* failed service instance, the system SHALL automatically migrate its workload to healthy instances.
**Validates: Requirements 8.2**

### Property 26: Data Consistency During Failover
*For any* failover event, the system SHALL maintain data consistency and prevent data loss.
**Validates: Requirements 8.3**

### Property 27: Graceful Shutdown
*For any* service instance shutting down, the instance SHALL drain existing connections before terminating.
**Validates: Requirements 8.4**

### Property 28: Circuit Breaker Pattern
*For any* cascading failure scenario, the system SHALL implement circuit breaker pattern to prevent failure propagation.
**Validates: Requirements 8.5**

### Property 29: Blockchain-Specific Cluster Isolation
*For any* blockchain cluster (EVM, Cosmos, Solana), the cluster SHALL be isolated with independent data and configuration.
**Validates: Requirements 9.1, 9.2**

### Property 30: Independent Blockchain Scaling
*For any* blockchain cluster, the cluster SHALL be independently scalable without affecting other blockchain clusters.
**Validates: Requirements 9.3**

### Property 31: Cross-Chain Unified API
*For any* cross-chain query, the unified API SHALL aggregate results from multiple blockchain clusters.
**Validates: Requirements 9.4**

### Property 32: Blockchain-Specific Logic Application
*For any* blockchain-specific logic, the logic SHALL be applied only to the relevant blockchain cluster.
**Validates: Requirements 9.5**

### Property 33: Database Replication
*For any* data written to the Database Cluster, the data SHALL be replicated to at least 2 other nodes.
**Validates: Requirements 10.1**

### Property 34: Database Failover
*For any* database node failure, the system SHALL automatically promote a replica to primary.
**Validates: Requirements 10.4**

### Property 35: ACID Properties Across Distributed Nodes
*For any* transaction in the Database Cluster, the transaction SHALL maintain ACID properties across all nodes.
**Validates: Requirements 10.2**

### Property 36: NoSQL Cache TTL
*For any* data in the NoSQL Cluster with TTL, the data SHALL be automatically evicted after TTL expires.
**Validates: Requirements 10.3**

### Property 37: Backup and Recovery
*For any* backup of the Database Cluster, the backup SHALL be restorable to a consistent state.
**Validates: Requirements 10.5**

## Error Handling Strategy

### Error Classification

1. **Transient Errors**: Network timeouts, temporary service unavailability
   - Strategy: Exponential backoff retry
   - Max retries: 3
   - Backoff: 100ms, 200ms, 400ms

2. **Permanent Errors**: Invalid configuration, authentication failures
   - Strategy: Log and alert, move to dead letter queue
   - Action: Manual intervention required

3. **Critical Errors**: Data corruption, system resource exhaustion
   - Strategy: Enter safe state, prevent further operations
   - Action: Immediate alert and graceful shutdown

### Retry Logic with Circuit Breaker

```go
type CircuitBreaker struct {
    MaxFailures      int
    ResetTimeout     time.Duration
    State            string // "closed", "open", "half-open"
    FailureCount     int
    LastFailureTime  time.Time
}

func (cb *CircuitBreaker) Call(operation func() error) error {
    if cb.State == "open" {
        if time.Since(cb.LastFailureTime) > cb.ResetTimeout {
            cb.State = "half-open"
        } else {
            return fmt.Errorf("circuit breaker open")
        }
    }
    
    err := operation()
    if err != nil {
        cb.FailureCount++
        cb.LastFailureTime = time.Now()
        if cb.FailureCount >= cb.MaxFailures {
            cb.State = "open"
        }
        return err
    }
    
    cb.FailureCount = 0
    cb.State = "closed"
    return nil
}
```

## Observability Strategy

### Metrics Collection

- **Service Metrics**: Request latency, throughput, error rate
- **Resource Metrics**: CPU, memory, disk usage
- **Queue Metrics**: Queue depth, processing latency
- **Cache Metrics**: Hit rate, miss rate, eviction rate
- **Database Metrics**: Query latency, connection pool status

### Distributed Tracing

- **Correlation IDs**: Trace requests across services
- **Span Creation**: Create spans for major operations
- **Span Linking**: Link spans across service boundaries

### Structured Logging

- **JSON Format**: Structured logs for easy parsing
- **Correlation IDs**: Include in all logs for tracing
- **Context Information**: Include relevant context in logs

## Testing Strategy

### Unit Testing

- Test individual components in isolation
- Mock external dependencies
- Test error conditions and edge cases
- Aim for >80% code coverage

### Property-Based Testing

- Test universal properties across many inputs
- Validate correctness properties from design
- Use property testing frameworks (QuickCheck, Hypothesis, fast-check)
- Minimum 100 iterations per property test

### Integration Testing

- Test component interactions
- Use test containers for external services
- Test end-to-end workflows
- Validate data consistency across components

### Chaos Engineering

- Test system resilience to failures
- Simulate service crashes
- Simulate network partitions
- Validate automatic recovery

## Deployment Considerations

### Monolithic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chainpulse-monolithic
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: chainpulse
        image: chainpulse:latest
        env:
        - name: DEPLOYMENT_MODE
          value: "monolithic"
```

### Microservice Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chainpulse-api
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api
        image: chainpulse-api:latest
        env:
        - name: DEPLOYMENT_MODE
          value: "microservice"
        - name: SERVICE_NAME
          value: "api"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: chainpulse-data-puller
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: data-puller
        image: chainpulse-data-puller:latest
        env:
        - name: DEPLOYMENT_MODE
          value: "microservice"
        - name: SERVICE_NAME
          value: "data-puller"
```

## Migration Path

### Phase 1: Preparation
- Set up Configuration Center
- Set up Message Queue Cluster
- Set up Cache Cluster
- Set up Database Cluster

### Phase 2: API Layer
- Deploy API instances behind load balancer
- Migrate session state to distributed cache
- Validate multi-protocol support

### Phase 3: Data Puller
- Deploy Data Puller instances
- Configure per-chain instances
- Validate event publishing to message queue

### Phase 4: Event Processor
- Deploy Event Processor instances
- Configure idempotency checking
- Validate event storage

### Phase 5: Service Discovery
- Deploy Service Registry
- Register all services
- Validate automatic discovery

### Phase 6: Cutover
- Switch traffic to distributed architecture
- Monitor for issues
- Maintain rollback capability

