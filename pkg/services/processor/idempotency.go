package processor

import (
	"context"
	"fmt"
	"sync"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// IdempotencyService provides duplicate detection for events
type IdempotencyService interface {
	// Initialize initializes the idempotency service
	Initialize(config *core.Config) error

	// Start starts the idempotency service
	Start() error

	// Stop stops the idempotency service
	Stop() error

	// Health returns the health status of the service
	Health() *core.HealthStatus

	// GenerateHash generates a deterministic hash for an event
	GenerateHash(event *core.BlockchainEvent) (string, error)

	// IsDuplicate checks if an event has been processed before
	IsDuplicate(ctx context.Context, hash string) (bool, error)

	// MarkProcessed marks an event as processed
	MarkProcessed(ctx context.Context, hash string) error

	// WarmUp pre-populates the dedup store from persisted state to
	// minimize duplicate-key errors after a restart.
	WarmUp(ctx context.Context, hashes []string) error

	// GetProcessedCount returns the count of processed events
	GetProcessedCount() int64

	// GetDuplicateCount returns the count of duplicate events
	GetDuplicateCount() int64

	// Clear clears all processed hashes (for testing)
	Clear() error
}

// DefaultIdempotencyService provides default idempotency implementation.
// The in-memory store acts as a fast path that never evicts records —
// blockchain events are permanently unique, so the natural key
// (chain_id, block_number, tx_hash, log_index) will never be seen again.
// Database-level unique constraints provide the ultimate dedup guarantee.
type DefaultIdempotencyService struct {
	mu               sync.RWMutex
	processedHashes  map[string]bool // hash → true (no TTL, no timestamps)
	processedCount   int64
	duplicateCount   int64
	initialized      bool
	running          bool
	lastHealthCheck  *core.HealthStatus
	config           *core.Config
	logger           core.Logger
	metricsCollector core.MetricsCollector
}

// NewDefaultIdempotencyService creates a new idempotency service
func NewDefaultIdempotencyService(logger core.Logger, metricsCollector core.MetricsCollector) *DefaultIdempotencyService {
	return &DefaultIdempotencyService{
		processedHashes:  make(map[string]bool),
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the idempotency service
func (s *DefaultIdempotencyService) Initialize(config *core.Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return fmt.Errorf("idempotency service already initialized")
	}

	if config == nil {
		return fmt.Errorf("config is required")
	}

	s.config = config
	s.initialized = true

	s.logger.Info("Idempotency service initialized", core.LogKeyComponent, "idempotency")

	return nil
}

// Start starts the idempotency service
func (s *DefaultIdempotencyService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return fmt.Errorf("idempotency service not initialized")
	}

	if s.running {
		return fmt.Errorf("idempotency service already running")
	}

	s.running = true

	s.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Idempotency service started",
	}

	s.logger.Info("Idempotency service started", core.LogKeyComponent, "idempotency")

	return nil
}

// Stop stops the idempotency service
func (s *DefaultIdempotencyService) Stop() error {
	s.mu.Lock()

	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("idempotency service not running")
	}

	s.running = false
	s.mu.Unlock()

	s.logger.Info("Idempotency service stopped", core.LogKeyComponent, "idempotency")

	return nil
}

// Health returns the health status of the service
func (s *DefaultIdempotencyService) Health() *core.HealthStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Idempotency service not initialized",
		}
	}

	if !s.running {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Idempotency service not running",
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "Idempotency service healthy",
	}
}

// GenerateHash generates a deterministic hash for an event using the
// canonical ComputeEventHash function from pkg/core.
func (s *DefaultIdempotencyService) GenerateHash(event *core.BlockchainEvent) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is required")
	}

	hash := core.ComputeEventHash(event)

	s.metricsCollector.RecordCounter("idempotency_hash_generated", 1, map[string]string{
		"network": event.Network,
	})

	return hash, nil
}

// IsDuplicate checks if an event has been processed before
func (s *DefaultIdempotencyService) IsDuplicate(ctx context.Context, hash string) (bool, error) {
	if hash == "" {
		return false, fmt.Errorf("hash is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return false, fmt.Errorf("idempotency service not running")
	}

	exists := s.processedHashes[hash]
	if !exists {
		return false, nil
	}

	s.metricsCollector.RecordCounter("idempotency_duplicate_detected", 1, map[string]string{})

	return true, nil
}

// MarkProcessed marks an event as processed
func (s *DefaultIdempotencyService) MarkProcessed(ctx context.Context, hash string) error {
	if hash == "" {
		return fmt.Errorf("hash is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("idempotency service not running")
	}

	// Check if already processed
	if s.processedHashes[hash] {
		s.duplicateCount++
		s.metricsCollector.RecordCounter("idempotency_duplicate_marked", 1, map[string]string{})
		return nil
	}

	s.processedHashes[hash] = true
	s.processedCount++

	s.metricsCollector.RecordCounter("idempotency_event_marked", 1, map[string]string{})
	s.metricsCollector.RecordGauge("idempotency_processed_count", float64(s.processedCount), map[string]string{})
	s.metricsCollector.RecordGauge("idempotency_stored_count", float64(len(s.processedHashes)), map[string]string{})

	return nil
}

// WarmUp pre-populates the in-memory dedup store to minimize duplicate-key
// errors after a restart. Only hashes that are not yet tracked are added.
func (s *DefaultIdempotencyService) WarmUp(_ context.Context, hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("idempotency service not running")
	}

	added := 0
	for _, hash := range hashes {
		if hash == "" {
			continue
		}
		if !s.processedHashes[hash] {
			s.processedHashes[hash] = true
			added++
		}
	}

	if added > 0 {
		s.logger.Info("idempotency warm-up complete",
			"total_loaded", len(hashes),
			"newly_added", added,
			"total_stored", len(s.processedHashes))
		s.metricsCollector.RecordCounter("idempotency_warmup_loaded", int64(added), nil)
	}

	return nil
}

// GetProcessedCount returns the count of processed events
func (s *DefaultIdempotencyService) GetProcessedCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.processedCount
}

// GetDuplicateCount returns the count of duplicate events
func (s *DefaultIdempotencyService) GetDuplicateCount() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.duplicateCount
}

// Clear clears all processed hashes (for testing)
func (s *DefaultIdempotencyService) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("idempotency service not running")
	}

	s.processedHashes = make(map[string]bool)
	s.processedCount = 0
	s.duplicateCount = 0

	s.metricsCollector.RecordCounter("idempotency_cleared", 1, map[string]string{})

	return nil
}
