package query

//go:generate mockgen -destination=mock_event_store.go -package=query . EventMetadataStore

import (
	"context"
	"time"

	"chainpulse/pkg/core"
	domainquery "chainpulse/pkg/domain/query"
)

// EventStore is an alias for the canonical domain-layer EventStore interface.
// All consumers should use this type; the underlying definition lives in
// pkg/domain/query to maintain correct dependency direction.
type EventStore = domainquery.EventStore

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
	LogIndex         int64
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
