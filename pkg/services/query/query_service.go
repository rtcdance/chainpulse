package query

import (
	"context"
	"fmt"
	"sync"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
)

// QueryService defines the interface for query execution with cache-first pattern
type QueryService interface {
	// Query executes a query with cache-first pattern
	Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)

	// QueryByHash retrieves a single item by hash
	QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error)

	// InvalidateCache invalidates cache for a key
	InvalidateCache(ctx context.Context, key string) error

	// Health returns the health status
	Health(ctx context.Context) *core.HealthStatus
}

// RuntimeSummarizer provides a compact runtime summary for operator-facing
// query service surfaces.
type RuntimeSummarizer interface {
	RuntimeSummary(ctx context.Context) *RuntimeSummary
}

// RuntimeSummary represents compact query runtime posture facts.
type RuntimeSummary struct {
	Status                 string
	Message                string
	QueryPosture           string
	CachePosture           string
	CircuitBreakerPosture  string
	ConsistencyPosture     string
	ReliabilityHint        string
}

// QueryRequest represents a query request
type QueryRequest struct {
	// Query type: "mongodb" or "postgresql"
	QueryType string

	// Collection or table name
	Collection string

	// Filter criteria
	Filter map[string]interface{}

	// Pagination
	Limit  int64
	Offset int64

	// Cache key
	CacheKey string

	// Cache TTL
	CacheTTL time.Duration

	// Sort order
	Sort map[string]int
}

// QueryResult represents a query result
type QueryResult struct {
	// Events returned
	Events []core.BlockchainEvent

	// Total count
	Total int64

	// Whether result came from cache
	CacheHit bool

	// Response time in milliseconds
	ResponseTime int64

	// Source: "cache", "mongodb", or "postgresql"
	Source string
}

// DefaultQueryService provides default implementation of QueryService
type DefaultQueryService struct {
	mu              sync.RWMutex
	dbManager       database.DatabaseManager
	mongoAdapter    MongoDBAdapter
	postgresAdapter PostgreSQLAdapter
	cacheService    CacheService
	logger          core.Logger
	metricsCollector core.MetricsCollector
	initialized     bool
	running         bool
}

// NewQueryService creates a new query service
func NewQueryService(
	dbManager database.DatabaseManager,
	mongoAdapter MongoDBAdapter,
	postgresAdapter PostgreSQLAdapter,
	cacheService CacheService,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) *DefaultQueryService {
	return &DefaultQueryService{
		dbManager:        dbManager,
		mongoAdapter:     mongoAdapter,
		postgresAdapter:  postgresAdapter,
		cacheService:     cacheService,
		logger:           logger,
		metricsCollector: metricsCollector,
	}
}

// Initialize initializes the query service
func (qs *DefaultQueryService) Initialize(ctx context.Context) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if qs.initialized {
		return fmt.Errorf("query service already initialized")
	}

	// Initialize adapters
	if err := qs.mongoAdapter.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize MongoDB adapter: %w", err)
	}

	if err := qs.postgresAdapter.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize PostgreSQL adapter: %w", err)
	}

	if err := qs.cacheService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize cache service: %w", err)
	}

	qs.initialized = true
	qs.logger.Info("Query service initialized", map[string]interface{}{
		"component": "query-service",
	})

	return nil
}

// Start starts the query service
func (qs *DefaultQueryService) Start(ctx context.Context) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if !qs.initialized {
		return fmt.Errorf("query service not initialized")
	}

	if qs.running {
		return fmt.Errorf("query service already running")
	}

	qs.running = true
	qs.logger.Info("Query service started", map[string]interface{}{
		"component": "query-service",
	})

	return nil
}

// Stop stops the query service
func (qs *DefaultQueryService) Stop(ctx context.Context) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if !qs.running {
		return fmt.Errorf("query service not running")
	}

	qs.running = false
	qs.logger.Info("Query service stopped", map[string]interface{}{
		"component": "query-service",
	})

	return nil
}

// Query executes a query with cache-first pattern
func (qs *DefaultQueryService) Query(ctx context.Context, req *QueryRequest) (*QueryResult, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if !qs.running {
		return nil, fmt.Errorf("query service not running")
	}

	if req == nil {
		return nil, fmt.Errorf("query request is required")
	}

	start := time.Now()

	// Step 1: Try cache first
	if req.CacheKey != "" {
		if cached, err := qs.cacheService.Get(ctx, req.CacheKey); err == nil && cached != nil {
			duration := time.Since(start).Milliseconds()
			qs.metricsCollector.RecordHistogram("query_cache_hit_time_ms", float64(duration), map[string]string{})
			qs.logger.Info("Cache hit", map[string]interface{}{
				"cache_key": req.CacheKey,
				"duration":  duration,
			})

			return &QueryResult{
				Events:       cached,
				Total:        int64(len(cached)),
				CacheHit:     true,
				ResponseTime: duration,
				Source:       "cache",
			}, nil
		}
	}

	// Step 2: Try MongoDB
	mongoResult, mongoErr := qs.mongoAdapter.Query(ctx, req)
	if mongoErr == nil && mongoResult != nil && len(mongoResult.Events) > 0 {
		// Cache the result
		if req.CacheKey != "" {
			_ = qs.cacheService.Set(ctx, req.CacheKey, mongoResult.Events, req.CacheTTL)
		}

		duration := time.Since(start).Milliseconds()
		qs.metricsCollector.RecordHistogram("query_mongodb_time_ms", float64(duration), map[string]string{})

		return &QueryResult{
			Events:       mongoResult.Events,
			Total:        mongoResult.Total,
			CacheHit:     false,
			ResponseTime: duration,
			Source:       "mongodb",
		}, nil
	}

	// Step 3: Fall back to PostgreSQL
	postgresResult, postgresErr := qs.postgresAdapter.Query(ctx, req)
	if postgresErr == nil && postgresResult != nil && len(postgresResult.Events) > 0 {
		// Cache the result
		if req.CacheKey != "" {
			_ = qs.cacheService.Set(ctx, req.CacheKey, postgresResult.Events, req.CacheTTL)
		}

		duration := time.Since(start).Milliseconds()
		qs.metricsCollector.RecordHistogram("query_postgres_time_ms", float64(duration), map[string]string{})

		return &QueryResult{
			Events:       postgresResult.Events,
			Total:        postgresResult.Total,
			CacheHit:     false,
			ResponseTime: duration,
			Source:       "postgresql",
		}, nil
	}

	// Both databases failed or returned no results
	duration := time.Since(start).Milliseconds()
	qs.metricsCollector.RecordCounter("query_error", 1, map[string]string{})

	if mongoErr != nil {
		qs.logger.Error("MongoDB query failed", map[string]interface{}{
			"error": mongoErr.Error(),
		})
	}

	if postgresErr != nil {
		qs.logger.Error("PostgreSQL query failed", map[string]interface{}{
			"error": postgresErr.Error(),
		})
	}

	return &QueryResult{
		Events:       []core.BlockchainEvent{},
		Total:        0,
		CacheHit:     false,
		ResponseTime: duration,
		Source:       "none",
	}, nil
}

// QueryByHash retrieves a single item by hash
func (qs *DefaultQueryService) QueryByHash(ctx context.Context, hash string) (*core.BlockchainEvent, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if !qs.running {
		return nil, fmt.Errorf("query service not running")
	}

	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	// Try cache first
	cacheKey := fmt.Sprintf("event:%s", hash)
	if cached, err := qs.cacheService.GetSingle(ctx, cacheKey); err == nil && cached != nil {
		duration := time.Since(start).Milliseconds()
		qs.metricsCollector.RecordHistogram("query_by_hash_cache_hit_time_ms", float64(duration), map[string]string{})
		return cached, nil
	}

	// Try MongoDB
	event, mongoErr := qs.mongoAdapter.QueryByHash(ctx, hash)
	if mongoErr == nil && event != nil {
		// Cache the result
		_ = qs.cacheService.SetSingle(ctx, cacheKey, event, 1*time.Hour)
		duration := time.Since(start).Milliseconds()
		qs.metricsCollector.RecordHistogram("query_by_hash_mongodb_time_ms", float64(duration), map[string]string{})
		return event, nil
	}

	// Fall back to PostgreSQL
	event, postgresErr := qs.postgresAdapter.QueryByHash(ctx, hash)
	if postgresErr == nil && event != nil {
		// Cache the result
		_ = qs.cacheService.SetSingle(ctx, cacheKey, event, 1*time.Hour)
		duration := time.Since(start).Milliseconds()
		qs.metricsCollector.RecordHistogram("query_by_hash_postgres_time_ms", float64(duration), map[string]string{})
		return event, nil
	}

	qs.metricsCollector.RecordCounter("query_by_hash_error", 1, map[string]string{})

	if mongoErr != nil {
		qs.logger.Error("MongoDB query by hash failed", map[string]interface{}{
			"hash":  hash,
			"error": mongoErr.Error(),
		})
	}

	if postgresErr != nil {
		qs.logger.Error("PostgreSQL query by hash failed", map[string]interface{}{
			"hash":  hash,
			"error": postgresErr.Error(),
		})
	}

	return nil, fmt.Errorf("event not found: %s", hash)
}

// InvalidateCache invalidates cache for a key
func (qs *DefaultQueryService) InvalidateCache(ctx context.Context, key string) error {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if !qs.running {
		return fmt.Errorf("query service not running")
	}

	if key == "" {
		return fmt.Errorf("cache key is required")
	}

	if err := qs.cacheService.Delete(ctx, key); err != nil {
		qs.logger.Error("Failed to invalidate cache", map[string]interface{}{
			"key":   key,
			"error": err.Error(),
		})
		return err
	}

	qs.logger.Info("Cache invalidated", map[string]interface{}{
		"key": key,
	})

	return nil
}

// Health returns the health status
func (qs *DefaultQueryService) Health(ctx context.Context) *core.HealthStatus {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if !qs.initialized {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Query service not initialized",
		}
	}

	if !qs.running {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Query service not running",
		}
	}

	// Check adapter health
	mongoHealth := qs.mongoAdapter.Health(ctx)
	postgresHealth := qs.postgresAdapter.Health(ctx)
	cacheHealth := qs.cacheService.Health(ctx)

	if mongoHealth.Status == "unhealthy" && postgresHealth.Status == "unhealthy" {
		return &core.HealthStatus{
			Status:  "unhealthy",
			Message: "Both MongoDB and PostgreSQL are unhealthy",
		}
	}

	if cacheHealth.Status == "unhealthy" {
		return &core.HealthStatus{
			Status:  "degraded",
			Message: "Cache is unhealthy",
		}
	}

	return &core.HealthStatus{
		Status:  "healthy",
		Message: "Query service healthy",
	}
}

// RuntimeSummary returns a compact query runtime summary suitable for
// operator-facing runtime surfaces.
func (qs *DefaultQueryService) RuntimeSummary(ctx context.Context) *RuntimeSummary {
	health := qs.Health(ctx)
	if health == nil {
		health = &core.HealthStatus{
			Status:  "unknown",
			Message: "Query service health unavailable",
		}
	}

	cachePosture := qs.classifyCachePosture(ctx)
	queryPosture := classifyQueryRuntimePosture(health.Status)
	circuitPosture := "circuit-not-wired"
	consistencyPosture := "consistency-not-wired"

	return &RuntimeSummary{
		Status:                health.Status,
		Message:               health.Message,
		QueryPosture:          queryPosture,
		CachePosture:          cachePosture,
		CircuitBreakerPosture: circuitPosture,
		ConsistencyPosture:    consistencyPosture,
		ReliabilityHint:       buildQueryRuntimeReliabilityHint(health.Status, cachePosture),
	}
}

func (qs *DefaultQueryService) classifyCachePosture(ctx context.Context) string {
	if qs.cacheService == nil {
		return "cache-unavailable"
	}

	health := qs.cacheService.Health(ctx)
	if health == nil {
		return "cache-unobserved"
	}

	switch health.Status {
	case "healthy":
		return "cache-ready"
	case "degraded":
		return "cache-degraded"
	case "unhealthy":
		return "cache-unhealthy"
	default:
		return "cache-unobserved"
	}
}

func classifyQueryRuntimePosture(status string) string {
	switch status {
	case "healthy":
		return "query-runtime-ready"
	case "degraded":
		return "query-runtime-degraded"
	case "unhealthy":
		return "query-runtime-unhealthy"
	default:
		return "query-runtime-unobserved"
	}
}

func buildQueryRuntimeReliabilityHint(status, cachePosture string) string {
	switch {
	case status == "healthy" && cachePosture == "cache-ready":
		return "query runtime is healthy and cache-first reads are available"
	case status == "healthy":
		return "query runtime is healthy, but cache posture should be verified before assuming cache-first reads"
	case status == "degraded" && cachePosture == "cache-unhealthy":
		return "query runtime is degraded and cache is unhealthy; expect store-backed reads while cache is restored"
	case status == "degraded":
		return "query runtime is degraded; investigate backing services before treating query reads as fully reliable"
	case status == "unhealthy":
		return "query runtime is unhealthy; restore backing services before relying on query reads"
	default:
		return "verify query runtime posture before relying on query service reads"
	}
}
