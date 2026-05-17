package domain

//go:generate mockgen -destination=mock_batch.go -package=domain . BatchProcessor,SharedBatchRuntime

import (
	"context"
	"time"

	"chainpulse/pkg/core"
)

// EventEnvelope is an alias for the core EventEnvelope.
// Defined here so domain-layer interfaces (BatchProcessor, SharedBatchRuntime)
// can reference it without depending on the application layer.
type EventEnvelope = core.EventEnvelope

// Checkpoint is an alias for the core Checkpoint.
type Checkpoint = core.Checkpoint

// ProcessingFailure is an alias for the core ProcessingFailure.
type ProcessingFailure = core.ProcessingFailure

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
func NewEventEnvelope(eventKey, chainID, txHash string, blockNumber uint64, logIndex uint64, payload any) EventEnvelope {
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
