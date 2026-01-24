# GraphQL API Refactoring - Design

## Architecture Decision

**Status**: CONFIRMED - Option A (GraphQL as True Plugin)

This specification implements GraphQL as a proper plugin where:
- **GraphQL API** implements the `APIPlugin` interface fully
- **Query Service** provides unified data access layer
- **Plugin Lifecycle** is properly managed (Initialize, Start, Stop, Health, GetStats)
- **HTTP Handler** adapts between HTTP and APIRequest/APIResponse types
- **Decoupling** from direct database/cache/indexer dependencies

**Rationale**: GraphQL should be a first-class plugin like other API implementations in the system.

## Architecture Overview

```
┌─────────────────────────────────────┐
│      HTTP Requests                   │
└──────────────┬──────────────────────┘
               │
               ▼
        ┌──────────────────┐
        │ HTTP Adapter     │
        │ (converts HTTP   │
        │  to APIRequest)  │
        └────────┬─────────┘
                 │
                 ▼
        ┌──────────────────┐
        │ GraphQL Plugin   │
        │ (implements      │
        │  APIPlugin)      │
        └────────┬─────────┘
                 │
                 ▼
        ┌──────────────────┐
        │ Query Service    │
        │ (Unified Layer)  │
        │ Decouples from   │
        │ indexers         │
        └────────┬─────────┘
                 │
        ┌────────┴────────┐
        │                 │
        ▼                 ▼
    ┌────────┐        ┌────────┐
    │Database│        │ Cache  │
    └────────┘        └────────┘
```

## Current State vs. Target State

### Current Issues (Before Refactoring)

| Issue | Impact |
|-------|--------|
| GraphQL not implementing APIPlugin | No plugin lifecycle management |
| GraphQL directly coupled to indexers | Hard to test, maintain, extend |
| No unified query layer | Code duplication, inconsistent caching |
| HTTP handler signature mismatch | Can't use standard plugin interface |
| Package structure inconsistency | APIPlugin in core package, not api |

### Target State (After Refactoring)

| Improvement | Benefit |
|-------------|---------|
| GraphQL implements APIPlugin | Proper plugin lifecycle, consistent interface |
| Query Service decouples APIs | Easier testing, maintenance, extension |
| Unified query layer | Single source of truth, consistent caching |
| HTTP adapter layer | Bridges HTTP and plugin interfaces |
| Correct package structure | APIPlugin in api package where it belongs |

## Package Structure

```
pkg/
├── plugins/
│   └── api/
│       ├── api_plugin.go           # API plugin interface
│       ├── graphql_plugin.go       # GraphQL plugin (NEW - implements APIPlugin)
│       ├── http_adapter.go         # HTTP to APIRequest adapter (NEW)
│       ├── graphql_server.go       # GraphQL server (REFACTORED)
│       ├── graphql_resolver.go     # GraphQL resolver (REFACTORED)
│       ├── graphql_schema.go       # GraphQL schema
│       ├── graphql_plugin_test.go  # Plugin tests (NEW)
│       ├── graphql_server_test.go  # Server tests
│       └── graphql_resolver_test.go # Resolver tests
│
└── services/
    └── query/                      # New package
        ├── query_service.go        # Query service interface
        ├── query_service_impl.go   # Default implementation
        ├── types.go                # Data types
        ├── query_service_test.go   # Unit tests
        └── query_service_property_test.go # Property tests
```

## Core Interfaces

### Query Service Interface

```go
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
```

### Data Types

```go
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

// TokenMetadata represents ERC20 token metadata
type TokenMetadata struct {
    Address      string
    Name         string
    Symbol       string
    Decimals     int
    TotalSupply  *big.Int
    LastUpdated  time.Time
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

// ... other types
```

## Implementation Details

### GraphQL Plugin Implementation

**File**: `pkg/plugins/api/graphql_plugin.go`

```go
type GraphQLPlugin struct {
    *core.BaseAPIPlugin
    queryService core.QueryService
    server       *GraphQLServer
    logger       core.Logger
    metrics      core.MetricsCollector
}

func NewGraphQLPlugin(
    queryService core.QueryService,
    logger core.Logger,
    metrics core.MetricsCollector,
) *GraphQLPlugin {
    return &GraphQLPlugin{
        BaseAPIPlugin: core.NewBaseAPIPlugin(logger, metrics),
        queryService:  queryService,
        logger:        logger,
        metrics:       metrics,
    }
}

// Implements APIPlugin interface
func (p *GraphQLPlugin) Initialize(config *core.Config) error {
    // Initialize base plugin
    if err := p.BaseAPIPlugin.Initialize(config); err != nil {
        return err
    }
    
    // Create GraphQL resolver with query service
    resolver := NewGraphQLResolver(p.queryService, p.logger, p.metrics)
    
    // Create GraphQL server
    p.server = NewGraphQLServer(resolver, p.logger)
    
    return nil
}

func (p *GraphQLPlugin) Start() error {
    if err := p.BaseAPIPlugin.Start(); err != nil {
        return err
    }
    return p.server.Start()
}

func (p *GraphQLPlugin) Stop() error {
    if err := p.server.Stop(); err != nil {
        return err
    }
    return p.BaseAPIPlugin.Stop()
}

func (p *GraphQLPlugin) Health() *core.HealthStatus {
    return p.BaseAPIPlugin.Health()
}

// HandleRequest converts HTTP request to GraphQL query
func (p *GraphQLPlugin) HandleRequest(request *core.APIRequest) (*core.APIResponse, error) {
    start := time.Now()
    p.RecordRequest()
    
    // Parse GraphQL request from API request body
    var gqlReq GraphQLRequest
    if err := json.Unmarshal(request.Body, &gqlReq); err != nil {
        p.RecordError()
        return &core.APIResponse{
            StatusCode:  400,
            Body:        []byte(`{"errors":[{"message":"Invalid JSON"}]}`),
            ContentType: "application/json",
            Timestamp:   time.Now(),
        }, nil
    }
    
    // Execute GraphQL query
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    response := p.server.executeQuery(ctx, &gqlReq)
    
    // Convert to API response
    body, _ := json.Marshal(response)
    statusCode := 200
    if len(response.Errors) > 0 {
        statusCode = 400
    }
    
    duration := time.Since(start).Milliseconds()
    p.RecordResponse(statusCode, duration)
    
    return &core.APIResponse{
        StatusCode:  statusCode,
        Body:        body,
        ContentType: "application/json",
        Timestamp:   time.Now(),
        Duration:    duration,
    }, nil
}

func (p *GraphQLPlugin) GetStats() *core.APIStats {
    return p.BaseAPIPlugin.GetStats()
}
```

### HTTP Adapter

**File**: `pkg/plugins/api/http_adapter.go`

```go
type HTTPAdapter struct {
    plugin core.APIPlugin
    logger core.Logger
}

func NewHTTPAdapter(plugin core.APIPlugin, logger core.Logger) *HTTPAdapter {
    return &HTTPAdapter{
        plugin: plugin,
        logger: logger,
    }
}

// HandleHTTP converts HTTP request to APIRequest and calls plugin
func (a *HTTPAdapter) HandleHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Failed to read request body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()
    
    // Convert HTTP request to APIRequest
    apiReq := &core.APIRequest{
        Method:    r.Method,
        Path:      r.URL.Path,
        Query:     r.URL.Query().Map(),
        Body:      body,
        Headers:   r.Header,
        ClientIP:  r.RemoteAddr,
        Timestamp: time.Now(),
    }
    
    // Call plugin
    apiResp, err := a.plugin.HandleRequest(apiReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Write HTTP response
    w.Header().Set("Content-Type", apiResp.ContentType)
    w.WriteHeader(apiResp.StatusCode)
    w.Write(apiResp.Body)
}
```

### GraphQL Resolver Refactoring

**File**: `pkg/plugins/api/graphql_resolver.go`

```go
type GraphQLResolver struct {
    queryService core.QueryService  // Use query service instead of database
    logger       core.Logger
    metrics      core.MetricsCollector
    subscriptions map[string]chan interface{}
    subscriptionsMu sync.RWMutex
}

func NewGraphQLResolver(
    queryService core.QueryService,
    logger core.Logger,
    metrics core.MetricsCollector,
) *GraphQLResolver {
    return &GraphQLResolver{
        queryService:  queryService,
        logger:        logger,
        metrics:       metrics,
        subscriptions: make(map[string]chan interface{}),
    }
}

// QueryEvent uses query service
func (r *GraphQLResolver) QueryEvent(ctx context.Context, id string) (*EventNode, error) {
    event, err := r.queryService.GetEvent(ctx, id)
    if err != nil {
        return nil, err
    }
    return r.eventToNode(event), nil
}

// ... other methods
```

## Data Flow

### GraphQL Query Flow (as Plugin)

```
HTTP POST /graphql
    ↓
HTTP Adapter
    ↓
Convert to APIRequest
    ↓
GraphQL Plugin (HandleRequest)
    ↓
Parse GraphQL Query
    ↓
GraphQL Resolver
    ↓
Query Service (GetEvent, QueryEvents, etc.)
    ↓
Cache Check
    ├─ Hit → Return cached data
    └─ Miss → Query Database
        ↓
    Database Query
        ↓
    Cache Result
        ↓
    Return to Resolver
        ↓
    Convert to GraphQL Node
        ↓
    Convert to APIResponse
        ↓
    HTTP Adapter converts to HTTP Response
        ↓
Return HTTP Response
```

### Plugin Lifecycle

```
System Startup
    ↓
Plugin Registry
    ↓
GraphQL Plugin.Initialize(config)
    ├─ Create Query Service
    ├─ Create GraphQL Resolver
    └─ Create GraphQL Server
    ↓
GraphQL Plugin.Start()
    ├─ Start GraphQL Server
    └─ Ready to handle requests
    ↓
Handle Requests
    ├─ HTTP Adapter receives HTTP request
    ├─ Converts to APIRequest
    ├─ Plugin.HandleRequest(APIRequest)
    └─ Returns APIResponse
    ↓
GraphQL Plugin.Stop()
    ├─ Stop GraphQL Server
    └─ Cleanup resources
    ↓
System Shutdown
```

## Testing Strategy

### Unit Tests
- Query service methods with mocked database/cache
- GraphQL resolver with mocked query service
- REST plugin with mocked query service
- HTTP server routing

### Integration Tests
- Query service with real database/cache
- GraphQL with real query service
- REST with real query service
- End-to-end HTTP requests

### Property Tests
- Query service caching behavior
- Pagination logic
- Error handling
- Concurrent access

## Migration Path

1. **Create query service** - No breaking changes
2. **Update GraphQL** - Backward compatible
3. **Create REST plugin** - New functionality
4. **Integrate HTTP server** - Unified endpoint
5. **Deprecate old API** - Gradual migration

## Performance Considerations

- **Caching**: 5-minute TTL for metadata, 1-minute for events
- **Pagination**: Default 20 items, max 100
- **Concurrency**: Thread-safe with RWMutex
- **Metrics**: Track cache hits/misses, query times

## Error Handling

- Validation errors → 400 Bad Request
- Not found errors → 404 Not Found
- Internal errors → 500 Internal Server Error
- Rate limit errors → 429 Too Many Requests

## Backward Compatibility

- Existing GraphQL queries remain unchanged
- New REST API is additive
- No breaking changes to data structures
- Gradual migration path for clients
