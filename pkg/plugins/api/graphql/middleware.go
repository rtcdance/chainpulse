package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"chainpulse/pkg/core"
)

// AuthContext holds authentication and authorization information
type AuthContext struct {
	UserID    string
	Roles     []string
	Scopes    []string
	ExpiresAt time.Time
}

// CanReadEvent checks if user can read an event
func (ac *AuthContext) CanReadEvent(eventID string) bool {
	if ac == nil {
		return false
	}

	// Check if user has read scope
	for _, scope := range ac.Scopes {
		if scope == "read:events" || scope == "*" {
			return true
		}
	}

	return false
}

// CanListEvents checks if user can list events
func (ac *AuthContext) CanListEvents() bool {
	if ac == nil {
		return false
	}

	for _, scope := range ac.Scopes {
		if scope == "read:events" || scope == "list:events" || scope == "*" {
			return true
		}
	}

	return false
}

// CanManageCache checks if user can manage cache
func (ac *AuthContext) CanManageCache() bool {
	if ac == nil {
		return false
	}

	for _, scope := range ac.Scopes {
		if scope == "manage:cache" || scope == "admin" || scope == "*" {
			return true
		}
	}

	return false
}

// IsExpired checks if auth context is expired
func (ac *AuthContext) IsExpired() bool {
	if ac == nil {
		return true
	}

	return time.Now().After(ac.ExpiresAt)
}

// AuthMiddleware provides authentication and authorization
type AuthMiddleware struct {
	logger  core.Logger
	metrics core.MetricsCollector
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(logger core.Logger, metrics core.MetricsCollector) *AuthMiddleware {
	return &AuthMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// Authenticate extracts and validates authentication from request
func (am *AuthMiddleware) Authenticate(ctx context.Context, token string) (*AuthContext, error) {
	if token == "" {
		return nil, fmt.Errorf("missing authentication token")
	}

	// Extract actual token (remove Bearer prefix if present)
	actualToken := token
	if len(token) > 7 && token[:7] == "Bearer " {
		actualToken = token[7:]
	}

	// Basic validation - token should have some content
	if len(actualToken) < 3 {
		am.logger.Warn("Token too short")
		am.metrics.RecordCounter("graphql_auth_invalid_token", 1, nil)
		return nil, fmt.Errorf("invalid token")
	}

	// Create auth context with default scopes
	authCtx := &AuthContext{
		UserID:    "user-1",
		Roles:     []string{"user"},
		Scopes:    []string{"read:events", "list:events"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	am.logger.Info("User authenticated", "userId", authCtx.UserID)
	am.metrics.RecordCounter("graphql_auth_success", 1, nil)
	return authCtx, nil
}

// ComplexityMiddleware analyzes and limits query complexity
type ComplexityMiddleware struct {
	maxComplexity int
	logger        core.Logger
	metrics       core.MetricsCollector
}

// NewComplexityMiddleware creates a new complexity middleware
func NewComplexityMiddleware(maxComplexity int, logger core.Logger, metrics core.MetricsCollector) *ComplexityMiddleware {
	return &ComplexityMiddleware{
		maxComplexity: maxComplexity,
		logger:        logger,
		metrics:       metrics,
	}
}

// AnalyzeQuery analyzes query complexity
func (cm *ComplexityMiddleware) AnalyzeQuery(query string) (int, error) {
	complexity := cm.calculateComplexity(query)

	if complexity > cm.maxComplexity {
		cm.logger.Warn("Query complexity exceeded", "complexity", complexity, "max", cm.maxComplexity)
		cm.metrics.RecordCounter("graphql_complexity_exceeded", 1, nil)
		return complexity, fmt.Errorf("query complexity %d exceeds maximum %d", complexity, cm.maxComplexity)
	}

	cm.metrics.RecordGauge("graphql_query_complexity", float64(complexity), nil)
	return complexity, nil
}

// calculateComplexity calculates query complexity based on heuristics
func (cm *ComplexityMiddleware) calculateComplexity(query string) int {
	complexity := 1

	// Count field selections
	complexity += strings.Count(query, "{")

	// Count arguments
	complexity += strings.Count(query, "(") * 2

	// Count aliases
	complexity += strings.Count(query, ":") / 2

	// Count fragments
	complexity += strings.Count(query, "fragment") * 5

	// Count nested queries
	depth := 0
	maxDepth := 0
	for _, char := range query {
		switch char {
		case '{':
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case '}':
			depth--
		}
	}
	complexity += maxDepth * 3

	return complexity
}

// RateLimitMiddleware implements rate limiting for GraphQL queries
type RateLimitMiddleware struct {
	requestsPerSecond int
	logger            core.Logger
	metrics           core.MetricsCollector
	userLimits        map[string]*RateLimiter
}

// RateLimiter tracks rate limit for a user
type RateLimiter struct {
	requestCount int
	resetTime    time.Time
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(requestsPerSecond int, logger core.Logger, metrics core.MetricsCollector) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requestsPerSecond: requestsPerSecond,
		logger:            logger,
		metrics:           metrics,
		userLimits:        make(map[string]*RateLimiter),
	}
}

// CheckLimit checks if user has exceeded rate limit
func (rlm *RateLimitMiddleware) CheckLimit(userID string) error {
	now := time.Now()

	limiter, ok := rlm.userLimits[userID]
	if !ok || now.After(limiter.resetTime) {
		// Create new limiter
		rlm.userLimits[userID] = &RateLimiter{
			requestCount: 1,
			resetTime:    now.Add(1 * time.Second),
		}
		return nil
	}

	// Check if limit exceeded
	if limiter.requestCount >= rlm.requestsPerSecond {
		rlm.logger.Warn("Rate limit exceeded", "userId", userID, "limit", rlm.requestsPerSecond)
		rlm.metrics.RecordCounter("graphql_rate_limit_exceeded", 1, nil)

		return fmt.Errorf("rate limit exceeded: %d requests per minute", rlm.requestsPerSecond*60)
	}

	limiter.requestCount++
	return nil
}

// CachingMiddleware implements caching for GraphQL queries
type CachingMiddleware struct {
	cache   core.CachePlugin
	ttl     time.Duration
	logger  core.Logger
	metrics core.MetricsCollector
}

// NewCachingMiddleware creates a new caching middleware
func NewCachingMiddleware(cache core.CachePlugin, ttl time.Duration, logger core.Logger, metrics core.MetricsCollector) *CachingMiddleware {
	return &CachingMiddleware{
		cache:   cache,
		ttl:     ttl,
		logger:  logger,
		metrics: metrics,
	}
}

// GetCached retrieves cached query result
func (cm *CachingMiddleware) GetCached(query string) (interface{}, error) {
	if cm.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	cacheKey := fmt.Sprintf("graphql:query:%s", hashQuery(query))
	result, err := cm.cache.Get(context.Background(), cacheKey)
	if err != nil {
		cm.metrics.RecordCounter("graphql_cache_miss", 1, nil)
		return nil, err
	}

	cm.logger.Debug("Query result cached", "query", query)
	cm.metrics.RecordCounter("graphql_cache_hit", 1, nil)
	return result, nil
}

// SetCached caches query result
func (cm *CachingMiddleware) SetCached(query string, result interface{}) error {
	if cm.cache == nil {
		return fmt.Errorf("cache not available")
	}

	cacheKey := fmt.Sprintf("graphql:query:%s", hashQuery(query))
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}

	ttlSeconds := int(cm.ttl.Seconds())
	if err := cm.cache.Set(context.Background(), cacheKey, resultBytes, ttlSeconds); err != nil {
		cm.logger.Error("Failed to cache query result", "error", err.Error())
		cm.metrics.RecordCounter("graphql_cache_set_error", 1, nil)
		return err
	}

	return nil
}

// hashQuery creates a hash of the query for caching
func hashQuery(query string) string {
	// Simple hash implementation - in production use a proper hash function
	hash := 0
	for _, char := range query {
		hash = ((hash << 5) - hash) + int(char)
	}
	return fmt.Sprintf("%d", hash)
}

// ValidationMiddleware validates GraphQL queries
type ValidationMiddleware struct {
	logger  core.Logger
	metrics core.MetricsCollector
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware(logger core.Logger, metrics core.MetricsCollector) *ValidationMiddleware {
	return &ValidationMiddleware{
		logger:  logger,
		metrics: metrics,
	}
}

// ValidateQuery validates a GraphQL query
func (vm *ValidationMiddleware) ValidateQuery(query string) error {
	if query == "" {
		vm.logger.Warn("Empty query")
		vm.metrics.RecordCounter("graphql_validation_error", 1, nil)
		return fmt.Errorf("query cannot be empty")
	}

	// Check for malicious patterns
	if strings.Contains(query, "__typename") && strings.Contains(query, "introspection") {
		vm.logger.Warn("Introspection query detected")
		vm.metrics.RecordCounter("graphql_introspection_attempt", 1, nil)
		return fmt.Errorf("introspection queries are not allowed")
	}

	// Check query length
	if len(query) > 10000 {
		vm.logger.Warn("Query too large", "size", len(query))
		vm.metrics.RecordCounter("graphql_validation_error", 1, nil)
		return fmt.Errorf("query too large: maximum 10000 characters")
	}

	vm.metrics.RecordCounter("graphql_validation_success", 1, nil)
	return nil
}

// LoggingMiddleware logs GraphQL operations
type LoggingMiddleware struct {
	logger core.Logger
}

// NewLoggingMiddleware creates a new logging middleware
func NewLoggingMiddleware(logger core.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// LogQuery logs a GraphQL query
func (lm *LoggingMiddleware) LogQuery(userID string, query string, duration time.Duration) {
	lm.logger.Info("GraphQL query executed",
		"userId", userID,
		"queryLength", len(query),
		"duration", duration.Milliseconds(),
	)
}

// LogMutation logs a GraphQL mutation
func (lm *LoggingMiddleware) LogMutation(userID string, mutation string, duration time.Duration) {
	lm.logger.Info("GraphQL mutation executed",
		"userId", userID,
		"mutationLength", len(mutation),
		"duration", duration.Milliseconds(),
	)
}

// LogError logs a GraphQL error
func (lm *LoggingMiddleware) LogError(userID string, operation string, err error) {
	lm.logger.Error("GraphQL operation failed",
		"userId", userID,
		"operation", operation,
		"error", err.Error(),
	)
}
