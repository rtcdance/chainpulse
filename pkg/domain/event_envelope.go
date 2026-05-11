package domain

//go:generate mockgen -destination=mock_batch.go -package=domain . BatchProcessor,SharedBatchRuntime

import (
	"context"
	"time"

	"chainpulse/pkg/application/indexing"
)

// EventEnvelope is an alias for the application-layer EventEnvelope,
// allowing lower-level packages to depend on domain instead of application.
type EventEnvelope = indexing.EventEnvelope

// Checkpoint is an alias for the application-layer Checkpoint.
type Checkpoint = indexing.Checkpoint

// ProcessingFailure is an alias for the application-layer ProcessingFailure.
type ProcessingFailure = indexing.ProcessingFailure

// Ensure these type aliases are valid by referencing the original types.
var (
	_ = indexing.EventEnvelope{}
	_ = indexing.Checkpoint{}
	_ = func() indexing.ProcessingFailure { return indexing.ProcessingFailure{} }
)

// BatchProcessor processes a batch of event envelopes.
// This interface allows services to depend on the domain layer
// instead of directly on the application layer.
type BatchProcessor interface {
	ProcessBatch(ctx context.Context, chainID string, events []EventEnvelope) error
}

// SharedBatchRuntime is the service-layer interface for batch processing.
type SharedBatchRuntime interface {
	ProcessBatch(ctx context.Context, chainID string, events []EventEnvelope) error
}

// NewEventEnvelope creates an EventEnvelope from core types.
func NewEventEnvelope(eventKey, chainID, txHash string, blockNumber uint64, logIndex uint64, payload interface{}) EventEnvelope {
	return EventEnvelope{
		EventKey:        eventKey,
		ChainID:         chainID,
		BlockNumber:     blockNumber,
		TransactionHash: txHash,
		LogIndex:        logIndex,
		Payload:         payload,
		ReceivedAt:      time.Now(),
	}
}
