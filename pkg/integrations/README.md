# Integrations Package

Third-party protocol and service integrations for blockchain indexing.

## Supported Protocols

### ERC-20 (`erc20/`)
Ethereum token standard indexing.

**Key Components**:
- **erc20_indexer.go** - ERC-20 token indexer
- **erc20_indexer_test.go** - Tests

**Features**:
- Token transfer tracking
- Balance queries
- Token metadata caching

### Uniswap (`uniswap/`)
Uniswap DEX integration.

**Key Components**:
- **uniswap_indexer.go** - Uniswap pool and swap indexer
- **uniswap_indexer_test.go** - Tests

**Features**:
- Pool tracking
- Swap event indexing
- Liquidity monitoring

### Generic (`generic/`)
Generic smart contract event indexing.

**Key Components**:
- **generic_indexer.go** - Generic contract indexer
- **generic_indexer_test.go** - Tests
- **generic_indexer_property_test.go** - Property-based tests

**Features**:
- Custom ABI support
- Event filtering
- Dynamic contract tracking

## Architecture

```
┌──────────────────────────────────────┐
│      Integration Registry            │
│  (Manages all integrations)          │
└──────────────┬───────────────────────┘
               │
       ┌───────┴────────────────────────┐
       │                                │
   ┌───▼────┐  ┌────────┐  ┌────────┐  │
   │ERC-20  │  │Uniswap │  │Generic │  │
   │Indexer │  │Indexer │  │Indexer │  │
   └────────┘  └────────┘  └────────┘  │
       │                                │
       └────────────────────────────────┘
               │
       ┌───────▼──────────────┐
       │  Multi-Chain Indexer │
       │  (Coordinates all)   │
       └──────────────────────┘
```

## Creating a Custom Integration

### 1. Define Integration Interface

```go
package myintegration

import (
    "context"
    "chainpulse/pkg/core"
)

type MyIntegration interface {
    // Index blockchain events
    Index(ctx context.Context, blockNumber uint64) error
    
    // Get indexed data
    GetData(ctx context.Context, query string) (interface{}, error)
    
    // Health check
    Health(ctx context.Context) error
}
```

### 2. Implement Integration

```go
type MyIndexer struct {
    config core.Config
    db     core.Database
    cache  core.Cache
}

func (i *MyIndexer) Index(ctx context.Context, blockNumber uint64) error {
    // Fetch events from blockchain
    // Parse and decode events
    // Store in database
    // Update cache
    return nil
}

func (i *MyIndexer) GetData(ctx context.Context, query string) (interface{}, error) {
    // Query indexed data
    return nil, nil
}

func (i *MyIndexer) Health(ctx context.Context) error {
    // Check health
    return nil
}
```

### 3. Register Integration

```go
registry := core.NewRegistry()
indexer := &MyIndexer{config: config, db: db, cache: cache}
registry.Register(indexer)
```

## Usage

Index ERC-20 tokens:
```go
import "chainpulse/pkg/integrations/erc20"

indexer := erc20.NewERC20Indexer(config, db, cache)
err := indexer.Index(ctx, blockNumber)
```

Index Uniswap pools:
```go
import "chainpulse/pkg/integrations/uniswap"

indexer := uniswap.NewUniswapIndexer(config, db, cache)
err := indexer.Index(ctx, blockNumber)
```

Index generic contracts:
```go
import "chainpulse/pkg/integrations/generic"

indexer := generic.NewGenericIndexer(config, db, cache)
err := indexer.Index(ctx, blockNumber)
```

## Configuration

Set environment variables:

```bash
# ERC-20 Integration
export ERC20_ENABLED=true
export ERC20_CACHE_TTL=3600

# Uniswap Integration
export UNISWAP_ENABLED=true
export UNISWAP_FACTORY_ADDRESS=0x1F98431c8aD98523631AE4a59f267346ea3113F
export UNISWAP_CACHE_TTL=3600

# Generic Integration
export GENERIC_ENABLED=true
export GENERIC_ABI_PATH=/etc/chainpulse/abis/
```

## Testing

Run integration tests:
```bash
go test ./pkg/integrations/...
```

Run specific integration tests:
```bash
go test ./pkg/integrations/erc20/...
go test ./pkg/integrations/uniswap/...
go test ./pkg/integrations/generic/...
```

## Best Practices

1. **Use standard interfaces** - Implement common indexer interface
2. **Cache results** - Use cache plugin for performance
3. **Handle errors** - Return meaningful errors
4. **Log operations** - Use structured logging
5. **Write tests** - Include unit and integration tests
6. **Document ABIs** - Include contract ABI files
7. **Version your integration** - Track version changes
8. **Support multiple chains** - Design for multi-chain support
