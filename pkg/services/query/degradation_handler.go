package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"
)

// DegradationMode represents the current degradation state
type DegradationMode int

const (
	// Normal operation - all stores available
	DegradationModeNormal DegradationMode = iota
	// MongoDB unavailable - use cache and PostgreSQL
	DegradationModeMongoDBAnavailable
	// PostgreSQL unavailable - use MongoDB only
	DegradationModePostgreSQLUnavailable
	// Both stores unavailable - use cache only
	DegradationModeBothUnavailable
	// Cache unavailable - use stores only
	DegradationModeCacheUnavailable
	// All stores unavailable - read-only mode
	DegradationModeReadOnly
)

// String returns the string representation of DegradationMode
func (m DegradationMode) String() string {
	switch m {
	case DegradationModeNormal:
		return "Normal"
	case DegradationModeMongoDBAnavailable:
		return "MongoDBUnavailable"
	case DegradationModePostgreSQLUnavailable:
		return "PostgreSQLUnavailable"
	case DegradationModeBothUnavailable:
		return "BothUnavailable"
	case DegradationModeCacheUnavailable:
		return "CacheUnavailable"
	case DegradationModeReadOnly:
		return "ReadOnly"
	default:
		return "Unknown"
	}
}

// FallbackStrategy defines how to handle degraded operations
type FallbackStrategy interface {
	// Name returns the strategy name
	Name() string
	// CanRetrieveEvent checks if event can be retrieved
	CanRetrieveEvent(ctx context.Context) bool
	// CanRetrieveMetadata checks if metadata can be retrieved
	CanRetrieveMetadata(ctx context.Context) bool
	// CanWrite checks if writes are supported
	CanWrite(ctx context.Context) bool
}

// DegradationHandler manages graceful degradation
type DegradationHandler interface {
	// Initialize initializes the degradation handler
	Initialize(ctx context.Context) error
	// GetDegradationMode returns the current degradation mode
	GetDegradationMode(ctx context.Context) DegradationMode
	// CanUseMongoDB checks if MongoDB is available
	CanUseMongoDB(ctx context.Context) bool
	// CanUsePostgreSQL checks if PostgreSQL is available
	CanUsePostgreSQL(ctx context.Context) bool
	// CanUseCache checks if cache is available
	CanUseCache(ctx context.Context) bool
	// SelectStrategy selects the best fallback strategy
	SelectStrategy(ctx context.Context) FallbackStrategy
	// RecordDegradation records a degradation event
	RecordDegradation(ctx context.Context, mode DegradationMode, reason string)
	// Health returns the health status
	Health(ctx context.Context) *core.HealthStatus
	// Close closes the degradation handler
	Close(ctx context.Context) error
}

// DefaultDegradationHandler implements DegradationHandler
type DefaultDegradationHandler struct {
	mu                sync.RWMutex
	eventStore        EventStore
	metadataStore     EventMetadataStore
	cacheService      CacheService
	logger            core.Logger
	metrics           core.MetricsCollector
	currentMode       DegradationMode
	lastModeChange    time.Time
	degradationEvents []DegradationEvent
	initialized       bool
}

// DegradationEvent represents a degradation event
type DegradationEvent struct {
	Timestamp time.Time
	Mode      DegradationMode
	Reason    string
}

// NewDegradationHandler creates a new degradation handler
func NewDegradationHandler(
	eventStore EventStore,
	metadataStore EventMetadataStore,
	cacheService CacheService,
	logger core.Logger,
	metrics core.MetricsCollector,
) DegradationHandler {
	return &DefaultDegradationHandler{
		eventStore:        eventStore,
		metadataStore:     metadataStore,
		cacheService:      cacheService,
		logger:            logger,
		metrics:           metrics,
		currentMode:       DegradationModeNormal,
		degradationEvents: make([]DegradationEvent, 0),
		initialized:       false,
	}
}

// Initialize initializes the degradation handler
func (h *DefaultDegradationHandler) Initialize(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.initialized {
		return nil
	}

	if h.eventStore == nil {
		return fmt.Errorf("event store is required")
	}

	if h.metadataStore == nil {
		return fmt.Errorf("metadata store is required")
	}

	if h.cacheService == nil {
		return fmt.Errorf("cache service is required")
	}

	h.initialized = true
	h.lastModeChange = time.Now()
	h.logger.Info("Degradation handler initialized")

	return nil
}

// GetDegradationMode returns the current degradation mode
func (h *DefaultDegradationHandler) GetDegradationMode(ctx context.Context) DegradationMode {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		return DegradationModeReadOnly
	}

	// Check store availability
	mongoAvailable := h.canUseMongoDB(ctx)
	postgresAvailable := h.canUsePostgreSQL(ctx)
	cacheAvailable := h.canUseCache(ctx)

	// Determine mode based on availability
	if mongoAvailable && postgresAvailable && cacheAvailable {
		return DegradationModeNormal
	}

	if !mongoAvailable && postgresAvailable && cacheAvailable {
		return DegradationModeMongoDBAnavailable
	}

	if mongoAvailable && !postgresAvailable && cacheAvailable {
		return DegradationModePostgreSQLUnavailable
	}

	if !mongoAvailable && !postgresAvailable && cacheAvailable {
		return DegradationModeBothUnavailable
	}

	if mongoAvailable && postgresAvailable && !cacheAvailable {
		return DegradationModeCacheUnavailable
	}

	return DegradationModeReadOnly
}

// CanUseMongoDB checks if MongoDB is available
func (h *DefaultDegradationHandler) CanUseMongoDB(ctx context.Context) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.canUseMongoDB(ctx)
}

// canUseMongoDB is the internal version without locking
func (h *DefaultDegradationHandler) canUseMongoDB(ctx context.Context) bool {
	if h.eventStore == nil {
		return false
	}

	health := h.eventStore.Health(ctx)
	return health != nil && health.Status == "healthy"
}

// CanUsePostgreSQL checks if PostgreSQL is available
func (h *DefaultDegradationHandler) CanUsePostgreSQL(ctx context.Context) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.canUsePostgreSQL(ctx)
}

// canUsePostgreSQL is the internal version without locking
func (h *DefaultDegradationHandler) canUsePostgreSQL(ctx context.Context) bool {
	if h.metadataStore == nil {
		return false
	}

	health := h.metadataStore.Health(ctx)
	return health != nil && health.Status == "healthy"
}

// CanUseCache checks if cache is available
func (h *DefaultDegradationHandler) CanUseCache(ctx context.Context) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.canUseCache(ctx)
}

// canUseCache is the internal version without locking
func (h *DefaultDegradationHandler) canUseCache(ctx context.Context) bool {
	if h.cacheService == nil {
		return false
	}

	health := h.cacheService.Health(ctx)
	return health != nil && health.Status == "healthy"
}

// SelectStrategy selects the best fallback strategy
func (h *DefaultDegradationHandler) SelectStrategy(ctx context.Context) FallbackStrategy {
	h.mu.RLock()
	defer h.mu.RUnlock()

	mode := h.GetDegradationMode(ctx)

	switch mode {
	case DegradationModeNormal:
		return NewHybridStrategy(h.eventStore, h.metadataStore, h.cacheService)
	case DegradationModeMongoDBAnavailable:
		return NewPostgreSQLOnlyStrategy(h.metadataStore, h.cacheService)
	case DegradationModePostgreSQLUnavailable:
		return NewMongoDBOnlyStrategy(h.eventStore)
	case DegradationModeBothUnavailable:
		return NewCacheOnlyStrategy(h.cacheService)
	case DegradationModeCacheUnavailable:
		return NewHybridStrategy(h.eventStore, h.metadataStore, nil)
	case DegradationModeReadOnly:
		return NewReadOnlyStrategy()
	default:
		return NewReadOnlyStrategy()
	}
}

// RecordDegradation records a degradation event
func (h *DefaultDegradationHandler) RecordDegradation(ctx context.Context, mode DegradationMode, reason string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	event := DegradationEvent{
		Timestamp: time.Now(),
		Mode:      mode,
		Reason:    reason,
	}

	h.degradationEvents = append(h.degradationEvents, event)

	// Keep only last 100 events
	if len(h.degradationEvents) > 100 {
		h.degradationEvents = h.degradationEvents[len(h.degradationEvents)-100:]
	}

	// Record metrics
	h.metrics.RecordCounter("degradation_mode_changes_total", 1, nil)
	h.metrics.RecordCounter(fmt.Sprintf("degradation_%s_total", mode.String()), 1, nil)

	h.logger.Warn("Degradation event recorded", "mode", mode.String(), "reason", reason)
}

// Health returns the health status
func (h *DefaultDegradationHandler) Health(ctx context.Context) *core.HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "degradation handler not initialized",
		}
	}

	mode := h.GetDegradationMode(ctx)

	if mode == DegradationModeReadOnly {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "all stores unavailable",
		}
	}

	if mode != DegradationModeNormal {
		return &core.HealthStatus{
			Status:  "degraded",
			Message: fmt.Sprintf("degradation mode: %s", mode.String()),
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "all systems operational",
	}
}

// Close closes the degradation handler
func (h *DefaultDegradationHandler) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.initialized = false
	return nil
}

// Fallback Strategy Implementations

// CacheOnlyStrategy retrieves from cache only
type CacheOnlyStrategy struct {
	cacheService CacheService
}

// NewCacheOnlyStrategy creates a new cache-only strategy
func NewCacheOnlyStrategy(cacheService CacheService) FallbackStrategy {
	return &CacheOnlyStrategy{
		cacheService: cacheService,
	}
}

// Name returns the strategy name
func (s *CacheOnlyStrategy) Name() string {
	return "CacheOnly"
}

// CanRetrieveEvent checks if event can be retrieved
func (s *CacheOnlyStrategy) CanRetrieveEvent(ctx context.Context) bool {
	return s.cacheService != nil
}

// CanRetrieveMetadata checks if metadata can be retrieved
func (s *CacheOnlyStrategy) CanRetrieveMetadata(ctx context.Context) bool {
	return false
}

// CanWrite checks if writes are supported
func (s *CacheOnlyStrategy) CanWrite(ctx context.Context) bool {
	return false
}

// MongoDBOnlyStrategy retrieves from MongoDB only
type MongoDBOnlyStrategy struct {
	eventStore EventStore
}

// NewMongoDBOnlyStrategy creates a new MongoDB-only strategy
func NewMongoDBOnlyStrategy(eventStore EventStore) FallbackStrategy {
	return &MongoDBOnlyStrategy{
		eventStore: eventStore,
	}
}

// Name returns the strategy name
func (s *MongoDBOnlyStrategy) Name() string {
	return "MongoDBOnly"
}

// CanRetrieveEvent checks if event can be retrieved
func (s *MongoDBOnlyStrategy) CanRetrieveEvent(ctx context.Context) bool {
	return s.eventStore != nil
}

// CanRetrieveMetadata checks if metadata can be retrieved
func (s *MongoDBOnlyStrategy) CanRetrieveMetadata(ctx context.Context) bool {
	return false
}

// CanWrite checks if writes are supported
func (s *MongoDBOnlyStrategy) CanWrite(ctx context.Context) bool {
	return s.eventStore != nil
}

// PostgreSQLOnlyStrategy retrieves from PostgreSQL only
type PostgreSQLOnlyStrategy struct {
	metadataStore EventMetadataStore
	cacheService  CacheService
}

// NewPostgreSQLOnlyStrategy creates a new PostgreSQL-only strategy
func NewPostgreSQLOnlyStrategy(metadataStore EventMetadataStore, cacheService CacheService) FallbackStrategy {
	return &PostgreSQLOnlyStrategy{
		metadataStore: metadataStore,
		cacheService:  cacheService,
	}
}

// Name returns the strategy name
func (s *PostgreSQLOnlyStrategy) Name() string {
	return "PostgreSQLOnly"
}

// CanRetrieveEvent checks if event can be retrieved
func (s *PostgreSQLOnlyStrategy) CanRetrieveEvent(ctx context.Context) bool {
	return s.cacheService != nil
}

// CanRetrieveMetadata checks if metadata can be retrieved
func (s *PostgreSQLOnlyStrategy) CanRetrieveMetadata(ctx context.Context) bool {
	return s.metadataStore != nil
}

// CanWrite checks if writes are supported
func (s *PostgreSQLOnlyStrategy) CanWrite(ctx context.Context) bool {
	return s.metadataStore != nil
}

// HybridStrategy uses available stores
type HybridStrategy struct {
	eventStore    EventStore
	metadataStore EventMetadataStore
	cacheService  CacheService
}

// NewHybridStrategy creates a new hybrid strategy
func NewHybridStrategy(eventStore EventStore, metadataStore EventMetadataStore, cacheService CacheService) FallbackStrategy {
	return &HybridStrategy{
		eventStore:    eventStore,
		metadataStore: metadataStore,
		cacheService:  cacheService,
	}
}

// Name returns the strategy name
func (s *HybridStrategy) Name() string {
	return "Hybrid"
}

// CanRetrieveEvent checks if event can be retrieved
func (s *HybridStrategy) CanRetrieveEvent(ctx context.Context) bool {
	return s.eventStore != nil || s.cacheService != nil
}

// CanRetrieveMetadata checks if metadata can be retrieved
func (s *HybridStrategy) CanRetrieveMetadata(ctx context.Context) bool {
	return s.metadataStore != nil
}

// CanWrite checks if writes are supported
func (s *HybridStrategy) CanWrite(ctx context.Context) bool {
	return s.eventStore != nil || s.metadataStore != nil
}

// ReadOnlyStrategy allows no operations
type ReadOnlyStrategy struct{}

// NewReadOnlyStrategy creates a new read-only strategy
func NewReadOnlyStrategy() FallbackStrategy {
	return &ReadOnlyStrategy{}
}

// Name returns the strategy name
func (s *ReadOnlyStrategy) Name() string {
	return "ReadOnly"
}

// CanRetrieveEvent checks if event can be retrieved
func (s *ReadOnlyStrategy) CanRetrieveEvent(ctx context.Context) bool {
	return false
}

// CanRetrieveMetadata checks if metadata can be retrieved
func (s *ReadOnlyStrategy) CanRetrieveMetadata(ctx context.Context) bool {
	return false
}

// CanWrite checks if writes are supported
func (s *ReadOnlyStrategy) CanWrite(ctx context.Context) bool {
	return false
}
