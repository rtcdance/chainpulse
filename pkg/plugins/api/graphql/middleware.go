package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api/shared"
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
	logger         core.Logger
	metrics        core.MetricsCollector
	tokenValidator TokenValidator
	requireAuth    bool
}

// TokenValidator defines the interface for token validation
type TokenValidator interface {
	ValidateToken(ctx context.Context, authHeader string) ValidationResult
	ValidateAPIKey(ctx context.Context, apiKey string) ValidationResult
}

// ValidationResult represents the outcome of token validation
type ValidationResult struct {
	Valid       bool
	ClientID    string
	UserID      string
	Roles       []string
	Permissions []string
	Error       string
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(logger core.Logger, metrics core.MetricsCollector) *AuthMiddleware {
	return &AuthMiddleware{
		logger:      logger,
		metrics:     metrics,
		requireAuth: true,
	}
}

// SetTokenValidator sets the token validator for real authentication
func (am *AuthMiddleware) SetTokenValidator(validator TokenValidator) {
	am.tokenValidator = validator
}

// SetRequireAuth configures whether authentication is required
func (am *AuthMiddleware) SetRequireAuth(require bool) {
	am.requireAuth = require
}

// Authenticate extracts and validates authentication from request
func (am *AuthMiddleware) Authenticate(ctx context.Context, token string) (*AuthContext, error) {
	if token == "" {
		return nil, fmt.Errorf("missing authentication token")
	}

	// If auth is not required, allow with limited scopes
	if !am.requireAuth {
		return &AuthContext{
			UserID:    "anonymous",
			Roles:     []string{"user"},
			Scopes:    []string{"read:events", "list:events"},
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}, nil
	}

	// Extract actual token (remove Bearer prefix if present)
	actualToken := token
	if len(token) > 7 && token[:7] == "Bearer " {
		actualToken = token[7:]
	}

	if len(actualToken) < 3 {
		am.logWarn("Token too short")
		am.recordCounter("graphql_auth_invalid_token", 1)
		return nil, fmt.Errorf("invalid token")
	}

	// Use real token validator if available
	if am.tokenValidator != nil {
		// Try JWT token first
		result := am.tokenValidator.ValidateToken(ctx, token)
		if result.Valid {
			authCtx := &AuthContext{
				UserID:    result.UserID,
				Roles:     result.Roles,
				Scopes:    result.Permissions,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}
			am.logInfo("User authenticated via JWT", "userId", authCtx.UserID)
			am.recordCounter("graphql_auth_success", 1)
			return authCtx, nil
		}

		// Try API key
		result = am.tokenValidator.ValidateAPIKey(ctx, actualToken)
		if result.Valid {
			authCtx := &AuthContext{
				UserID:    result.UserID,
				Roles:     result.Roles,
				Scopes:    result.Permissions,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}
			am.logInfo("User authenticated via API key", "userId", authCtx.UserID)
			am.recordCounter("graphql_auth_success", 1)
			return authCtx, nil
		}

		am.logWarn("Authentication failed", "error", result.Error)
		am.recordCounter("graphql_auth_failed", 1)
		return nil, fmt.Errorf("authentication failed: %s", result.Error)
	}

	// No validator configured — reject in production mode
	am.logWarn("No token validator configured, authentication denied")
	am.recordCounter("graphql_auth_no_validator", 1)
	return nil, fmt.Errorf("authentication not configured")
}

func (am *AuthMiddleware) logInfo(msg string, args ...any) {
	if am.logger != nil {
		am.logger.Info(msg, args...)
	}
}

func (am *AuthMiddleware) logWarn(msg string, args ...any) {
	if am.logger != nil {
		am.logger.Warn(msg, args...)
	}
}

func (am *AuthMiddleware) recordCounter(name string, value int64) {
	if am.metrics != nil {
		am.metrics.RecordCounter(name, value, nil)
	}
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

// calculateComplexity calculates query complexity by walking the parsed AST.
// Each field selection costs 1 point, each level of nesting multiplies by a
// depth factor, and list fields (edges, events) carry an additional multiplier.
func (cm *ComplexityMiddleware) calculateComplexity(query string) int {
	src := source.Source{Body: []byte(query)}
	doc, err := parser.Parse(parser.ParseParams{Source: &src})
	if err != nil {
		// If the query cannot be parsed, fall back to character-based estimate
		return len(query)/10 + 1
	}

	total := 0
	for _, def := range doc.Definitions {
		if op, ok := def.(*ast.OperationDefinition); ok {
			total += cm.walkSelectionSet(op.GetSelectionSet(), 1)
		}
		if frag, ok := def.(*ast.FragmentDefinition); ok {
			total += cm.walkSelectionSet(frag.GetSelectionSet(), 1)
		}
	}
	if total == 0 {
		return 1
	}
	return total
}

// walkSelectionSet recursively walks a GraphQL selection set, accumulating
// complexity. Each field costs 1, nested fields are multiplied by depthFactor,
// and known list fields carry a listMultiplier.
func (cm *ComplexityMiddleware) walkSelectionSet(selSet *ast.SelectionSet, depth int) int {
	if selSet == nil {
		return 0
	}

	const depthFactor = 2     // each nesting level doubles cost
	const listMultiplier = 10 // list fields (edges, events) cost 10x

	cost := 0
	for _, sel := range selSet.Selections {
		switch s := sel.(type) {
		case *ast.Field:
			fieldCost := 1
			// Known list fields that return arrays
			name := s.Name.Value
			if name == "edges" || name == "events" || name == "nodes" ||
				name == "transactions" || name == "logs" {
				fieldCost *= listMultiplier
			}
			// Deeply nested fields are exponentially more expensive
			if depth > 1 {
				fieldCost *= depthFactor
			}
			cost += fieldCost
			if s.SelectionSet != nil {
				cost += cm.walkSelectionSet(s.SelectionSet, depth+1)
			}
		case *ast.InlineFragment:
			if s.SelectionSet != nil {
				cost += cm.walkSelectionSet(s.SelectionSet, depth)
			}
		case *ast.FragmentSpread:
			// Fragment spreads are counted as their field count (estimate)
			cost += 5
		}
	}
	return cost
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
func (cm *CachingMiddleware) GetCached(query string) (any, error) {
	if cm.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	cacheKey := fmt.Sprintf("graphql:query:%s", shared.HashCacheKey(query))
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
func (cm *CachingMiddleware) SetCached(query string, result any) error {
	if cm.cache == nil {
		return fmt.Errorf("cache not available")
	}

	cacheKey := fmt.Sprintf("graphql:query:%s", shared.HashCacheKey(query))
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
	lm.logger.Info(
		"GraphQL query executed",
		"userId", userID,
		"queryLength", len(query),
		"duration", duration.Milliseconds(),
	)
}

// LogMutation logs a GraphQL mutation
func (lm *LoggingMiddleware) LogMutation(userID string, mutation string, duration time.Duration) {
	lm.logger.Info(
		"GraphQL mutation executed",
		"userId", userID,
		"mutationLength", len(mutation),
		"duration", duration.Milliseconds(),
	)
}

// LogError logs a GraphQL error
func (lm *LoggingMiddleware) LogError(userID string, operation string, err error) {
	lm.logger.Error(
		"GraphQL operation failed",
		"userId", userID,
		"operation", operation,
		"error", err.Error(),
	)
}
