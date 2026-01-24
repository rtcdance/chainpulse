# Services Package

Business logic services for event processing, querying, and blockchain integration.

## Modules

### Decoder
- **contract_manager.go** - Smart contract ABI management and caching
- **event_decoder.go** - Event log decoding and parsing

### Query Service
- **query_service.go** - Protocol-agnostic query interface for events, tokens, pools, and contracts
- **database_indexes.sql** - Database index definitions for query optimization

### Indexing
- **chain_indexer.go** - Single-chain indexer implementation
- **multi_chain_indexer.go** - Multi-chain indexing coordinator

### Consistency
- **consistency_checker.go** - Data consistency verification and repair

### Reorganization
- **reorg_handler.go** - Blockchain reorganization (reorg) handling

## Architecture

```
┌──────────────────────────────────────┐
│      Query Service Interface         │
│  (Events, Tokens, Pools, Contracts)  │
└──────────────┬───────────────────────┘
               │
       ┌───────▼────────────┐
       │  Multi-Chain       │
       │  Indexer           │
       └───────┬────────────┘
               │
       ┌───────┴────────────┐
       │                    │
   ┌───▼────┐          ┌───▼────┐
   │Chain   │          │Chain   │
   │Indexer │          │Indexer │
   │(ETH)   │          │(Polygon)
   └───┬────┘          └───┬────┘
       │                    │
       └───────┬────────────┘
               │
       ┌───────▼──────────────┐
       │  Event Decoder       │
       │  (Contract Manager)  │
       └───────┬──────────────┘
               │
       ┌───────▼──────────────┐
       │  Consistency Checker │
       │  & Reorg Handler     │
       └──────────────────────┘
```

## Key Interfaces

### QueryService
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
}
```

### ChainIndexer
```go
type ChainIndexer interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    GetCurrentBlockHeight() uint64
    GetSyncStatus() SyncStatus
}
```

## Usage

Query events:
```go
import "chainpulse/pkg/services/query"

// Get a specific event
event, err := queryService.GetEvent(ctx, "event-id")

// Query events with filter
filter := &query.EventFilter{
    ContractAddress: "0x...",
    EventName: "Transfer",
}
events, err := queryService.QueryEvents(ctx, filter, 100, 0)
```

Start indexing:
```go
import "chainpulse/pkg/services/indexing"

indexer := indexing.NewMultiChainIndexer(config)
err := indexer.Start(ctx)
defer indexer.Stop(ctx)
```

## Configuration

Set environment variables:
```bash
# Indexing
export CHAINS=ethereum,polygon
export BLOCKCHAIN_NODE_URLS=http://localhost:8545,http://localhost:8546
export START_BLOCK=0

# Query Service
export QUERY_CACHE_TTL=3600
export QUERY_BATCH_SIZE=100

# Consistency
export CONSISTENCY_CHECK_INTERVAL=3600
export CONSISTENCY_REPAIR_ENABLED=true
```

## Testing

Run tests:
```bash
go test ./pkg/services/...
```

Run integration tests:
```bash
go test ./test/integration/...
```
