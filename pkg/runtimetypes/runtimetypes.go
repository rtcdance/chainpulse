package runtimetypes

import "time"

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

type Checkpoint struct {
	ChainID     string
	Cursor      string
	BlockNumber uint64
	UpdatedAt   time.Time
}

type ProcessingFailure struct {
	EventKey   string
	ChainID    string
	Retryable  bool
	Reason     string
	OccurredAt time.Time
}

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
