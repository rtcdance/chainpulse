# Microservices Data Layer Integration - Design

**Date:** January 12, 2026  
**Status:** Draft

## Overview

This design document specifies how to integrate MongoDB (NoSQL) and PostgreSQL (relational) database layers into the API Service and Event Processor microservices. The architecture follows a cache-first pattern where MongoDB serves as the primary data store for fast reads/writes, with PostgreSQL as the fallback for structured data and complex queries.

## Architecture

### Data Layer Stack

```
┌─────────────────────────────────────────────────────────┐
│                  Microservice                           │
│         (API Service / Event Processor)                 │
└────────────────────┬────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
    ┌────────┐  ┌────────┐  ┌────────┐
    │ Cache  │  │ NoSQL  │  │ SQL    │
    │ Layer  │  │ (Mongo)│  │(Postgres)
    │(Redis) │  │        │  │        │
    └────────┘  └────────┘  └────────┘
```

### Query Flow

**Read Operations:**
1. Check Redis cache first (fastest)
2. If miss, query MongoDB (fast)
3. If miss, query PostgreSQL (slower but authoritative)
4. Cache result in Redis and MongoDB for future requests

**Write Operations:**
1. Write to MongoDB first (fast)
2. Write to PostgreSQL (for consistency)
3. Invalidate Redis cache
4. Return success only if both writes succeed

## Components

### 1. Database Connection Manager

**Purpose:** Manage connection pools for both MongoDB and PostgreSQL

**Responsibilities:**
- Initialize connection pools on startup
- Provide connection acquisition and release
- Implement health checks
- Handle connection failures and reconnection
- Track connection pool metrics

**Interface:**
```go
type DatabaseManager interface {
    // MongoDB operations
    GetMongoClient() *mongo.Client
    GetMongoDatabase(name string) *mongo.Database
    
    // PostgreSQL operations
    GetPostgresDB() *sql.DB
    
    // Health checks
    CheckMongoHealth(ctx context.Context) error
    CheckPostgresHealth(ctx context.Context) error
    
    // Lifecycle
    Close(ctx context.Context) error
}
```

### 2. API Service Data Layer

**Purpose:** Provide data access for API Service queries

**Components:**

#### a. Query Service
- Handles REST/GraphQL query execution
- Implements cache-first pattern
- Falls back to databases as needed
- Caches results

#### b. MongoDB Adapter
- Connects to MongoDB
- Executes document queries
- Implements batch operations
- Manages indexes

#### c. PostgreSQL Adapter
- Connects to PostgreSQL
- Executes SQL queries
- Manages transactions
- Implements prepared statements

#### d. Cache Service
- Manages Redis cache
- Implements TTL policies
- Handles cache invalidation

### 3. Event Processor Data Layer

**Purpose:** Store and retrieve processed events

**Components:**

#### a. Event Store
- Stores processed events in MongoDB
- Implements batch inserts
- Manages event indexes
- Implements TTL policies

#### b. Event Metadata Store
- Stores event metadata in PostgreSQL
- Maintains referential integrity
- Supports analytics queries
- Implements transactions

#### c. Event Retrieval Service
- Queries events from MongoDB
- Queries metadata from PostgreSQL
- Joins data as needed
- Implements pagination

## Data Models

### MongoDB Collections

#### Events Collection
```javascript
{
  _id: ObjectId,
  eventId: string,
  chainId: number,
  blockNumber: number,
  transactionHash: string,
  logIndex: number,
  contractAddress: string,
  eventName: string,
  eventData: object,
  decodedData: object,
  timestamp: Date,
  processedAt: Date,
  createdAt: Date,
  expiresAt: Date  // TTL index
}
```

#### Indexes:
- `{ chainId: 1, blockNumber: 1 }`
- `{ contractAddress: 1, eventName: 1 }`
- `{ timestamp: 1 }`
- `{ expiresAt: 1 }` (TTL index)

### PostgreSQL Tables

#### events_metadata
```sql
CREATE TABLE events_metadata (
  id BIGSERIAL PRIMARY KEY,
  event_id VARCHAR(255) UNIQUE NOT NULL,
  chain_id INTEGER NOT NULL,
  block_number BIGINT NOT NULL,
  transaction_hash VARCHAR(255) NOT NULL,
  log_index INTEGER NOT NULL,
  contract_address VARCHAR(255) NOT NULL,
  event_name VARCHAR(255) NOT NULL,
  processed_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  
  FOREIGN KEY (chain_id) REFERENCES chains(id),
  INDEX idx_chain_block (chain_id, block_number),
  INDEX idx_contract_event (contract_address, event_name),
  INDEX idx_processed_at (processed_at)
);
```

## Configuration

### Environment Variables

```bash
# MongoDB Configuration
MONGODB_URI=mongodb://mongo-1:27017,mongo-2:27017,mongo-3:27017
MONGODB_DATABASE=chainpulse
MONGODB_TIMEOUT_MS=5000

# PostgreSQL Configuration
DATABASE_URL=postgres://user:password@postgres-primary:5432/chainpulse
DATABASE_REPLICA_URL=postgres://user:password@postgres-replica:5432/chainpulse
DATABASE_POOL_SIZE=20
DATABASE_TIMEOUT_MS=5000

# Connection Pool Configuration
DB_POOL_SIZE=10
DB_TIMEOUT_MS=5000
DB_RETRY_ATTEMPTS=3
DB_RETRY_DELAY_MS=100

# Cache Configuration
REDIS_CLUSTER=redis-1:6379,redis-2:6379,redis-3:6379
CACHE_TTL_SECONDS=3600

# Event Processor Configuration
EVENT_BATCH_SIZE=100
EVENT_TTL_DAYS=30
```

## Error Handling

### Database Connection Errors

```
Connection Failed
  ├─ Retry with exponential backoff (max 3 attempts)
  ├─ If all retries fail, mark database as unavailable
  ├─ Use circuit breaker pattern
  └─ Return error to client with appropriate status code
```

### Data Consistency Errors

```
Write to MongoDB succeeds, PostgreSQL fails
  ├─ Log inconsistency
  ├─ Attempt to write to PostgreSQL again
  ├─ If still fails, queue for async reconciliation
  └─ Return partial success to client
```

### Query Errors

```
Query fails on all databases
  ├─ Log error with context
  ├─ Check database health
  ├─ Return error to client
  └─ Trigger monitoring alert
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Cache-First Consistency

**For any** query operation, if data exists in the cache, the cached data should be returned without querying the databases.

**Validates: Requirements 1.2, 1.3**

### Property 2: Fallback Chain Correctness

**For any** query operation, if MongoDB returns no results, the system should query PostgreSQL and return those results.

**Validates: Requirements 1.3, 2.2**

### Property 3: Write Atomicity

**For any** write operation to both MongoDB and PostgreSQL, either both writes succeed or both fail (no partial writes).

**Validates: Requirements 8.1**

### Property 4: Connection Pool Reuse

**For any** sequence of database operations, connections should be reused from the pool rather than creating new connections.

**Validates: Requirements 6.1, 6.3**

### Property 5: Health Check Accuracy

**For any** database health check, the result should accurately reflect the current connectivity status of that database.

**Validates: Requirements 1.7, 2.7, 3.7, 4.7**

### Property 6: Batch Insert Completeness

**For any** batch insert operation, all records in the batch should be inserted or none should be inserted (atomic batch operation).

**Validates: Requirements 3.3, 4.3**

### Property 7: TTL Expiration

**For any** document in MongoDB with a TTL index, the document should be automatically deleted after the TTL period expires.

**Validates: Requirements 3.5**

### Property 8: Error Logging Completeness

**For any** database operation failure, the error should be logged with sufficient context (operation type, database, error details) for debugging.

**Validates: Requirements 7.1**

## Testing Strategy

### Unit Tests

- Test connection pool creation and management
- Test query execution with mocked databases
- Test error handling and retry logic
- Test cache invalidation
- Test batch operations
- Test health checks

### Integration Tests

- Test with real MongoDB and PostgreSQL instances
- Test cache-first pattern with actual data
- Test fallback chain (cache → MongoDB → PostgreSQL)
- Test write operations to both databases
- Test connection pool under load
- Test graceful shutdown

### Property-Based Tests

- Generate random queries and verify cache-first behavior
- Generate random write operations and verify atomicity
- Generate random connection sequences and verify pool reuse
- Generate random batch operations and verify completeness
- Generate random TTL scenarios and verify expiration

## Implementation Phases

### Phase 1: Database Connection Management
- Implement DatabaseManager interface
- Create MongoDB connection pool
- Create PostgreSQL connection pool
- Implement health checks

### Phase 2: API Service Data Layer
- Implement Query Service
- Implement MongoDB Adapter
- Implement PostgreSQL Adapter
- Implement cache integration

### Phase 3: Event Processor Data Layer
- Implement Event Store
- Implement Event Metadata Store
- Implement batch operations
- Implement TTL policies

### Phase 4: Error Handling and Resilience
- Implement circuit breaker pattern
- Implement retry logic with exponential backoff
- Implement comprehensive error logging
- Implement monitoring and alerting

### Phase 5: Testing and Optimization
- Write comprehensive unit tests
- Write integration tests
- Write property-based tests
- Performance optimization and tuning

## Deployment Considerations

### Database Initialization

- Create MongoDB collections and indexes on startup
- Create PostgreSQL tables and indexes on startup
- Verify database connectivity before service starts
- Fail fast if databases are unavailable

### Monitoring

- Monitor connection pool metrics
- Monitor query latency (MongoDB vs PostgreSQL)
- Monitor cache hit rates
- Monitor error rates and types
- Alert on database connectivity issues

### Scaling

- Connection pool size should scale with expected load
- MongoDB should be deployed as a replica set
- PostgreSQL should be deployed with read replicas
- Redis should be deployed as a cluster

## Security Considerations

- Database credentials should be loaded from environment variables
- Connection strings should support SSL/TLS encryption
- Database operations should use prepared statements
- Access to sensitive data should be logged
- Database backups should be encrypted
