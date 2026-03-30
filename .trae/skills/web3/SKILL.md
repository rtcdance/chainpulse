---
name: "web3"
description: "Web3 blockchain development best practices. Invoke when working with Ethereum, blockchain data, or smart contracts."
---

# Web3 Development Guidelines

## Purpose
Ensure best practices for Web3 blockchain development.

## When to Invoke
- Working with Ethereum/blockchain data
- Implementing event indexing
- Handling blockchain reorganizations
- Processing smart contract events

## Blockchain Data Handling

### Block Processing
```go
type BlockProcessor struct {
    client     *ethclient.Client
    chainID    *big.Int
    confirmations uint64
}

func (p *BlockProcessor) ProcessBlock(ctx context.Context, blockNum uint64) error {
    header, err := p.client.HeaderByNumber(ctx, big.NewInt(int64(blockNum)))
    if err != nil {
        return fmt.Errorf("failed to get block %d: %w", blockNum, err)
    }
    
    if !p.isFinalized(header) {
        return ErrBlockNotFinalized
    }
    
    return p.processFinalizedBlock(ctx, header)
}
```

### Reorg Handling
```go
type ReorgHandler struct {
    db          Database
    confirmations uint64
}

func (h *ReorgHandler) CheckReorg(ctx context.Context, blockNum uint64) (bool, error) {
    storedHash, err := h.db.GetBlockHash(blockNum)
    if err != nil {
        return false, err
    }
    
    currentHash, err := h.getCurrentHash(ctx, blockNum)
    if err != nil {
        return false, err
    }
    
    return storedHash != currentHash, nil
}

func (h *ReorgHandler) Rollback(ctx context.Context, fromBlock uint64) error {
    return h.db.RollbackEvents(ctx, fromBlock)
}
```

### Event Filtering
```go
func (p *Puller) FilterEvents(
    ctx context.Context,
    contractABI abi.ABI,
    addresses []common.Address,
    fromBlock, toBlock uint64,
) ([]types.Log, error) {
    query := ethereum.FilterQuery{
        FromBlock: big.NewInt(int64(fromBlock)),
        ToBlock:   big.NewInt(int64(toBlock)),
        Addresses: addresses,
    }
    
    logs, err := p.client.FilterLogs(ctx, query)
    if err != nil {
        return nil, fmt.Errorf("failed to filter logs: %w", err)
    }
    
    return logs, nil
}
```

## Idempotency

```go
type IdempotentProcessor struct {
    db Database
}

func (p *IdempotentProcessor) ProcessEvent(event *Event) error {
    eventID := p.generateEventID(event)
    
    if processed, _ := p.db.IsProcessed(eventID); processed {
        return nil
    }
    
    if err := p.db.StoreEvent(event); err != nil {
        return err
    }
    
    return p.db.MarkProcessed(eventID)
}

func (p *IdempotentProcessor) generateEventID(event *Event) string {
    return fmt.Sprintf("%s-%d-%d", 
        event.TxHash.Hex(),
        event.BlockNumber,
        event.LogIndex,
    )
}
```

## Multi-Chain Support

```go
type ChainConfig struct {
    ChainID        *big.Int
    Name           string
    RPCURL         string
    Confirmations  uint64
    BlockTime      time.Duration
}

var ChainConfigs = map[string]ChainConfig{
    "ethereum": {
        ChainID:       big.NewInt(1),
        Confirmations: 12,
        BlockTime:     12 * time.Second,
    },
    "polygon": {
        ChainID:       big.NewInt(137),
        Confirmations: 128,
        BlockTime:     2 * time.Second,
    },
}
```

## Error Handling

```go
var (
    ErrBlockNotFinalized = errors.New("block not finalized")
    ErrReorgDetected     = errors.New("reorg detected")
    ErrRPCConnection     = errors.New("RPC connection failed")
)

func (p *Puller) withRetry(ctx context.Context, fn func() error) error {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        if err := fn(); err != nil {
            if isReorgError(err) {
                return ErrReorgDetected
            }
            lastErr = err
            time.Sleep(backoff(i))
            continue
        }
        return nil
    }
    return lastErr
}
```

## Constraints
- ALWAYS handle reorg scenarios
- ALWAYS implement idempotency
- ALWAYS use proper confirmation depth
- NEVER assume blocks are final
- NEVER skip error handling for RPC calls
- ALWAYS support multiple chains
