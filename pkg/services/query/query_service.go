package query

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/logkeys"
)

// QueryService defines the interface for query execution with cache-first pattern.
// This is a type alias for core.QueryService — both are the same type.
type QueryService = core.QueryService

// QueryRequest is a type alias for core.QueryRequest.
type QueryRequest = core.QueryRequest

// QueryResult is a type alias for core.QueryResult.
type QueryResult = core.QueryResult

// RuntimeSummarizer provides a compact runtime summary for operator-facing
// query service surfaces.
type RuntimeSummarizer interface {
	RuntimeSummary(ctx context.Context) *RuntimeSummary
}

// RuntimeSummary represents compact query runtime posture facts.
type RuntimeSummary struct {
	Status                string
	Message               string
	QueryPosture          string
	CachePosture          string
	CircuitBreakerPosture string
	ConsistencyPosture    string
	ReliabilityHint       string
}

// DefaultQueryService provides default implementation of QueryService
type DefaultQueryService struct {
	mu               sync.RWMutex
	mongoAdapter     MongoDBAdapter
	postgresAdapter  PostgreSQLAdapter
	cacheService     CacheService
	logger           core.Logger
	metricsCollector core.MetricsCollector
	initialized      bool
	running          bool
}

// NewQueryService creates a new query service
func NewQueryService(
	mongoAdapter MongoDBAdapter,
	postgresAdapter PostgreSQLAdapter,
	cacheService CacheService,
	logger core.Logger,
	metricsCollector core.MetricsCollector,
) *DefaultQueryService {
	return &DefaultQueryService{
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
		qs.logger.Warn("failed to initialize PostgreSQL adapter, falling back to MongoDB only", "error", err.Error())
	}

	if err := qs.cacheService.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize cache service: %w", err)
	}

	qs.initialized = true
	qs.logger.Info("Query service initialized", logkeys.LogKeyComponent, core.ComponentQueryService)

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

	if err := qs.cacheService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start cache service: %w", err)
	}

	qs.running = true
	qs.logger.Info("Query service started", logkeys.LogKeyComponent, core.ComponentQueryService)

	return nil
}

// Stop stops the query service
func (qs *DefaultQueryService) Stop(ctx context.Context) error {
	qs.mu.Lock()
	defer qs.mu.Unlock()

	if !qs.running {
		return nil
	}

	if err := qs.cacheService.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop cache service: %w", err)
	}

	qs.running = false
	qs.logger.Info("Query service stopped", logkeys.LogKeyComponent, core.ComponentQueryService)

	return nil
}

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

	var cacheGet func() (*QueryResult, bool)
	var cacheSet func(*QueryResult) error
	if req.CacheKey != "" {
		cacheKey := req.CacheKey
		cacheTTL := req.CacheTTL
		cacheGet = func() (*QueryResult, bool) {
			events, total, err := qs.cacheService.GetQueryResult(ctx, cacheKey)
			if err == nil && events != nil {
				return &QueryResult{Events: events, Total: total}, true
			}
			return nil, false
		}
		cacheSet = func(r *QueryResult) error {
			return qs.cacheService.SetQueryResult(ctx, cacheKey, r.Events, r.Total, cacheTTL)
		}
	}

	result, fromCache, source, mongoErr, postgresErr := queryWithFallback(
		ctx,
		qs.logger,
		qs.metricsCollector,
		cacheGet,
		cacheSet,
		func() (*QueryResult, error) { return qs.mongoAdapter.Query(ctx, req) },
		func() (*QueryResult, error) { return qs.postgresAdapter.Query(ctx, req) },
		func(r *QueryResult) bool { return r == nil || len(r.Events) == 0 },
		"failed to cache query result",
		"cache_key", req.CacheKey,
		map[string]string{"adapter": "mongodb"},
		map[string]string{"adapter": "postgres"},
	)

	duration := time.Since(start).Milliseconds()

	if fromCache {
		qs.metricsCollector.RecordHistogram("query_cache_hit_time_ms", float64(duration), map[string]string{})
		qs.logger.Info("Cache hit", logkeys.LogKeyKey, req.CacheKey, logkeys.LogKeyDuration, duration)
		return &QueryResult{
			Events: result.Events, Total: result.Total,
			CacheHit: true, ResponseTime: duration, Source: "cache",
		}, nil
	}

	if source != "" {
		histogram := "query_mongodb_time_ms"
		if source == "postgresql" {
			histogram = "query_postgres_time_ms"
		}
		qs.metricsCollector.RecordHistogram(histogram, float64(duration), map[string]string{})
		return &QueryResult{
			Events: result.Events, Total: result.Total,
			CacheHit: false, ResponseTime: duration, Source: source,
		}, nil
	}

	if mongoErr != nil {
		qs.logger.Error("MongoDB query failed", logkeys.LogKeyError, mongoErr)
	}

	if postgresErr != nil {
		qs.logger.Error("PostgreSQL query failed", logkeys.LogKeyError, postgresErr)
	}

	if mongoErr != nil && postgresErr != nil {
		qs.metricsCollector.RecordCounter("query_error", 1, map[string]string{"reason": "both_backends_failed"})
		return nil, fmt.Errorf("both backends failed: mongo=%w, postgres=%w", mongoErr, postgresErr)
	}

	qs.metricsCollector.RecordCounter("query_empty", 1, map[string]string{})

	return &QueryResult{
		Events: []blockchain.BlockchainEvent{}, Total: 0,
		CacheHit: false, ResponseTime: duration, Source: "none",
	}, nil
}

func (qs *DefaultQueryService) QueryByHash(ctx context.Context, hash string) (*blockchain.BlockchainEvent, error) {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if !qs.running {
		return nil, fmt.Errorf("query service not running")
	}

	if hash == "" {
		return nil, fmt.Errorf("hash is required")
	}

	start := time.Now()

	cacheKey := fmt.Sprintf("event:%s", hash)
	cacheGet := func() (*blockchain.BlockchainEvent, bool) {
		event, err := qs.cacheService.GetSingle(ctx, cacheKey)
		if err == nil && event != nil {
			return event, true
		}
		return nil, false
	}
	cacheSet := func(event *blockchain.BlockchainEvent) error {
		return qs.cacheService.SetSingle(ctx, cacheKey, event, core.DefaultEventCacheTTL)
	}

	result, fromCache, source, mongoErr, postgresErr := queryWithFallback(
		ctx,
		qs.logger,
		qs.metricsCollector,
		cacheGet,
		cacheSet,
		func() (*blockchain.BlockchainEvent, error) { return qs.mongoAdapter.QueryByHash(ctx, hash) },
		func() (*blockchain.BlockchainEvent, error) { return qs.postgresAdapter.QueryByHash(ctx, hash) },
		func(e *blockchain.BlockchainEvent) bool { return e == nil },
		"failed to cache event by hash",
		"hash", hash,
		map[string]string{"adapter": "mongodb", "op": "by_hash"},
		map[string]string{"adapter": "postgres", "op": "by_hash"},
	)

	duration := time.Since(start).Milliseconds()

	if fromCache {
		qs.metricsCollector.RecordHistogram("query_by_hash_cache_hit_time_ms", float64(duration), map[string]string{})
		return result, nil
	}

	if source != "" {
		histogram := "query_by_hash_mongodb_time_ms"
		if source == "postgresql" {
			histogram = "query_by_hash_postgres_time_ms"
		}
		qs.metricsCollector.RecordHistogram(histogram, float64(duration), map[string]string{})
		return result, nil
	}

	qs.metricsCollector.RecordCounter("query_by_hash_error", 1, map[string]string{})

	if mongoErr != nil {
		qs.logger.Error("MongoDB query by hash failed", logkeys.LogKeyHash, hash, logkeys.LogKeyError, mongoErr)
	}

	if postgresErr != nil {
		qs.logger.Error("PostgreSQL query by hash failed", logkeys.LogKeyHash, hash, logkeys.LogKeyError, postgresErr)
	}

	return nil, fmt.Errorf("event not found: %s", hash)
}

func queryWithFallback[T any](
	ctx context.Context,
	logger core.Logger,
	metrics core.MetricsCollector,
	cacheGet func() (T, bool),
	cacheSet func(T) error,
	mongoFetch func() (T, error),
	postgresFetch func() (T, error),
	isEmpty func(T) bool,
	cacheWarnMsg string,
	cacheWarnKey string,
	cacheWarnVal string,
	mongoCacheLabel map[string]string,
	postgresCacheLabel map[string]string,
) (result T, fromCache bool, source string, mongoErr, postgresErr error) {
	if cacheGet != nil {
		if cached, ok := cacheGet(); ok {
			return cached, true, "cache", nil, nil
		}
	}

	mongoResult, mongoErr := mongoFetch()
	if mongoErr == nil && !isEmpty(mongoResult) {
		if cacheSet != nil {
			if err := cacheSet(mongoResult); err != nil {
				logger.Warn(cacheWarnMsg, cacheWarnKey, cacheWarnVal, "error", err)
				metrics.RecordCounter("query_cache_write_errors", 1, mongoCacheLabel)
			}
		}
		return mongoResult, false, "mongodb", nil, nil
	}

	postgresResult, postgresErr := postgresFetch()
	if postgresErr == nil && !isEmpty(postgresResult) {
		if cacheSet != nil {
			if err := cacheSet(postgresResult); err != nil {
				logger.Warn(cacheWarnMsg, cacheWarnKey, cacheWarnVal, "error", err)
				metrics.RecordCounter("query_cache_write_errors", 1, postgresCacheLabel)
			}
		}
		return postgresResult, false, "postgresql", nil, nil
	}

	var zero T
	return zero, false, "", mongoErr, postgresErr
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
		qs.logger.Error("Failed to invalidate cache", logkeys.LogKeyKey, key, logkeys.LogKeyError, err)
		return fmt.Errorf("invalidate cache key %s: %w", key, err)
	}

	qs.logger.Info("Cache invalidated", logkeys.LogKeyKey, key)

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

	degradedDependencies := make([]string, 0, 3)
	if mongoHealth.Status == "unhealthy" {
		degradedDependencies = append(degradedDependencies, "MongoDB")
	}
	if postgresHealth.Status == "unhealthy" {
		degradedDependencies = append(degradedDependencies, "PostgreSQL")
	}
	if cacheHealth.Status == "unhealthy" {
		degradedDependencies = append(degradedDependencies, "Cache")
	}

	if len(degradedDependencies) > 0 {
		return &core.HealthStatus{
			Status:  "degraded",
			Message: strings.Join(degradedDependencies, " and ") + " are unhealthy",
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
	circuitPosture := "circuit-breaker-not-applicable"
	consistencyPosture := "consistency-check-not-applicable"

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
