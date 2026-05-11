package query

//go:generate mockgen -destination=mock_event_store.go -package=query . EventStore

import (
	"context"

	"chainpulse/pkg/core"
)

// EventStore defines event retrieval contract at the domain boundary.
// This interface unifies read, write, and lifecycle operations.
//
// See also:
//   - pkg/services/query/event_store.go — service-layer EventStore (similar methods, with doc comments)
//   - pkg/infrastructure/processing/event_storage.go — infrastructure EventStore (uses local *Event type, different API)
//   - pkg/services/processor/event_processor.go — EventStorage (minimal write-only interface)
type EventStore interface {
	// Lifecycle
	Initialize(ctx context.Context) error
	Close(ctx context.Context) error

	// Write operations
	InsertEvent(ctx context.Context, event *core.BlockchainEvent) error
	InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error

	// Read operations
	GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error)
	GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error)
	GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error)
	GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error)
	GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error)
	GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error)
	GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error)
	GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error)
	CountEvents(ctx context.Context) (int64, error)

	// Maintenance
	DeleteExpiredEvents(ctx context.Context) (int64, error)
	Health(ctx context.Context) *core.HealthStatus
}
