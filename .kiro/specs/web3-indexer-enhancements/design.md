# Design Document: Web3 Indexer Enhancements

## Overview

This design document describes the architecture and implementation approach for enhancing ChainPulse into a production-ready Web3 indexer with advanced filtering, optimization, integrations, and monitoring.

## Architecture

### High-Level Architecture

```
Data Puller → Event Processor → Event Indexer → Query Optimizer → API Gateway
                                      ↓
                            Event Decoder + Contract Manager
                                      ↓
                            Reorg Handler + Consistency Checker
                                      ↓
                            Indexer Metrics + Health Checker
```

### Component Responsibilities

1. **Event Indexer**: Indexes events with advanced filtering
2. **Event Decoder**: Decodes raw events into structured data
3. **Contract Manager**: Manages contract ABIs
4. **Query Optimizer**: Optimizes queries and manages caching
5. **Reorg Handler**: Detects and recovers from reorgs
6. **Consistency Checker**: Verifies data consistency
7. **Indexer Metrics**: Tracks indexer-specific metrics
8. **Health Checker**: Provides health status

## Components and Interfaces

### 1. Advanced Event Filter

```go
type EventFilter struct {
    Network         string
    ContractAddress []common.Address
    EventSignature  []common.Hash
    Topics          [][]common.Hash
    FromBlock       uint64
    ToBlock         uint64
    FromTimestamp   int64
    ToTimestamp     int64
    Status          []string
    Limit           int
    Offset          int
}

func (ef *EventFilter) Validate() error
func (ef *EventFilter) ToQuery() string
```

### 2. Event Indexer

```go
type EventIndexer struct {
    database DatabasePlugin
    cache    CachePlugin
    logger   Logger
}

func (ei *EventIndexer) IndexEvent(ctx context.Context, event *BlockchainEvent) error
func (ei *EventIndexer) IndexBatch(ctx context.Context, events []*BlockchainEvent) error
func (ei *EventIndexer) QueryEvents(ctx context.Context, filter *EventFilter) ([]*BlockchainEvent, error)
func (ei *EventIndexer) GetEventsByTopic(ctx context.Context, topic common.Hash, fromBlock, toBlock uint64) ([]*BlockchainEvent, error)
func (ei *EventIndexer) GetEventsByAddress(ctx context.Context, address common.Address, fromBlock, toBlock uint64) ([]*BlockchainEvent, error)
```

### 3. Event Decoder

```go
type EventDecoder struct {
    contractManager *ContractManager
    logger          Logger
}

func (ed *EventDecoder) DecodeEvent(ctx context.Context, rawEvent *types.Log, contractABI abi.ABI) (*DecodedEvent, error)
func (ed *EventDecoder) DecodeEventBatch(ctx context.Context, rawEvents []*types.Log, contractABI abi.ABI) ([]*DecodedEvent, error)

type DecodedEvent struct {
    EventName  string
    Parameters map[string]interface{}
    Indexed    map[string]interface{}
    NonIndexed map[string]interface{}
}
```

### 4. Contract Manager

```go
type ContractManager struct {
    contracts map[string]*ContractMetadata
    mu        sync.RWMutex
}

func (cm *ContractManager) LoadContractABI(name string, abiJSON []byte) error
func (cm *ContractManager) GetEventSignature(contractName, eventName string) (common.Hash, error)
func (cm *ContractManager) GetABI(contractName string) (abi.ABI, error)
```

### 5. Query Optimizer

```go
type QueryOptimizer struct {
    database DatabasePlugin
    cache    CachePlugin
    logger   Logger
}

func (qo *QueryOptimizer) OptimizeQuery(filter *EventFilter) *OptimizedQuery
func (qo *QueryOptimizer) ExecuteOptimized(ctx context.Context, optimized *OptimizedQuery) ([]*BlockchainEvent, error)

type OptimizedQuery struct {
    Query           string
    Indexes         []string
    CacheKey        string
    CacheTTL        time.Duration
    EstimatedRows   int64
    ExecutionPlan   string
}
```

### 6. Reorg Handler

```go
type ReorgHandler struct {
    database DatabasePlugin
    logger   Logger
    metrics  *IndexerMetrics
}

func (rh *ReorgHandler) DetectReorg(ctx context.Context, currentBlock uint64, newBlockHash common.Hash) (bool, uint64, error)
func (rh *ReorgHandler) HandleReorg(ctx context.Context, reorgBlock uint64) error
func (rh *ReorgHandler) RollbackEvents(ctx context.Context, blockNumber uint64) error
```

### 7. Consistency Checker

```go
type ConsistencyChecker struct {
    database DatabasePlugin
    logger   Logger
}

func (cc *ConsistencyChecker) CheckConsistency(ctx context.Context) (*ConsistencyReport, error)
func (cc *ConsistencyChecker) VerifyEventSequence(ctx context.Context) error
func (cc *ConsistencyChecker) VerifyBlockSequence(ctx context.Context) error
func (cc *ConsistencyChecker) RepairInconsistencies(ctx context.Context) error
```

### 8. Indexer Metrics

```go
type IndexerMetrics struct {
    CurrentBlockNumber   prometheus.Gauge
    LatestBlockNumber    prometheus.Gauge
    IndexingLag          prometheus.Gauge
    EventsIndexed        prometheus.Counter
    EventsProcessed      prometheus.Counter
    EventsFailed         prometheus.Counter
    IndexingLatency      prometheus.Histogram
    QueryLatency         prometheus.Histogram
    ReorgDetected        prometheus.Counter
}

func (im *IndexerMetrics) RecordIndexingProgress(currentBlock, latestBlock uint64)
func (im *IndexerMetrics) RecordEventIndexed(latency time.Duration)
func (im *IndexerMetrics) RecordReorg(blocksRolledBack uint64)
```

## Data Models

### Enhanced BlockchainEvent

```go
type BlockchainEvent struct {
    ID              string
    EventHash       string
    EventSignature  common.Hash
    BlockNumber     uint64
    BlockHash       common.Hash
    BlockTimestamp  int64
    TransactionHash common.Hash
    TransactionIndex uint
    GasUsed         uint64
    GasPrice        *big.Int
    LogIndex        uint
    Removed         bool
    ContractAddress common.Address
    EventName       string
    EventTopic      []common.Hash
    EventData       []byte
    DecodedData     map[string]interface{}
    ChainID         string
    Network         string
    Status          EventStatus
    CreatedAt       time.Time
    ProcessedAt     time.Time
    IndexedAt       time.Time
}

type EventStatus string

const (
    EventStatusPending    EventStatus = "pending"
    EventStatusConfirmed  EventStatus = "confirmed"
    EventStatusFailed     EventStatus = "failed"
    EventStatusReorged    EventStatus = "reorged"
)
```

### Transaction Model

```go
type Transaction struct {
    Hash            common.Hash
    From            common.Address
    To              *common.Address
    Value           *big.Int
    Gas             uint64
    GasPrice        *big.Int
    Input           []byte
    Nonce           uint64
    BlockNumber     uint64
    BlockHash       common.Hash
    TransactionIndex uint
    Status          uint64
    ContractAddress *common.Address
    CumulativeGasUsed uint64
    Logs            []*types.Log
}
```

### Block Model

```go
type Block struct {
    Number       uint64
    Hash         common.Hash
    ParentHash   common.Hash
    Timestamp    int64
    Miner        common.Address
    Difficulty   *big.Int
    GasLimit     uint64
    GasUsed      uint64
    Transactions []common.Hash
    LogsBloom    types.Bloom
}
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.

### Property 1: Event Filtering Completeness
**For any** event filter and any set of events, all returned events SHALL match the filter criteria exactly.
**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: Query Optimization Correctness
**For any** query, the optimized query SHALL return identical results to the unoptimized query.
**Validates: Requirements 3.1, 3.2, 3.3**

### Property 3: Cache Consistency
**For any** cached query result, the cached result SHALL match the current database state until cache TTL expires.
**Validates: Requirements 3.4, 3.5, 3.6**

### Property 4: Event Decoding Accuracy
**For any** raw blockchain event and valid ABI, the decoded event SHALL contain all indexed and non-indexed parameters correctly.
**Validates: Requirements 8.1, 8.2, 8.3**

### Property 5: Reorg Recovery Completeness
**For any** reorg, after recovery, the database SHALL contain exactly the events that exist on the canonical chain.
**Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**

### Property 6: Data Consistency Invariant
**For any** indexed event, the event SHALL not appear as duplicate in the database.
**Validates: Requirements 7.1, 7.2, 7.3**

### Property 7: Indexing Lag Boundedness
**For any** time period, the indexing lag SHALL not exceed configured maximum blocks.
**Validates: Requirements 5.1, 5.2**

### Property 8: Query Performance SLA
**For any** query, the query latency SHALL be less than configured SLA (200ms for 99th percentile).
**Validates: Requirements 9.2, 9.3, 9.4**

## Error Handling

### Error Classification

1. **Transient Errors**: Network timeouts, temporary database unavailability
   - Action: Retry with exponential backoff
   - Max retries: 3
   - Backoff: 100ms, 200ms, 400ms

2. **Permanent Errors**: Invalid configuration, corrupted data, invalid ABI
   - Action: Log error and alert operator
   - Recovery: Manual intervention required

3. **Critical Errors**: Data corruption, reorg detection failure
   - Action: Enter safe state, prevent further indexing
   - Recovery: Operator intervention required

### Error Handling Strategy

- All errors logged with full context and stack trace
- Errors include correlation IDs for distributed tracing
- Errors propagated with wrapped context
- Graceful degradation where possible

## Testing Strategy

### Unit Tests
- Test each component in isolation
- Mock external dependencies
- Test error cases and edge cases
- Target: >90% code coverage

### Property-Based Tests
- Test correctness properties with generated inputs
- Verify properties hold across many inputs
- Test edge cases and boundary conditions
- Minimum 100 iterations per property

### Integration Tests
- Test component interactions
- Use in-memory or containerized services
- Test end-to-end workflows
- Test error recovery

### E2E Tests
- Test complete indexing flow
- Test query flow
- Test reorg flow
- Test monitoring and alerting

## Deployment

### Deployment Modes

1. **Monolithic**: All components in single binary
2. **Microservice**: Independent services communicating via message queue

### Deployment Platforms

1. **Docker**: Containerized deployment
2. **Kubernetes**: Orchestrated deployment
3. **Cloud**: AWS, GCP, Azure support

### Configuration

- Environment variables for all settings
- Sensible defaults for all parameters
- Validation on startup
- Hot reload support where applicable

## Monitoring and Observability

### Metrics

- Indexing lag (latest_block - current_block)
- Events indexed per second
- Query latency (p50, p95, p99)
- Cache hit rate
- Reorg frequency and impact
- Error rate by type

### Logging

- Structured JSON logging
- Correlation IDs for tracing
- Log levels: DEBUG, INFO, WARN, ERROR, FATAL
- Sensitive data redaction

### Health Checks

- Indexing status (current block, latest block, lag)
- Database connectivity
- Cache connectivity
- Message queue connectivity

## Performance Targets

- Indexing latency: < 100ms per event
- Query latency: < 200ms (99th percentile)
- Cache hit latency: < 10ms
- Database query latency: < 100ms (indexed queries)
- Indexing throughput: > 10,000 events/sec
- Cache hit ratio: > 80%
- Indexing lag: < 10 blocks

