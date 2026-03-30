---
name: "architecture-patterns"
description: "DDD, CQRS, and Event Sourcing patterns for enterprise Web3 applications. Invoke when designing system architecture or implementing core patterns."
---

# Architecture Patterns Guidelines

## Purpose
Ensure consistent application of enterprise architecture patterns.

## When to Invoke
- Designing system architecture
- Implementing core business logic
- Creating new services or modules
- Refactoring existing architecture

## DDD Layered Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Presentation Layer                        │
│              (API, gRPC, WebSocket Handlers)                 │
├─────────────────────────────────────────────────────────────┤
│                    Application Layer                         │
│              (Use Cases, Orchestration)                      │
├─────────────────────────────────────────────────────────────┤
│                      Domain Layer                            │
│         (Entities, Value Objects, Domain Services)           │
├─────────────────────────────────────────────────────────────┤
│                   Infrastructure Layer                       │
│        (Repositories, External Services, Adapters)           │
└─────────────────────────────────────────────────────────────┘
```

### Domain Layer (pkg/core/)

```go
package core

type Block struct {
    number      uint64
    hash        common.Hash
    parentHash  common.Hash
    timestamp   time.Time
    events      []*Event
}

func NewBlock(number uint64, hash common.Hash) *Block {
    return &Block{
        number: number,
        hash:   hash,
    }
}

func (b *Block) AddEvent(event *Event) error {
    if event.BlockNumber() != b.number {
        return ErrEventBlockMismatch
    }
    b.events = append(b.events, event)
    return nil
}
```

### Application Layer (pkg/services/)

```go
type IndexerUseCase struct {
    blockRepo   BlockRepository
    eventRepo   EventRepository
    puller      BlockPuller
}

func (uc *IndexerUseCase) IndexBlock(ctx context.Context, blockNum uint64) error {
    block, err := uc.puller.Pull(ctx, blockNum)
    if err != nil {
        return fmt.Errorf("pull block %d: %w", blockNum, err)
    }
    
    if err := uc.blockRepo.Save(ctx, block); err != nil {
        return fmt.Errorf("save block: %w", err)
    }
    
    for _, event := range block.Events() {
        if err := uc.eventRepo.Save(ctx, event); err != nil {
            return fmt.Errorf("save event: %w", err)
        }
    }
    
    return nil
}
```

## CQRS Pattern

```go
type CommandHandler interface {
    Handle(ctx context.Context, cmd Command) error
}

type QueryHandler interface {
    Handle(ctx context.Context, query Query) (Result, error)
}

type IndexBlockCommand struct {
    BlockNumber uint64
    ChainID     *big.Int
}

type GetEventsQuery struct {
    Contract   common.Address
    FromBlock  uint64
    ToBlock    uint64
    Limit      int
}
```

## Event Sourcing

```go
type EventStore interface {
    Append(ctx context.Context, aggregateID string, events []DomainEvent) error
    Load(ctx context.Context, aggregateID string) ([]DomainEvent, error)
}

type DomainEvent interface {
    AggregateID() string
    EventType() string
    OccurredAt() time.Time
}

type BlockIndexedEvent struct {
    blockNumber uint64
    chainID     *big.Int
    occurredAt  time.Time
}

func (e *BlockIndexedEvent) AggregateID() string {
    return fmt.Sprintf("chain-%s-blocks", e.chainID.String())
}

func (e *BlockIndexedEvent) EventType() string {
    return "block.indexed"
}
```

## Repository Pattern

```go
type BlockRepository interface {
    Save(ctx context.Context, block *Block) error
    FindByNumber(ctx context.Context, chainID *big.Int, number uint64) (*Block, error)
    FindLatest(ctx context.Context, chainID *big.Int) (*Block, error)
}

type EventRepository interface {
    Save(ctx context.Context, event *Event) error
    FindByTxHash(ctx context.Context, txHash common.Hash) (*Event, error)
    FindByContract(ctx context.Context, addr common.Address, opts QueryOptions) ([]*Event, error)
}
```

## Plugin Architecture

```go
type Plugin interface {
    Name() string
    Init(config Config) error
    Start(ctx context.Context) error
    Stop() error
}

type PluginRegistry struct {
    plugins map[string]Plugin
}

func (r *PluginRegistry) Register(plugin Plugin) {
    r.plugins[plugin.Name()] = plugin
}

func (r *PluginRegistry) Get(name string) (Plugin, bool) {
    p, ok := r.plugins[name]
    return p, ok
}
```

## Constraints

- ALWAYS keep domain logic in domain layer
- ALWAYS use interfaces for external dependencies
- NEVER let infrastructure concerns leak into domain
- NEVER bypass repository interfaces
- ALWAYS use value objects for identifiers
- ALWAYS design for testability
