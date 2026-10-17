package reliability

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	domainquery "github.com/rtcdance/chainpulse/pkg/domain/query"
)

// StatelessService represents a service with no local state
type StatelessService struct {
	mu           sync.RWMutex
	id           string
	name         string
	cache        *DistributedCache
	database     domainquery.EventStore
	stateVersion int64
	lastSyncTime time.Time
	metrics      *StatelessMetrics
	healthStatus string
}

// StatelessMetrics tracks stateless service metrics
type StatelessMetrics struct {
	mu                    sync.RWMutex
	RequestsProcessed     int64
	StateRetrievals       int64
	StateSyncs            int64
	CacheMisses           int64
	DatabaseQueries       int64
	AverageLatency        time.Duration
	TotalProcessingTime   time.Duration
	LastProcessedTime     time.Time
	ExternalStateAccesses int64
}

// DistributedCache represents a distributed cache for state
type DistributedCache struct {
	mu      sync.RWMutex
	data    map[string]any
	ttl     map[string]time.Time
	metrics *CacheMetrics
}

// CacheMetrics tracks cache metrics
type CacheMetrics struct {
	mu        sync.RWMutex
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
}

// MessageQueueClient represents a message queue client
type MessageQueueClient interface {
	Publish(ctx context.Context, topic string, message any) error
	Subscribe(ctx context.Context, topic string, handler func(any) error) error
	GetMetrics() map[string]any
}

// NewStatelessService creates a new stateless service
func NewStatelessService(id, name string, cache *DistributedCache, db domainquery.EventStore) *StatelessService {
	return &StatelessService{
		id:           id,
		name:         name,
		cache:        cache,
		database:     db,
		lastSyncTime: time.Now(),
		healthStatus: "healthy",
		metrics: &StatelessMetrics{
			LastProcessedTime: time.Now(),
		},
	}
}

// ProcessRequest processes a request without storing local state
func (ss *StatelessService) ProcessRequest(ctx context.Context, requestID string, data any) (any, error) {
	start := time.Now()
	defer func() {
		ss.recordLatency(time.Since(start))
	}()

	ss.mu.RLock()
	defer ss.mu.RUnlock()

	// Retrieve state from external systems
	state, err := ss.retrieveState(ctx, requestID)
	if err != nil {
		ss.metrics.mu.Lock()
		ss.metrics.RequestsProcessed++
		ss.metrics.mu.Unlock()
		return nil, fmt.Errorf("failed to retrieve state: %w", err)
	}

	// Process request with external state
	result := ss.processWithExternalState(ctx, state, data)

	// Store result in external systems
	if err := ss.storeState(ctx, requestID, result); err != nil {
		return nil, fmt.Errorf("failed to store state: %w", err)
	}

	ss.metrics.mu.Lock()
	ss.metrics.RequestsProcessed++
	ss.metrics.LastProcessedTime = time.Now()
	ss.metrics.mu.Unlock()

	return result, nil
}

// retrieveState retrieves state from external systems
func (ss *StatelessService) retrieveState(ctx context.Context, requestID string) (any, error) {
	ss.metrics.mu.Lock()
	ss.metrics.StateRetrievals++
	ss.metrics.ExternalStateAccesses++
	ss.metrics.mu.Unlock()

	// Try cache first
	if ss.cache != nil {
		if state, ok := ss.cache.Get(requestID); ok {
			ss.cache.metrics.mu.Lock()
			ss.cache.metrics.Hits++
			ss.cache.metrics.mu.Unlock()
			return state, nil
		}

		ss.cache.metrics.mu.Lock()
		ss.cache.metrics.Misses++
		ss.cache.metrics.mu.Unlock()
	}

	// Fall back to database
	if ss.database != nil {
		ss.metrics.mu.Lock()
		ss.metrics.DatabaseQueries++
		ss.metrics.mu.Unlock()

		event, err := ss.database.GetEvent(ctx, requestID)
		if err != nil {
			return nil, err
		}

		// Cache the result
		if ss.cache != nil {
			ss.cache.Set(requestID, event, 5*time.Minute)
		}

		return event, nil
	}

	return nil, fmt.Errorf("no state storage available")
}

// storeState stores state in external systems
func (ss *StatelessService) storeState(ctx context.Context, requestID string, state any) error {
	ss.metrics.mu.Lock()
	ss.metrics.ExternalStateAccesses++
	ss.metrics.mu.Unlock()

	// Store in cache
	if ss.cache != nil {
		ss.cache.Set(requestID, state, 5*time.Minute)
	}

	// Store in database
	if ss.database != nil && ss.isEvent(state) {
		event, ok := state.(*blockchain.BlockchainEvent)
		if !ok {
			return fmt.Errorf("state is not a BlockchainEvent")
		}
		if err := ss.database.InsertEvent(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

// processWithExternalState processes request using external state
func (ss *StatelessService) processWithExternalState(ctx context.Context, state any, data any) any {
	slog.Warn("processWithExternalState: placeholder — actual business logic not yet implemented, returning raw input")
	return map[string]any{
		"state": state,
		"data":  data,
	}
}

// isEvent checks if value is an Event
func (ss *StatelessService) isEvent(v any) bool {
	_, ok := v.(*blockchain.BlockchainEvent)
	return ok
}

// recordLatency records request processing latency
func (ss *StatelessService) recordLatency(latency time.Duration) {
	ss.metrics.mu.Lock()
	defer ss.metrics.mu.Unlock()

	ss.metrics.TotalProcessingTime += latency
	if ss.metrics.RequestsProcessed > 0 {
		ss.metrics.AverageLatency = ss.metrics.TotalProcessingTime / time.Duration(ss.metrics.RequestsProcessed)
	}
}

// SyncState synchronizes state across instances
func (ss *StatelessService) SyncState(ctx context.Context) error {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.metrics.mu.Lock()
	ss.metrics.StateSyncs++
	ss.metrics.mu.Unlock()

	ss.lastSyncTime = time.Now()
	ss.stateVersion++

	return nil
}

// GetMetrics returns service metrics
func (ss *StatelessService) GetMetrics() StatelessMetrics {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	ss.metrics.mu.RLock()
	defer ss.metrics.mu.RUnlock()

	return StatelessMetrics{
		RequestsProcessed:     ss.metrics.RequestsProcessed,
		StateRetrievals:       ss.metrics.StateRetrievals,
		StateSyncs:            ss.metrics.StateSyncs,
		CacheMisses:           ss.metrics.CacheMisses,
		DatabaseQueries:       ss.metrics.DatabaseQueries,
		AverageLatency:        ss.metrics.AverageLatency,
		TotalProcessingTime:   ss.metrics.TotalProcessingTime,
		LastProcessedTime:     ss.metrics.LastProcessedTime,
		ExternalStateAccesses: ss.metrics.ExternalStateAccesses,
	}
}

// Health returns service health status
func (ss *StatelessService) Health() core.HealthStatus {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return core.HealthStatus{
		Status:    ss.healthStatus,
		Timestamp: time.Now(),
		Details: map[string]any{
			"service_id":              ss.id,
			"service_name":            ss.name,
			"state_version":           ss.stateVersion,
			"last_sync_time":          ss.lastSyncTime,
			"requests_processed":      ss.metrics.RequestsProcessed,
			"external_state_accesses": ss.metrics.ExternalStateAccesses,
		},
	}
}

// NewDistributedCache creates a new distributed cache
func NewDistributedCache() *DistributedCache {
	return &DistributedCache{
		data:    make(map[string]any),
		ttl:     make(map[string]time.Time),
		metrics: &CacheMetrics{},
	}
}

// Get retrieves a value from cache
func (dc *DistributedCache) Get(key string) (any, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// Check if key exists and hasn't expired
	if expiry, exists := dc.ttl[key]; exists {
		if time.Now().Before(expiry) {
			value := dc.data[key]
			dc.metrics.mu.Lock()
			dc.metrics.Hits++
			dc.metrics.mu.Unlock()
			return value, true
		}
	}

	dc.metrics.mu.Lock()
	dc.metrics.Misses++
	dc.metrics.mu.Unlock()

	return nil, false
}

// Set stores a value in cache with TTL
func (dc *DistributedCache) Set(key string, value any, ttl time.Duration) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.data[key] = value
	dc.ttl[key] = time.Now().Add(ttl)

	dc.metrics.mu.Lock()
	dc.metrics.Sets++
	dc.metrics.mu.Unlock()
}

// Delete removes a value from cache
func (dc *DistributedCache) Delete(key string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	delete(dc.data, key)
	delete(dc.ttl, key)

	dc.metrics.mu.Lock()
	dc.metrics.Deletes++
	dc.metrics.mu.Unlock()
}

// Cleanup removes expired entries
func (dc *DistributedCache) Cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	for key, expiry := range dc.ttl {
		if now.After(expiry) {
			delete(dc.data, key)
			delete(dc.ttl, key)

			dc.metrics.mu.Lock()
			dc.metrics.Evictions++
			dc.metrics.mu.Unlock()
		}
	}
}

// GetMetrics returns cache metrics
func (dc *DistributedCache) GetMetrics() map[string]any {
	dc.metrics.mu.RLock()
	defer dc.metrics.mu.RUnlock()

	return map[string]any{
		"hits":      dc.metrics.Hits,
		"misses":    dc.metrics.Misses,
		"sets":      dc.metrics.Sets,
		"deletes":   dc.metrics.Deletes,
		"evictions": dc.metrics.Evictions,
	}
}

// GetSize returns the number of items in cache
func (dc *DistributedCache) GetSize() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.data)
}

// Clear clears all cache entries
func (dc *DistributedCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.data = make(map[string]any)
	dc.ttl = make(map[string]time.Time)
}
