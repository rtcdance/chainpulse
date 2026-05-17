// Package core defines shared interfaces, data models, and type definitions
// for the ChainPulse event indexing system.
//
// All cross-package shared types live here to prevent circular dependencies
// between domain, application, and infrastructure layers.
package core

import "time"

// EventEnvelope is the shared unit of work passed through indexing runtime
// stages before persistence and downstream publication.
//
// Defined in core rather than application/indexing so that domain-layer
// interfaces (e.g., BatchProcessor) can reference it without depending on
// the application layer — preserving clean DDD layering.
type EventEnvelope struct {
	EventKey         string
	ChainID          string
	BlockNumber      uint64
	TransactionHash  string
	LogIndex         uint64
	Payload          any
	ReceivedAt       time.Time
	CheckpointCursor string
}

// Checkpoint captures chain progress for restart/replay aware runtimes.
type Checkpoint struct {
	ChainID     string
	Cursor      string
	BlockNumber uint64
	UpdatedAt   time.Time
}

// ProcessingFailure classifies an event handling failure and the action taken.
type ProcessingFailure struct {
	EventKey   string
	ChainID    string
	Retryable  bool
	Reason     string
	OccurredAt time.Time
}

// RuntimeStatus reports additive indexing runtime health and counters.
// Moved from pkg/application/indexing to core so that cmd/runtime_summary
// can reference it without depending on the application layer.
type RuntimeStatus struct {
	State                   string
	Initialized             bool
	Started                 bool
	Chains                  []string
	CheckpointingEnabled    bool
	IdempotencyEnabled      bool
	FailureRoutingEnabled   bool
	ReplayEnabled           bool
	ProcessedEvents         int64
	SkippedDuplicates       int64
	RoutedFailures          int64
	RecoveryState           string
	RecoveryRuns            int64
	RecoveryFailures        int64
	RecoveryCheckpointLoads int64
	RecoveryReplayedEvents  int64
	LastRecoveryChainID     string
	LastRecoveryCursor      string
	LastRecoveryBlock       uint64
	LastRecoveryReplayCount int64
	LastRecoveryError       string
	LastRecoveryAt          time.Time
	LastCheckpointChainID   string
	LastCheckpointCursor    string
	LastCheckpointBlock     uint64
	LastUpdatedAt           time.Time
}
