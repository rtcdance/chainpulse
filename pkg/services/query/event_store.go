package query

import (
	"context"
	"time"

	"chainpulse/pkg/core"
)

// EventStore defines the interface for storing and retrieving events from MongoDB
type EventStore interface {
	// Initialize initializes the event store
	Initialize(ctx context.Context) error

	// InsertEvent inserts a single event into the store
	InsertEvent(ctx context.Context, event *core.BlockchainEvent) error

	// InsertEventBatch inserts multiple events in a batch operation
	InsertEventBatch(ctx context.Context, events []*core.BlockchainEvent) error

	// GetEvent retrieves a single event by ID
	GetEvent(ctx context.Context, eventID string) (*core.BlockchainEvent, error)

	// GetEventsByChain retrieves events for a specific chain
	GetEventsByChain(ctx context.Context, chainID int, limit int, offset int) ([]*core.BlockchainEvent, error)

	// GetEventsByContract retrieves events for a specific contract
	GetEventsByContract(ctx context.Context, contractAddress string, limit int, offset int) ([]*core.BlockchainEvent, error)

	// GetEventsByEventName retrieves events by event name
	GetEventsByEventName(ctx context.Context, eventName string, limit int, offset int) ([]*core.BlockchainEvent, error)

	// GetEventsByBlock retrieves events by block number
	GetEventsByBlock(ctx context.Context, blockNumber int64) ([]*core.BlockchainEvent, error)

	// GetEventsByAddress retrieves events by contract address with limit
	GetEventsByAddress(ctx context.Context, address string, limit int) ([]*core.BlockchainEvent, error)

	// GetEventsByName retrieves events by event name with limit
	GetEventsByName(ctx context.Context, eventName string, limit int) ([]*core.BlockchainEvent, error)

	// GetEventsPaginated retrieves events with cursor-based pagination
	GetEventsPaginated(ctx context.Context, cursor string, limit int) ([]*core.BlockchainEvent, bool, error)

	// DeleteExpiredEvents deletes events that have exceeded their TTL
	DeleteExpiredEvents(ctx context.Context) (int64, error)

	// Health returns the health status of the event store
	Health(ctx context.Context) *core.HealthStatus

	// Close closes the event store
	Close(ctx context.Context) error
}

// EventMetadataStore defines the interface for storing event metadata in PostgreSQL
type EventMetadataStore interface {
	// Initialize initializes the metadata store
	Initialize(ctx context.Context) error

	// InsertMetadata inserts a single event metadata record
	InsertMetadata(ctx context.Context, metadata *EventMetadata) error

	// InsertMetadataBatch inserts multiple metadata records in a batch operation
	InsertMetadataBatch(ctx context.Context, metadataList []*EventMetadata) error

	// GetMetadata retrieves metadata for a single event
	GetMetadata(ctx context.Context, eventID string) (*EventMetadata, error)

	// GetMetadataByChain retrieves metadata for events in a specific chain
	GetMetadataByChain(ctx context.Context, chainID int, limit int, offset int) ([]*EventMetadata, error)

	// UpdateMetadata updates metadata for an event
	UpdateMetadata(ctx context.Context, metadata *EventMetadata) error

	// Health returns the health status of the metadata store
	Health(ctx context.Context) *core.HealthStatus

	// Close closes the metadata store
	Close(ctx context.Context) error
}

// EventMetadata represents metadata about a processed event
type EventMetadata struct {
	ID               int64
	EventID          string
	ChainID          int
	BlockNumber      int64
	TransactionHash  string
	LogIndex         int
	ContractAddress  string
	EventName        string
	ProcessedAt      time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	ProcessingStatus string // pending, processing, completed, failed
	ProcessingError  string
	RetryCount       int
	LastRetryAt      *time.Time
}

// EventStoreConfig holds configuration for the event store
type EventStoreConfig struct {
	// MongoDB collection name
	CollectionName string

	// TTL for events in days (0 = no TTL)
	TTLDays int

	// Batch size for bulk operations
	BatchSize int

	// Index creation timeout
	IndexTimeout time.Duration
}

// DefaultEventStoreConfig returns default configuration for event store
func DefaultEventStoreConfig() *EventStoreConfig {
	return &EventStoreConfig{
		CollectionName: "events",
		TTLDays:        30,
		BatchSize:      100,
		IndexTimeout:   10 * time.Second,
	}
}
