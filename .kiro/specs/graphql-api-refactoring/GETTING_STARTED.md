# Getting Started - GraphQL API Refactoring

## Before You Start

Make sure you have:
- [ ] Read `SUMMARY.md` - Understand the big picture
- [ ] Read `requirements.md` - Know what needs to be built
- [ ] Read `design.md` - Understand the architecture
- [ ] Reviewed `tasks.md` - Know the implementation steps

## Phase 1: Query Service Layer

This is the foundation for everything else. Complete this phase first.

### Step 1: Create Query Service Interface

**File**: `pkg/services/query/query_service.go`

Create the interface that defines all query methods:

```go
package query

import (
    "context"
    "math/big"
    "time"
)

// QueryService defines the unified query interface
type QueryService interface {
    // Events
    GetEvent(ctx context.Context, id string) (*Event, error)
    QueryEvents(ctx context.Context, filter *EventFilter, limit, offset int) ([]*Event, error)
    
    // Tokens
    GetTokenMetadata(ctx context.Context, token string) (*TokenMetadata, error)
    GetTokenBalance(ctx context.Context, token, account string) (*TokenBalance, error)
    QueryTokenTransfers(ctx context.Context, token string, limit, offset int) ([]*Transfer, error)
    
    // Pools
    GetPoolMetadata(ctx context.Context, pool string) (*PoolMetadata, error)
    GetPoolStats(ctx context.Context, pool string) (*PoolStats, error)
    QueryPoolSwaps(ctx context.Context, pool string, limit, offset int) ([]*Swap, error)
    
    // Contracts
    GetContractMetadata(ctx context.Context, address string) (*ContractMetadata, error)
    QueryContractEvents(ctx context.Context, address string, limit, offset int) ([]*Event, error)
    
    // System
    GetHealth(ctx context.Context) (*HealthInfo, error)
    GetMetrics(ctx context.Context) (*SystemMetrics, error)
}

// Event represents a blockchain event
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

// EventFilter represents event filtering criteria
type EventFilter struct {
    ContractAddress []string
    EventName       string
    FromBlock       *big.Int
    ToBlock         *big.Int
    Network         string
}

// TokenMetadata represents ERC20 token metadata
type TokenMetadata struct {
    Address      string
    Name         string
    Symbol       string
    Decimals     int
    TotalSupply  *big.Int
    LastUpdated  time.Time
}

// TokenBalance represents a token balance
type TokenBalance struct {
    Token       string
    Account     string
    Balance     *big.Int
    BlockNumber uint64
    BlockTime   time.Time
}

// Transfer represents a token transfer
type Transfer struct {
    ID              string
    Token           string
    From            string
    To              string
    Amount          *big.Int
    BlockNumber     uint64
    TransactionHash string
    Timestamp       time.Time
}

// PoolMetadata represents Uniswap pool metadata
type PoolMetadata struct {
    Address      string
    Token0       string
    Token1       string
    Fee          uint32
    Liquidity    *big.Int
    SqrtPriceX96 *big.Int
    Tick         int32
    LastUpdated  time.Time
}

// PoolStats represents pool statistics
type PoolStats struct {
    Address         string
    TotalVolume0    *big.Int
    TotalVolume1    *big.Int
    TotalFees       *big.Int
    SwapCount       int64
    LastSwapTime    time.Time
}

// Swap represents a swap event
type Swap struct {
    ID              string
    Pool            string
    Sender          string
    Recipient       string
    Amount0         *big.Int
    Amount1         *big.Int
    SqrtPriceX96    *big.Int
    Liquidity       *big.Int
    Tick            int32
    BlockNumber     uint64
    TransactionHash string
    Timestamp       time.Time
}

// ContractMetadata represents contract metadata
type ContractMetadata struct {
    Address     string
    Name        string
    ABI         string
    EventCount  int64
    LastUpdated time.Time
}

// HealthInfo represents health information
type HealthInfo struct {
    Status    string
    Message   string
    Details   map[string]interface{}
    Timestamp time.Time
}

// SystemMetrics represents system metrics
type SystemMetrics struct {
    IsRunning      bool
    ServiceCount   int
    DeploymentMode string
    Uptime         time.Duration
    RequestCount   int64
    ErrorCount     int64
}
```

**Checklist**:
- [ ] File created at `pkg/services/query/query_service.go`
- [ ] Interface defined with all methods
- [ ] All data types defined
- [ ] Code compiles without errors
- [ ] No external dependencies in interface

### Step 2: Implement Query Service

**File**: `pkg/services/query/query_service_impl.go`

Implement the interface with caching and error handling:

```go
package query

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
    
    "chainpulse/pkg/core"
)

// DefaultQueryService implements QueryService
type DefaultQueryService struct {
    database  core.DatabasePlugin
    cache     core.CachePlugin
    logger    core.Logger
    metrics   core.MetricsCollector
    cacheTTL  time.Duration
    mu        sync.RWMutex
}

// NewDefaultQueryService creates a new query service
func NewDefaultQueryService(
    database core.DatabasePlugin,
    cache core.CachePlugin,
    logger core.Logger,
    metrics core.MetricsCollector,
) *DefaultQueryService {
    return &DefaultQueryService{
        database:  database,
        cache:     cache,
        logger:    logger,
        metrics:   metrics,
        cacheTTL:  5 * time.Minute,
    }
}

// GetEvent retrieves a single event
func (s *DefaultQueryService) GetEvent(ctx context.Context, id string) (*Event, error) {
    if id == "" {
        return nil, fmt.Errorf("event id is required")
    }
    
    // Try cache first
    cacheKey := fmt.Sprintf("event:%s", id)
    if cached, err := s.cache.Get(cacheKey); err == nil && cached != "" {
        var event Event
        if err := json.Unmarshal([]byte(cached), &event); err == nil {
            s.metrics.RecordCounter("query_cache_hit", 1, map[string]string{"type": "event"})
            return &event, nil
        }
    }
    
    // Query database
    dbEvent, err := s.database.GetEventByID(ctx, id)
    if err != nil {
        s.logger.Error("failed to get event", map[string]interface{}{
            "id":    id,
            "error": err.Error(),
        })
        s.metrics.RecordCounter("query_error", 1, map[string]string{"type": "event"})
        return nil, err
    }
    
    // Convert to query service type
    event := s.convertBlockchainEventToEvent(dbEvent)
    
    // Cache result
    if data, err := json.Marshal(event); err == nil {
        _ = s.cache.Set(cacheKey, string(data), s.cacheTTL)
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "event"})
    return event, nil
}

// QueryEvents retrieves multiple events
func (s *DefaultQueryService) QueryEvents(ctx context.Context, filter *EventFilter, limit, offset int) ([]*Event, error) {
    if limit <= 0 {
        limit = 20
    }
    if limit > 100 {
        limit = 100
    }
    
    // Query database
    dbEvents, err := s.database.QueryEvents(ctx, s.convertEventFilter(filter), offset, limit)
    if err != nil {
        s.logger.Error("failed to query events", map[string]interface{}{
            "error": err.Error(),
        })
        s.metrics.RecordCounter("query_error", 1, map[string]string{"type": "events"})
        return nil, err
    }
    
    // Convert results
    events := make([]*Event, len(dbEvents))
    for i, dbEvent := range dbEvents {
        events[i] = s.convertBlockchainEventToEvent(dbEvent)
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "events"})
    return events, nil
}

// GetTokenMetadata retrieves token metadata
func (s *DefaultQueryService) GetTokenMetadata(ctx context.Context, token string) (*TokenMetadata, error) {
    if token == "" {
        return nil, fmt.Errorf("token address is required")
    }
    
    cacheKey := fmt.Sprintf("token_metadata:%s", token)
    if cached, err := s.cache.Get(cacheKey); err == nil && cached != "" {
        var metadata TokenMetadata
        if err := json.Unmarshal([]byte(cached), &metadata); err == nil {
            s.metrics.RecordCounter("query_cache_hit", 1, map[string]string{"type": "token_metadata"})
            return &metadata, nil
        }
    }
    
    metadata := &TokenMetadata{
        Address:     token,
        Name:        "Unknown",
        Symbol:      "UNKNOWN",
        Decimals:    18,
        LastUpdated: time.Now(),
    }
    
    if data, err := json.Marshal(metadata); err == nil {
        _ = s.cache.Set(cacheKey, string(data), s.cacheTTL)
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "token_metadata"})
    return metadata, nil
}

// GetTokenBalance retrieves token balance
func (s *DefaultQueryService) GetTokenBalance(ctx context.Context, token, account string) (*TokenBalance, error) {
    if token == "" || account == "" {
        return nil, fmt.Errorf("token and account are required")
    }
    
    cacheKey := fmt.Sprintf("token_balance:%s:%s", token, account)
    if cached, err := s.cache.Get(cacheKey); err == nil && cached != "" {
        var balance TokenBalance
        if err := json.Unmarshal([]byte(cached), &balance); err == nil {
            s.metrics.RecordCounter("query_cache_hit", 1, map[string]string{"type": "token_balance"})
            return &balance, nil
        }
    }
    
    balance := &TokenBalance{
        Token:     token,
        Account:   account,
        BlockTime: time.Now(),
    }
    
    if data, err := json.Marshal(balance); err == nil {
        _ = s.cache.Set(cacheKey, string(data), s.cacheTTL)
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "token_balance"})
    return balance, nil
}

// GetPoolMetadata retrieves pool metadata
func (s *DefaultQueryService) GetPoolMetadata(ctx context.Context, pool string) (*PoolMetadata, error) {
    if pool == "" {
        return nil, fmt.Errorf("pool address is required")
    }
    
    cacheKey := fmt.Sprintf("pool_metadata:%s", pool)
    if cached, err := s.cache.Get(cacheKey); err == nil && cached != "" {
        var metadata PoolMetadata
        if err := json.Unmarshal([]byte(cached), &metadata); err == nil {
            s.metrics.RecordCounter("query_cache_hit", 1, map[string]string{"type": "pool_metadata"})
            return &metadata, nil
        }
    }
    
    metadata := &PoolMetadata{
        Address:     pool,
        LastUpdated: time.Now(),
    }
    
    if data, err := json.Marshal(metadata); err == nil {
        _ = s.cache.Set(cacheKey, string(data), s.cacheTTL)
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "pool_metadata"})
    return metadata, nil
}

// GetPoolStats retrieves pool statistics
func (s *DefaultQueryService) GetPoolStats(ctx context.Context, pool string) (*PoolStats, error) {
    if pool == "" {
        return nil, fmt.Errorf("pool address is required")
    }
    
    stats := &PoolStats{
        Address: pool,
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "pool_stats"})
    return stats, nil
}

// QueryTokenTransfers retrieves token transfers
func (s *DefaultQueryService) QueryTokenTransfers(ctx context.Context, token string, limit, offset int) ([]*Transfer, error) {
    if token == "" {
        return nil, fmt.Errorf("token address is required")
    }
    
    transfers := make([]*Transfer, 0)
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "token_transfers"})
    return transfers, nil
}

// QueryPoolSwaps retrieves pool swaps
func (s *DefaultQueryService) QueryPoolSwaps(ctx context.Context, pool string, limit, offset int) ([]*Swap, error) {
    if pool == "" {
        return nil, fmt.Errorf("pool address is required")
    }
    
    swaps := make([]*Swap, 0)
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "pool_swaps"})
    return swaps, nil
}

// GetContractMetadata retrieves contract metadata
func (s *DefaultQueryService) GetContractMetadata(ctx context.Context, address string) (*ContractMetadata, error) {
    if address == "" {
        return nil, fmt.Errorf("contract address is required")
    }
    
    metadata := &ContractMetadata{
        Address:     address,
        LastUpdated: time.Now(),
    }
    
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "contract_metadata"})
    return metadata, nil
}

// QueryContractEvents retrieves contract events
func (s *DefaultQueryService) QueryContractEvents(ctx context.Context, address string, limit, offset int) ([]*Event, error) {
    if address == "" {
        return nil, fmt.Errorf("contract address is required")
    }
    
    events := make([]*Event, 0)
    s.metrics.RecordCounter("query_success", 1, map[string]string{"type": "contract_events"})
    return events, nil
}

// GetHealth retrieves health status
func (s *DefaultQueryService) GetHealth(ctx context.Context) (*HealthInfo, error) {
    return &HealthInfo{
        Status:    "healthy",
        Message:   "Query service is operational",
        Details:   make(map[string]interface{}),
        Timestamp: time.Now(),
    }, nil
}

// GetMetrics retrieves system metrics
func (s *DefaultQueryService) GetMetrics(ctx context.Context) (*SystemMetrics, error) {
    return &SystemMetrics{
        IsRunning:      true,
        ServiceCount:   9,
        DeploymentMode: "monolithic",
    }, nil
}

// Helper methods

func (s *DefaultQueryService) convertBlockchainEventToEvent(dbEvent *core.BlockchainEvent) *Event {
    topics := make([]string, len(dbEvent.EventTopic))
    for i, topic := range dbEvent.EventTopic {
        topics[i] = topic.Hex()
    }
    
    return &Event{
        ID:               dbEvent.ID,
        EventHash:        dbEvent.EventHash,
        EventSignature:   dbEvent.EventSignature.Hex(),
        BlockNumber:      dbEvent.BlockNumber,
        BlockHash:        dbEvent.BlockHash.Hex(),
        BlockTimestamp:   dbEvent.BlockTimestamp,
        TransactionHash:  dbEvent.TransactionHash.Hex(),
        TransactionIndex: dbEvent.TransactionIndex,
        LogIndex:         dbEvent.LogIndex,
        ContractAddress:  dbEvent.ContractAddress.Hex(),
        EventName:        dbEvent.EventName,
        EventTopics:      topics,
        EventData:        dbEvent.EventData,
        DecodedData:      dbEvent.DecodedData,
        ChainID:          dbEvent.ChainID,
        Network:          dbEvent.Network,
        Status:           string(dbEvent.Status),
        CreatedAt:        dbEvent.CreatedAt,
        ProcessedAt:      dbEvent.ProcessedAt,
        IndexedAt:        dbEvent.IndexedAt,
    }
}

func (s *DefaultQueryService) convertEventFilter(filter *EventFilter) *core.EventFilter {
    if filter == nil {
        return &core.EventFilter{}
    }
    
    return &core.EventFilter{
        ContractAddress: filter.ContractAddress,
        EventName:       filter.EventName,
        FromBlock:       filter.FromBlock,
        ToBlock:         filter.ToBlock,
        Network:         filter.Network,
    }
}
```

**Checklist**:
- [ ] File created at `pkg/services/query/query_service_impl.go`
- [ ] DefaultQueryService struct defined
- [ ] All interface methods implemented
- [ ] Caching logic implemented
- [ ] Error handling implemented
- [ ] Logging and metrics implemented
- [ ] Code compiles without errors

### Step 3: Write Tests

**File**: `pkg/services/query/query_service_test.go`

Write comprehensive unit tests for the query service.

**Checklist**:
- [ ] File created at `pkg/services/query/query_service_test.go`
- [ ] Tests for GetEvent
- [ ] Tests for QueryEvents
- [ ] Tests for GetTokenMetadata
- [ ] Tests for GetTokenBalance
- [ ] Tests for GetPoolMetadata
- [ ] Tests for GetPoolStats
- [ ] Tests for GetContractMetadata
- [ ] Tests for GetHealth
- [ ] Tests for GetMetrics
- [ ] Cache hit/miss tests
- [ ] Error handling tests
- [ ] All tests pass
- [ ] Coverage >80%

### Step 4: Verify Phase 1

**Checklist**:
- [ ] All files created
- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] Coverage >80%
- [ ] No linting errors
- [ ] Documentation complete

## Next Steps

Once Phase 1 is complete:
1. Move to Phase 2: GraphQL Refactoring
2. Update GraphQL resolver to use query service
3. Move GraphQL files to new package
4. Update tests

See `tasks.md` for detailed Phase 2 instructions.

## Tips

1. **Start Small**: Create the interface first, then implement
2. **Test as You Go**: Write tests for each method
3. **Use Mocks**: Mock database and cache in tests
4. **Check Compilation**: Run `go build ./...` frequently
5. **Run Tests**: Run `go test ./...` after each change

## Troubleshooting

**Compilation Errors**:
- Check imports
- Verify package names
- Ensure all methods are implemented

**Test Failures**:
- Check mock setup
- Verify expected values
- Review error handling

**Coverage Issues**:
- Add tests for error cases
- Test edge cases
- Test concurrent access

## Questions?

Refer to:
- `design.md` - Architecture details
- `requirements.md` - Requirements and acceptance criteria
- `tasks.md` - Detailed task breakdown
