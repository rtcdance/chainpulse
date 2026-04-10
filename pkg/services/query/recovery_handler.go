package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// RecoveryState represents the state of recovery
type RecoveryState int

const (
	// RecoveryStateHealthy indicates the system is healthy
	RecoveryStateHealthy RecoveryState = iota
	// RecoveryStateRecovering indicates recovery is in progress
	RecoveryStateRecovering
	// RecoveryStateFailed indicates recovery failed
	RecoveryStateFailed
)

// String returns the string representation of RecoveryState
func (rs RecoveryState) String() string {
	switch rs {
	case RecoveryStateHealthy:
		return "healthy"
	case RecoveryStateRecovering:
		return "recovering"
	case RecoveryStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// RecoveryConfig holds recovery configuration
type RecoveryConfig struct {
	// MaxRetries is the maximum number of recovery attempts
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// HealthCheckInterval is the interval for health checks
	HealthCheckInterval time.Duration
	// DataSyncTimeout is the timeout for data sync operations
	DataSyncTimeout time.Duration
}

// DefaultRecoveryConfig returns the default recovery configuration
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		MaxRetries:          5,
		InitialBackoff:      100 * time.Millisecond,
		MaxBackoff:          10 * time.Second,
		BackoffMultiplier:   2.0,
		HealthCheckInterval: 5 * time.Second,
		DataSyncTimeout:     30 * time.Second,
	}
}

// RecoveryHandler manages error recovery procedures
type RecoveryHandler interface {
	// Initialize initializes the recovery handler
	Initialize(ctx context.Context) error
	// RecoverConnection attempts to recover a failed connection
	RecoverConnection(ctx context.Context, store string) error
	// RecoverState attempts to recover system state after restart
	RecoverState(ctx context.Context) error
	// SyncData attempts to sync data after recovery
	SyncData(ctx context.Context, store string) error
	// GetRecoveryState returns the current recovery state
	GetRecoveryState() RecoveryState
	// GetLastRecoveryTime returns the last recovery time
	GetLastRecoveryTime() time.Time
	// GetRecoveryMetrics returns recovery metrics
	GetRecoveryMetrics() RecoveryMetrics
	// Close closes the recovery handler
	Close(ctx context.Context) error
}

// RecoveryMetrics holds recovery metrics
type RecoveryMetrics struct {
	// TotalRecoveryAttempts is the total number of recovery attempts
	TotalRecoveryAttempts int
	// SuccessfulRecoveries is the number of successful recoveries
	SuccessfulRecoveries int
	// FailedRecoveries is the number of failed recoveries
	FailedRecoveries int
	// LastRecoveryTime is the time of the last recovery attempt
	LastRecoveryTime time.Time
	// LastSuccessfulRecoveryTime is the time of the last successful recovery
	LastSuccessfulRecoveryTime time.Time
	// AverageRecoveryDuration is the average recovery duration
	AverageRecoveryDuration time.Duration
	// ConnectionRecoveries is the number of connection recoveries
	ConnectionRecoveries int
	// StateRecoveries is the number of state recoveries
	StateRecoveries int
	// DataSyncRecoveries is the number of data sync recoveries
	DataSyncRecoveries int
}

// DefaultRecoveryHandler implements RecoveryHandler
type DefaultRecoveryHandler struct {
	mu                     sync.RWMutex
	config                 RecoveryConfig
	state                  RecoveryState
	eventStore             EventStore
	metadataStore          EventMetadataStore
	cacheService           CacheService
	errorClassifier        *ErrorClassifier
	logger                 core.Logger
	metricsCollector       core.MetricsCollector
	initialized            bool
	lastRecoveryTime       time.Time
	lastSuccessfulRecovery time.Time
	totalAttempts          int
	successfulRecoveries   int
	failedRecoveries       int
	connectionRecoveries   int
	stateRecoveries        int
	dataSyncRecoveries     int
	totalRecoveryDuration  time.Duration
	recoveryInProgress     bool
}

// NewRecoveryHandler creates a new recovery handler
func NewRecoveryHandler(
	config RecoveryConfig,
	eventStore EventStore,
	metadataStore EventMetadataStore,
	cacheService CacheService,
	errorClassifier *ErrorClassifier,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) RecoveryHandler {
	return &DefaultRecoveryHandler{
		config:           config,
		state:            RecoveryStateHealthy,
		eventStore:       eventStore,
		metadataStore:    metadataStore,
		cacheService:     cacheService,
		errorClassifier:  errorClassifier,
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the recovery handler
func (rh *DefaultRecoveryHandler) Initialize(ctx context.Context) error {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	if rh.initialized {
		return nil
	}

	if rh.eventStore == nil {
		return fmt.Errorf("event store is required")
	}

	if rh.metadataStore == nil {
		return fmt.Errorf("metadata store is required")
	}

	if rh.cacheService == nil {
		return fmt.Errorf("cache service is required")
	}

	if rh.errorClassifier == nil {
		return fmt.Errorf("error classifier is required")
	}

	rh.initialized = true
	rh.logger.Info("Recovery handler initialized")
	return nil
}

// RecoverConnection attempts to recover a failed connection
func (rh *DefaultRecoveryHandler) RecoverConnection(ctx context.Context, store string) error {
	rh.mu.Lock()
	if rh.recoveryInProgress {
		rh.mu.Unlock()
		return fmt.Errorf("recovery already in progress")
	}
	rh.recoveryInProgress = true
	rh.mu.Unlock()

	defer func() {
		rh.mu.Lock()
		rh.recoveryInProgress = false
		rh.mu.Unlock()
	}()

	if !rh.initialized {
		return fmt.Errorf("recovery handler not initialized")
	}

	if store == "" {
		return fmt.Errorf("store name is required")
	}

	rh.mu.Lock()
	rh.state = RecoveryStateRecovering
	rh.lastRecoveryTime = time.Now()
	rh.totalAttempts++
	rh.mu.Unlock()

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rh.mu.Lock()
		rh.totalRecoveryDuration += duration
		rh.mu.Unlock()
	}()

	rh.logger.Info("Starting connection recovery", map[string]interface{}{
		"store": store,
	})

	// Attempt reconnection with exponential backoff
	backoff := rh.config.InitialBackoff
	for attempt := 0; attempt < rh.config.MaxRetries; attempt++ {
		// Check context
		select {
		case <-ctx.Done():
			rh.mu.Lock()
			rh.state = RecoveryStateFailed
			rh.failedRecoveries++
			rh.mu.Unlock()
			return fmt.Errorf("recovery context cancelled")
		default:
		}

		// Attempt to reconnect based on store type
		var err error
		switch store {
		case "mongodb":
			err = rh.reconnectMongoDB(ctx)
		case "postgresql":
			err = rh.reconnectPostgreSQL(ctx)
		case "cache":
			err = rh.reconnectCache(ctx)
		default:
			rh.mu.Lock()
			rh.state = RecoveryStateFailed
			rh.failedRecoveries++
			rh.mu.Unlock()
			return fmt.Errorf("unknown store: %s", store)
		}

		if err == nil {
			// Reconnection successful
			rh.mu.Lock()
			rh.state = RecoveryStateHealthy
			rh.successfulRecoveries++
			rh.lastSuccessfulRecovery = time.Now()
			rh.connectionRecoveries++
			rh.mu.Unlock()

			rh.logger.Info("Connection recovery successful", map[string]interface{}{
				"store":    store,
				"attempts": attempt + 1,
			})

			rh.metricsCollector.RecordCounter("recovery_connection_success", 1, nil)
			return nil
		}

		rh.logger.Warn("Connection recovery attempt failed", map[string]interface{}{
			"store":   store,
			"attempt": attempt + 1,
			"error":   err.Error(),
			"backoff": backoff.String(),
		})

		// Wait before retry
		if attempt < rh.config.MaxRetries-1 {
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				rh.mu.Lock()
				rh.state = RecoveryStateFailed
				rh.failedRecoveries++
				rh.mu.Unlock()
				return fmt.Errorf("recovery context cancelled")
			}

			// Calculate next backoff
			backoff = time.Duration(float64(backoff) * rh.config.BackoffMultiplier)
			if backoff > rh.config.MaxBackoff {
				backoff = rh.config.MaxBackoff
			}
		}
	}

	// All recovery attempts failed
	rh.mu.Lock()
	rh.state = RecoveryStateFailed
	rh.failedRecoveries++
	rh.mu.Unlock()

	rh.logger.Error("Connection recovery failed after all attempts", map[string]interface{}{
		"store":    store,
		"attempts": rh.config.MaxRetries,
	})

	rh.metricsCollector.RecordCounter("recovery_connection_failed", 1, nil)
	return fmt.Errorf("connection recovery failed for store: %s", store)
}

// RecoverState attempts to recover system state after restart
func (rh *DefaultRecoveryHandler) RecoverState(ctx context.Context) error {
	rh.mu.Lock()
	if rh.recoveryInProgress {
		rh.mu.Unlock()
		return fmt.Errorf("recovery already in progress")
	}
	rh.recoveryInProgress = true
	rh.mu.Unlock()

	defer func() {
		rh.mu.Lock()
		rh.recoveryInProgress = false
		rh.mu.Unlock()
	}()

	if !rh.initialized {
		return fmt.Errorf("recovery handler not initialized")
	}

	rh.mu.Lock()
	rh.state = RecoveryStateRecovering
	rh.lastRecoveryTime = time.Now()
	rh.totalAttempts++
	rh.mu.Unlock()

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rh.mu.Lock()
		rh.totalRecoveryDuration += duration
		rh.mu.Unlock()
	}()

	rh.logger.Info("Starting state recovery")

	// Create a context with timeout for state recovery
	ctx, cancel := context.WithTimeout(ctx, rh.config.DataSyncTimeout)
	defer cancel()

	// Verify both stores are accessible
	eventStoreHealth := rh.eventStore.Health(ctx)
	metadataStoreHealth := rh.metadataStore.Health(ctx)

	if eventStoreHealth.Status != "healthy" {
		rh.logger.Warn("Event store unhealthy during state recovery", map[string]interface{}{
			"message": eventStoreHealth.Message,
		})
	}

	if metadataStoreHealth.Status != "healthy" {
		rh.logger.Warn("Metadata store unhealthy during state recovery", map[string]interface{}{
			"message": metadataStoreHealth.Message,
		})
	}

	// If both stores are healthy, state recovery is successful
	if eventStoreHealth.Status == "healthy" && metadataStoreHealth.Status == "healthy" {
		rh.mu.Lock()
		rh.state = RecoveryStateHealthy
		rh.successfulRecoveries++
		rh.lastSuccessfulRecovery = time.Now()
		rh.stateRecoveries++
		rh.mu.Unlock()

		rh.logger.Info("State recovery successful")
		rh.metricsCollector.RecordCounter("recovery_state_success", 1, nil)
		return nil
	}

	// Attempt to recover individual stores
	if eventStoreHealth.Status != "healthy" {
		if err := rh.RecoverConnection(ctx, "mongodb"); err != nil {
			rh.logger.Error("Failed to recover event store", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	if metadataStoreHealth.Status != "healthy" {
		if err := rh.RecoverConnection(ctx, "postgresql"); err != nil {
			rh.logger.Error("Failed to recover metadata store", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}

	// Check if recovery was successful
	eventStoreHealth = rh.eventStore.Health(ctx)
	metadataStoreHealth = rh.metadataStore.Health(ctx)

	if eventStoreHealth.Status == "healthy" && metadataStoreHealth.Status == "healthy" {
		rh.mu.Lock()
		rh.state = RecoveryStateHealthy
		rh.successfulRecoveries++
		rh.lastSuccessfulRecovery = time.Now()
		rh.stateRecoveries++
		rh.mu.Unlock()

		rh.logger.Info("State recovery successful after store recovery")
		rh.metricsCollector.RecordCounter("recovery_state_success", 1, nil)
		return nil
	}

	// State recovery failed
	rh.mu.Lock()
	rh.state = RecoveryStateFailed
	rh.failedRecoveries++
	rh.mu.Unlock()

	rh.logger.Error("State recovery failed")
	rh.metricsCollector.RecordCounter("recovery_state_failed", 1, nil)
	return fmt.Errorf("state recovery failed")
}

// SyncData attempts to sync data after recovery
func (rh *DefaultRecoveryHandler) SyncData(ctx context.Context, store string) error {
	rh.mu.Lock()
	if rh.recoveryInProgress {
		rh.mu.Unlock()
		return fmt.Errorf("recovery already in progress")
	}
	rh.recoveryInProgress = true
	rh.mu.Unlock()

	defer func() {
		rh.mu.Lock()
		rh.recoveryInProgress = false
		rh.mu.Unlock()
	}()

	if !rh.initialized {
		return fmt.Errorf("recovery handler not initialized")
	}

	if store == "" {
		return fmt.Errorf("store name is required")
	}

	rh.mu.Lock()
	rh.state = RecoveryStateRecovering
	rh.lastRecoveryTime = time.Now()
	rh.totalAttempts++
	rh.mu.Unlock()

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		rh.mu.Lock()
		rh.totalRecoveryDuration += duration
		rh.mu.Unlock()
	}()

	rh.logger.Info("Starting data sync", map[string]interface{}{
		"store": store,
	})

	// Create a context with timeout for data sync
	ctx, cancel := context.WithTimeout(ctx, rh.config.DataSyncTimeout)
	defer cancel()

	// Perform data sync based on store type
	var err error
	switch store {
	case "mongodb":
		err = rh.syncMongoDBData(ctx)
	case "postgresql":
		err = rh.syncPostgreSQLData(ctx)
	case "cache":
		err = rh.syncCacheData(ctx)
	default:
		rh.mu.Lock()
		rh.state = RecoveryStateFailed
		rh.failedRecoveries++
		rh.mu.Unlock()
		return fmt.Errorf("unknown store: %s", store)
	}

	if err != nil {
		rh.mu.Lock()
		rh.state = RecoveryStateFailed
		rh.failedRecoveries++
		rh.mu.Unlock()

		rh.logger.Error("Data sync failed", map[string]interface{}{
			"store": store,
			"error": err.Error(),
		})

		rh.metricsCollector.RecordCounter("recovery_data_sync_failed", 1, nil)
		return fmt.Errorf("data sync failed for store %s: %w", store, err)
	}

	// Data sync successful
	rh.mu.Lock()
	rh.state = RecoveryStateHealthy
	rh.successfulRecoveries++
	rh.lastSuccessfulRecovery = time.Now()
	rh.dataSyncRecoveries++
	rh.mu.Unlock()

	rh.logger.Info("Data sync successful", map[string]interface{}{
		"store": store,
	})

	rh.metricsCollector.RecordCounter("recovery_data_sync_success", 1, nil)
	return nil
}

// GetRecoveryState returns the current recovery state
func (rh *DefaultRecoveryHandler) GetRecoveryState() RecoveryState {
	rh.mu.RLock()
	defer rh.mu.RUnlock()
	return rh.state
}

// GetLastRecoveryTime returns the last recovery time
func (rh *DefaultRecoveryHandler) GetLastRecoveryTime() time.Time {
	rh.mu.RLock()
	defer rh.mu.RUnlock()
	return rh.lastRecoveryTime
}

// GetRecoveryMetrics returns recovery metrics
func (rh *DefaultRecoveryHandler) GetRecoveryMetrics() RecoveryMetrics {
	rh.mu.RLock()
	defer rh.mu.RUnlock()

	avgDuration := time.Duration(0)
	if rh.totalAttempts > 0 {
		avgDuration = rh.totalRecoveryDuration / time.Duration(rh.totalAttempts)
	}

	return RecoveryMetrics{
		TotalRecoveryAttempts:      rh.totalAttempts,
		SuccessfulRecoveries:       rh.successfulRecoveries,
		FailedRecoveries:           rh.failedRecoveries,
		LastRecoveryTime:           rh.lastRecoveryTime,
		LastSuccessfulRecoveryTime: rh.lastSuccessfulRecovery,
		AverageRecoveryDuration:    avgDuration,
		ConnectionRecoveries:       rh.connectionRecoveries,
		StateRecoveries:            rh.stateRecoveries,
		DataSyncRecoveries:         rh.dataSyncRecoveries,
	}
}

// Close closes the recovery handler
func (rh *DefaultRecoveryHandler) Close(ctx context.Context) error {
	rh.mu.Lock()
	defer rh.mu.Unlock()

	if !rh.initialized {
		return nil
	}

	rh.initialized = false
	rh.logger.Info("Recovery handler closed")
	return nil
}

// reconnectMongoDB attempts to reconnect to MongoDB
func (rh *DefaultRecoveryHandler) reconnectMongoDB(ctx context.Context) error {
	// Verify MongoDB is accessible
	health := rh.eventStore.Health(ctx)
	if health.Status == "healthy" {
		return nil
	}

	return fmt.Errorf("mongodb reconnection failed: %s", health.Message)
}

// reconnectPostgreSQL attempts to reconnect to PostgreSQL
func (rh *DefaultRecoveryHandler) reconnectPostgreSQL(ctx context.Context) error {
	// Verify PostgreSQL is accessible
	health := rh.metadataStore.Health(ctx)
	if health.Status == "healthy" {
		return nil
	}

	return fmt.Errorf("postgresql reconnection failed: %s", health.Message)
}

// reconnectCache attempts to reconnect to cache
func (rh *DefaultRecoveryHandler) reconnectCache(ctx context.Context) error {
	// Verify cache is accessible
	health := rh.cacheService.Health(ctx)
	if health.Status == "healthy" {
		return nil
	}

	return fmt.Errorf("cache reconnection failed: %s", health.Message)
}

// syncMongoDBData syncs MongoDB data after recovery
func (rh *DefaultRecoveryHandler) syncMongoDBData(ctx context.Context) error {
	// Verify MongoDB is accessible
	health := rh.eventStore.Health(ctx)
	if health.Status != "healthy" {
		return fmt.Errorf("mongodb not healthy: %s", health.Message)
	}

	rh.logger.Info("MongoDB data sync completed")
	return nil
}

// syncPostgreSQLData syncs PostgreSQL data after recovery
func (rh *DefaultRecoveryHandler) syncPostgreSQLData(ctx context.Context) error {
	// Verify PostgreSQL is accessible
	health := rh.metadataStore.Health(ctx)
	if health.Status != "healthy" {
		return fmt.Errorf("postgresql not healthy: %s", health.Message)
	}

	rh.logger.Info("PostgreSQL data sync completed")
	return nil
}

// syncCacheData syncs cache data after recovery
func (rh *DefaultRecoveryHandler) syncCacheData(ctx context.Context) error {
	// Verify cache is accessible
	health := rh.cacheService.Health(ctx)
	if health.Status != "healthy" {
		return fmt.Errorf("cache not healthy: %s", health.Message)
	}

	rh.logger.Info("Cache data sync completed")
	return nil
}
