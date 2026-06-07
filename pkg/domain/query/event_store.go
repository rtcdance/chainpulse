package query

//go:generate mockgen -destination=mock_event_store.go -package=query . EventStore,EventWriter,EventReader

import (
	"context"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// EventWriter defines write-only contract for event persistence at the domain boundary.
type EventWriter interface {
	Initialize(ctx context.Context) error
	Close(ctx context.Context) error
	InsertEvent(ctx context.Context, event *blockchain.BlockchainEvent) error
	InsertEventBatch(ctx context.Context, events []*blockchain.BlockchainEvent) error
	DeleteExpiredEvents(ctx context.Context) (int64, error)
	Health(ctx context.Context) *core.HealthStatus
}

// EventReader defines read-only contract for event retrieval at the domain boundary.
type EventReader interface {
	GetEvent(ctx context.Context, eventID string) (*blockchain.BlockchainEvent, error)
	GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*blockchain.BlockchainEvent, error)
	GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*blockchain.BlockchainEvent, error)
	GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*blockchain.BlockchainEvent, error)
	GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*blockchain.BlockchainEvent, error)
	GetEventsByAddress(ctx context.Context, address string, limit int) ([]*blockchain.BlockchainEvent, error)
	GetEventsByName(ctx context.Context, eventName string, limit int) ([]*blockchain.BlockchainEvent, error)
	GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*blockchain.BlockchainEvent, bool, error)
	// GetEventsByCorrelationID returns events across all chains that share a
	// correlation ID, enabling cross-chain event correlation for bridge
	// transfers, multi-chain contract interactions, and other linked events.
	GetEventsByCorrelationID(ctx context.Context, correlationID string, limit int, offset int) ([]*blockchain.BlockchainEvent, error)
	CountEvents(ctx context.Context) (int64, error)
	// GetEventStats returns aggregated event counts by chain and by event name,
	// plus the total reorged count. Uses MongoDB aggregation for efficiency.
	GetEventStats(ctx context.Context) (byChain map[string]int64, byEventName map[string]int64, reorged int64, err error)
	Health(ctx context.Context) *core.HealthStatus
}

// EventStore defines the combined event read/write contract at the domain boundary.
// For consumers that only need one direction, prefer EventWriter or EventReader.
//
// See also:
//   - pkg/services/query/event_store.go — service-layer EventStore (similar methods, with doc comments)
//   - pkg/infrastructure/processing/event_storage.go — infrastructure EventStore (uses local *Event type, different API)
//   - pkg/services/processor/event_processor.go — EventStorage (minimal write-only interface)
type EventStore interface {
	EventWriter
	EventReader
}
