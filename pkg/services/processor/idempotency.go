package processor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"

	"chainpulse/pkg/core"
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
	IsDuplicate(hash string) (bool, error)

	// MarkProcessed marks an event as processed
	MarkProcessed(hash string) error

	// GetProcessedCount returns the count of processed events
	GetProcessedCount() int64

	// GetDuplicateCount returns the count of duplicate events
	GetDuplicateCount() int64

	// Clear clears all processed hashes (for testing)
	Clear() error
}

// DefaultIdempotencyService provides default idempotency implementation
type DefaultIdempotencyService struct {
	mu               sync.RWMutex
	processedHashes  map[string]bool
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

	s.logger.Info("Idempotency service initialized", map[string]interface{}{
		"component": "idempotency",
	})

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

	s.logger.Info("Idempotency service started", map[string]interface{}{
		"component": "idempotency",
	})

	return nil
}

// Stop stops the idempotency service
func (s *DefaultIdempotencyService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("idempotency service not running")
	}

	s.running = false
	s.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Idempotency service stopped",
	}

	s.logger.Info("Idempotency service stopped", map[string]interface{}{
		"component": "idempotency",
	})

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

// GenerateHash generates a deterministic hash for an event
func (s *DefaultIdempotencyService) GenerateHash(event *core.BlockchainEvent) (string, error) {
	if event == nil {
		return "", fmt.Errorf("event is required")
	}

	// Create a deterministic string representation of the event
	hashInput := fmt.Sprintf("%s:%d:%s:%d:%s",
		event.Network,
		event.BlockNumber,
		event.TransactionHash.Hex(),
		event.LogIndex,
		event.ContractAddress.Hex(),
	)

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(hashInput))
	hashStr := hex.EncodeToString(hash[:])

	s.metricsCollector.RecordCounter("idempotency_hash_generated", 1, map[string]string{
		"network": event.Network,
	})

	return hashStr, nil
}

// IsDuplicate checks if an event has been processed before
func (s *DefaultIdempotencyService) IsDuplicate(hash string) (bool, error) {
	if hash == "" {
		return false, fmt.Errorf("hash is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running {
		return false, fmt.Errorf("idempotency service not running")
	}

	isDuplicate := s.processedHashes[hash]

	if isDuplicate {
		s.metricsCollector.RecordCounter("idempotency_duplicate_detected", 1, map[string]string{})
	}

	return isDuplicate, nil
}

// MarkProcessed marks an event as processed
func (s *DefaultIdempotencyService) MarkProcessed(hash string) error {
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
