package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// defaultMaxIdempotencySize is the safety limit for in-memory dedup entries.
// At ~40 bytes per entry this is ~40MB for 1M entries.
const defaultMaxIdempotencySize = 1_000_000

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
	WarmUp(_ context.Context, hashes []string) error

	// GetProcessedCount returns the count of processed events
	GetProcessedCount() int64

	// GetDuplicateCount returns the count of duplicate events
	GetDuplicateCount() int64

	// Clear clears all processed hashes (for testing)
	Clear() error
}

// DefaultIdempotencyService provides default idempotency implementation.
// The in-memory store acts as a fast path with TTL-based eviction.
// Config provides:
//   - IdempotencyRecordTTL: how long a hash is kept in memory (default 24h)
//   - IdempotencyCleanupInterval: how often expired entries are purged (default 10m)
// A hard cap (defaultMaxIdempotencySize) prevents unbounded memory growth.
// Database-level unique constraints provide the ultimate dedup guarantee.
type DefaultIdempotencyService struct {
	mu               sync.RWMutex
	processedHashes  map[string]time.Time // hash → timestamp when marked
	processedCount   int64
	duplicateCount   int64
	initialized      bool
	running          bool
	lastHealthCheck  *core.HealthStatus
	config           *core.Config
	logger           core.Logger
	metricsCollector core.MetricsCollector
	stopCh           chan struct{}
	maxSize          int
}

// NewDefaultIdempotencyService creates a new idempotency service
func NewDefaultIdempotencyService(logger core.Logger, metricsCollector core.MetricsCollector) *DefaultIdempotencyService {
	return &DefaultIdempotencyService{
		processedHashes:  make(map[string]time.Time),
		logger:           logger,
		metricsCollector: metricsCollector,
		maxSize:          defaultMaxIdempotencySize,
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

	// Use config values if set, otherwise keep defaults
	if s.config.IdempotencyRecordTTL > 0 {
		// TTL is used by the cleanup goroutine
	}
	if s.config.IdempotencyCleanupInterval > 0 {
		// cleanup interval used by the cleanup goroutine
	}
	if s.config.IdempotencyRecordTTL <= 0 {
		s.config.IdempotencyRecordTTL = 86400 // 24h default
	}
	if s.config.IdempotencyCleanupInterval <= 0 {
		s.config.IdempotencyCleanupInterval = 600 // 10m default
	}

	s.initialized = true
	s.logger.Info("Idempotency service initialized",
		core.LogKeyComponent, "idempotency",
		"record_ttl_seconds", s.config.IdempotencyRecordTTL,
		"cleanup_interval_seconds", s.config.IdempotencyCleanupInterval,
		"max_size", s.maxSize,
	)

	return nil
}

// Start starts the idempotency service and launches the background cleanup goroutine.
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
	s.stopCh = make(chan struct{})

	s.lastHealthCheck = &core.HealthStatus{
		Status:  "healthy",
		Message: "Idempotency service started",
	}

	// Start background cleanup goroutine
	if s.config != nil && s.config.IdempotencyCleanupInterval > 0 {
		cleanupInterval := time.Duration(s.config.IdempotencyCleanupInterval) * time.Second
		go s.cleanupLoop(cleanupInterval)
		s.logger.Info("Idempotency cleanup goroutine started",
			"cleanup_interval", cleanupInterval.String(),
			"record_ttl", time.Duration(s.config.IdempotencyRecordTTL)*time.Second,
		)
	}

	s.logger.Info("Idempotency service started", core.LogKeyComponent, "idempotency")

	return nil
}

// Stop stops the idempotency service and the background cleanup goroutine.
func (s *DefaultIdempotencyService) Stop() error {
	s.mu.Lock()

	if !s.running {
		s.mu.Unlock()
		return fmt.Errorf("idempotency service not running")
	}

	s.running = false
	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.mu.Unlock()

	s.logger.Info("Idempotency service stopped",
		core.LogKeyComponent, "idempotency",
		"total_stored", len(s.processedHashes),
	)
	return nil
}

// cleanupLoop periodically evicts expired entries from the in-memory map.
func (s *DefaultIdempotencyService) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.evictExpired()
		case <-s.stopCh:
			return
		}
	}
}

// evictExpired removes entries that have exceeded the TTL.
// Must NOT be called with the lock held.
func (s *DefaultIdempotencyService) evictExpired() {
	ttl := time.Duration(s.config.IdempotencyRecordTTL) * time.Second
	cutoff := time.Now().Add(-ttl)

	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.processedHashes)
	for hash, addedAt := range s.processedHashes {
		if addedAt.Before(cutoff) {
			delete(s.processedHashes, hash)
		}
	}

	evicted := before - len(s.processedHashes)
	if evicted > 0 {
		s.logger.Debug("idempotency expired entries evicted",
			"evicted", evicted,
			"remaining", len(s.processedHashes),
			"ttl_seconds", s.config.IdempotencyRecordTTL,
		)
		s.metricsCollector.RecordGauge("idempotency_evicted_count", float64(evicted), nil)
		s.metricsCollector.RecordGauge("idempotency_stored_count", float64(len(s.processedHashes)), nil)
	}
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

	_, exists := s.processedHashes[hash]
	if !exists {
		return false, nil
	}

	s.metricsCollector.RecordCounter("idempotency_duplicate_detected", 1, map[string]string{})

	return true, nil
}

// MarkProcessed marks an event as processed. If the in-memory map has reached
// the maxSize limit, the oldest entry is evicted to make room (approximate LRU).
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
	if _, exists := s.processedHashes[hash]; exists {
		s.duplicateCount++
		s.metricsCollector.RecordCounter("idempotency_duplicate_marked", 1, map[string]string{})
		return nil
	}

	// Evict oldest entry if at capacity (safety limit)
	if len(s.processedHashes) >= s.maxSize {
		var oldestHash string
		var oldestTime time.Time
		first := true
		for h, t := range s.processedHashes {
			if first || t.Before(oldestTime) {
				oldestHash = h
				oldestTime = t
				first = false
			}
		}
		delete(s.processedHashes, oldestHash)
		s.metricsCollector.RecordCounter("idempotency_max_size_eviction", 1, map[string]string{})
	}

	s.processedHashes[hash] = time.Now()
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
	now := time.Now()
	for _, hash := range hashes {
		if hash == "" {
			continue
		}
		if _, exists := s.processedHashes[hash]; !exists {
			s.processedHashes[hash] = now
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

	s.processedHashes = make(map[string]time.Time)
	s.processedCount = 0
	s.duplicateCount = 0

	s.metricsCollector.RecordCounter("idempotency_cleared", 1, map[string]string{})

	return nil
}
