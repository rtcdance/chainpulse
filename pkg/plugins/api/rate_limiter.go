package api

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"chainpulse/pkg/core"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// rateLimitInfoKey is the key for storing rate limit info in context.Context
	rateLimitInfoKey contextKey = "rate_limit_info"
	// routeContextKey is the key for storing route in context.Context
	routeContextKey contextKey = "route"
	// paramsContextKey is the key for storing route parameters in context.Context
	paramsContextKey contextKey = "params"
)

// RateLimiter implements rate limiting for API requests
type RateLimiter struct {
	// Per-client rate limiters (keyed by client identifier)
	clientLimiters map[string]*TokenBucket
	// Per-endpoint rate limiters (keyed by endpoint path)
	endpointLimiters map[string]*TokenBucket
	// Per-IP rate limiters (keyed by IP address)
	ipLimiters map[string]*TokenBucket

	logger  core.Logger
	metrics core.MetricsCollector

	// Configuration
	defaultRequestsPerSecond float64
	defaultBurstSize         int
	cleanupInterval          time.Duration
	lastCleanup              time.Time

	mu sync.RWMutex
}

// TokenBucket implements token bucket algorithm for rate limiting
type TokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	requestCount   int64
	rejectedCount  int64
	mu             sync.RWMutex
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Global settings
	DefaultRequestsPerSecond float64
	DefaultBurstSize         int

	// Per-endpoint settings
	EndpointLimits map[string]EndpointLimit

	// Per-client settings
	ClientLimits map[string]ClientLimit

	// Cleanup settings
	CleanupInterval time.Duration
}

// EndpointLimit defines rate limit for a specific endpoint
type EndpointLimit struct {
	Path                string
	RequestsPerSecond   float64
	BurstSize           int
	BypassAuthenticated bool
	BypassHealthChecks  bool
}

// ClientLimit defines rate limit for a specific client
type ClientLimit struct {
	ClientID          string
	RequestsPerSecond float64
	BurstSize         int
}

// RequestsPerMinuteToPerSecond converts an operator-facing req/min budget into
// the internal token-bucket refill rate used by the rate limiter.
func RequestsPerMinuteToPerSecond(requestsPerMinute int) float64 {
	if requestsPerMinute <= 0 {
		return 0
	}

	return float64(requestsPerMinute) / 60.0
}

// BurstSizeFromRequestsPerMinute derives a small bounded burst window from a
// req/min budget using roughly 10 seconds of allowance.
func BurstSizeFromRequestsPerMinute(requestsPerMinute int) int {
	if requestsPerMinute <= 0 {
		return 1
	}

	burst := int(math.Ceil(float64(requestsPerMinute) / 6.0))
	if burst < 10 {
		return 10
	}

	return burst
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(logger core.Logger, metrics core.MetricsCollector, config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = &RateLimitConfig{
			DefaultRequestsPerSecond: 100.0,
			DefaultBurstSize:         10,
			CleanupInterval:          5 * time.Minute,
		}
	}

	rl := &RateLimiter{
		clientLimiters:           make(map[string]*TokenBucket),
		endpointLimiters:         make(map[string]*TokenBucket),
		ipLimiters:               make(map[string]*TokenBucket),
		logger:                   logger,
		metrics:                  metrics,
		defaultRequestsPerSecond: config.DefaultRequestsPerSecond,
		defaultBurstSize:         config.DefaultBurstSize,
		cleanupInterval:          config.CleanupInterval,
		lastCleanup:              time.Now(),
	}

	// Initialize endpoint limiters
	if config.EndpointLimits != nil {
		for _, limit := range config.EndpointLimits {
			rl.endpointLimiters[limit.Path] = NewTokenBucket(
				limit.RequestsPerSecond,
				float64(limit.BurstSize),
			)
		}
	}

	// Initialize client limiters
	if config.ClientLimits != nil {
		for _, limit := range config.ClientLimits {
			rl.clientLimiters[limit.ClientID] = NewTokenBucket(
				limit.RequestsPerSecond,
				float64(limit.BurstSize),
			)
		}
	}

	return rl
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(refillRate float64, maxTokens float64) *TokenBucket {
	return &TokenBucket{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
		requestCount:   0,
		rejectedCount:  0,
	}
}

// AllowRequest checks if a request is allowed based on rate limits
func (rl *RateLimiter) AllowRequest(r *http.Request, clientID string) (bool, *RateLimitInfo) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Perform cleanup if needed
	if time.Since(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanup()
		rl.lastCleanup = time.Now()
	}

	info := &RateLimitInfo{
		Allowed:           true,
		RequestsRemaining: 0,
		ResetTime:         time.Now().Add(1 * time.Second),
	}

	// Check endpoint-specific rate limit
	if endpointLimit, ok := rl.endpointLimiters[r.URL.Path]; ok {
		if !endpointLimit.Allow() {
			info.Allowed = false
			rl.metrics.RecordCounter("rate_limit_exceeded_endpoint", 1, nil)
			rl.logger.Warn("Rate limit exceeded for endpoint",
				"path", r.URL.Path,
				"clientID", clientID,
			)
			return false, info
		}
		info.RequestsRemaining = int(endpointLimit.GetAvailableTokens())
	}

	// Check IP-based rate limit
	clientIP := getClientIP(r)
	if ipLimiter, ok := rl.ipLimiters[clientIP]; ok {
		if !ipLimiter.Allow() {
			info.Allowed = false
			rl.metrics.RecordCounter("rate_limit_exceeded_ip", 1, nil)
			rl.logger.Warn("Rate limit exceeded for IP",
				"ip", clientIP,
				"clientID", clientID,
			)
			return false, info
		}
		info.RequestsRemaining = int(ipLimiter.GetAvailableTokens())
	} else {
		// Create new IP limiter if not exists
		ipLimiter := NewTokenBucket(rl.defaultRequestsPerSecond, float64(rl.defaultBurstSize))
		if !ipLimiter.Allow() {
			info.Allowed = false
			rl.metrics.RecordCounter("rate_limit_exceeded_ip", 1, nil)
			return false, info
		}
		rl.ipLimiters[clientIP] = ipLimiter
		info.RequestsRemaining = int(ipLimiter.GetAvailableTokens())
	}

	// Check client-specific rate limit
	if clientID != "" {
		if clientLimiter, ok := rl.clientLimiters[clientID]; ok {
			if !clientLimiter.Allow() {
				info.Allowed = false
				rl.metrics.RecordCounter("rate_limit_exceeded_client", 1, nil)
				rl.logger.Warn("Rate limit exceeded for client",
					"clientID", clientID,
				)
				return false, info
			}
			info.RequestsRemaining = int(clientLimiter.GetAvailableTokens())
		} else {
			// Create new client limiter if not exists
			clientLimiter := NewTokenBucket(rl.defaultRequestsPerSecond, float64(rl.defaultBurstSize))
			if !clientLimiter.Allow() {
				info.Allowed = false
				rl.metrics.RecordCounter("rate_limit_exceeded_client", 1, nil)
				return false, info
			}
			rl.clientLimiters[clientID] = clientLimiter
			info.RequestsRemaining = int(clientLimiter.GetAvailableTokens())
		}
	}

	rl.metrics.RecordCounter("rate_limit_allowed", 1, nil)
	return true, info
}

// RateLimitInfo contains rate limit information for a request
type RateLimitInfo struct {
	Allowed           bool
	RequestsRemaining int
	ResetTime         time.Time
	RetryAfter        time.Duration
}

// Allow checks if a token is available in the bucket
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		tb.requestCount++
		return true
	}

	tb.rejectedCount++
	return false
}

// GetAvailableTokens returns the number of available tokens
func (tb *TokenBucket) GetAvailableTokens() float64 {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	tb.refill()
	return tb.tokens
}

// refill adds tokens based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()

	if elapsed > 0 {
		tokensToAdd := elapsed * tb.refillRate
		tb.tokens = min(tb.tokens+tokensToAdd, tb.maxTokens)
		tb.lastRefillTime = now
	}
}

// GetStats returns statistics for the token bucket
func (tb *TokenBucket) GetStats() map[string]interface{} {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	return map[string]interface{}{
		"available_tokens": tb.tokens,
		"max_tokens":       tb.maxTokens,
		"refill_rate":      tb.refillRate,
		"request_count":    tb.requestCount,
		"rejected_count":   tb.rejectedCount,
	}
}

// cleanup removes stale limiters
func (rl *RateLimiter) cleanup() {
	// Remove IP limiters that haven't been used recently
	for ip, limiter := range rl.ipLimiters {
		if limiter.requestCount == 0 && limiter.rejectedCount == 0 {
			delete(rl.ipLimiters, ip)
		}
	}

	// Remove client limiters that haven't been used recently
	for clientID, limiter := range rl.clientLimiters {
		if limiter.requestCount == 0 && limiter.rejectedCount == 0 {
			delete(rl.clientLimiters, clientID)
		}
	}
}

// SetEndpointLimit sets rate limit for a specific endpoint
func (rl *RateLimiter) SetEndpointLimit(path string, requestsPerSecond float64, burstSize int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.endpointLimiters[path] = NewTokenBucket(requestsPerSecond, float64(burstSize))
	rl.logger.Info("Endpoint rate limit set",
		"path", path,
		"requestsPerSecond", requestsPerSecond,
		"burstSize", burstSize,
	)
}

// SetClientLimit sets rate limit for a specific client
func (rl *RateLimiter) SetClientLimit(clientID string, requestsPerSecond float64, burstSize int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.clientLimiters[clientID] = NewTokenBucket(requestsPerSecond, float64(burstSize))
	rl.logger.Info("Client rate limit set",
		"clientID", clientID,
		"requestsPerSecond", requestsPerSecond,
		"burstSize", burstSize,
	)
}

// SetIPLimit sets rate limit for a specific IP address
func (rl *RateLimiter) SetIPLimit(ip string, requestsPerSecond float64, burstSize int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.ipLimiters[ip] = NewTokenBucket(requestsPerSecond, float64(burstSize))
	rl.logger.Info("IP rate limit set",
		"ip", ip,
		"requestsPerSecond", requestsPerSecond,
		"burstSize", burstSize,
	)
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	endpointStats := make(map[string]interface{})
	for path, limiter := range rl.endpointLimiters {
		endpointStats[path] = limiter.GetStats()
	}

	clientStats := make(map[string]interface{})
	for clientID, limiter := range rl.clientLimiters {
		clientStats[clientID] = limiter.GetStats()
	}

	ipStats := make(map[string]interface{})
	for ip, limiter := range rl.ipLimiters {
		ipStats[ip] = limiter.GetStats()
	}

	return map[string]interface{}{
		"endpoint_limiters": endpointStats,
		"client_limiters":   clientStats,
		"ip_limiters":       ipStats,
		"total_endpoints":   len(rl.endpointLimiters),
		"total_clients":     len(rl.clientLimiters),
		"total_ips":         len(rl.ipLimiters),
	}
}

// RateLimitMiddleware wraps an HTTP handler with rate limiting
type RateLimitMiddleware struct {
	limiter *RateLimiter
	logger  core.Logger
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(limiter *RateLimiter, logger core.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiter: limiter,
		logger:  logger,
	}
}

// Limiter returns the underlying rate limiter.
func (m *RateLimitMiddleware) Limiter() *RateLimiter {
	if m == nil {
		return nil
	}

	return m.limiter
}

// Middleware wraps an HTTP handler with rate limiting
func (m *RateLimitMiddleware) Middleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client ID from context or headers
			clientID := extractClientID(r)

			// Check rate limit
			allowed, info := limiter.AllowRequest(r, clientID)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", info.RequestsRemaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", info.ResetTime.Unix()))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(info.RetryAfter.Seconds())))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Call next handler
			next.ServeHTTP(w, r.WithContext(RateLimitContext(r.Context(), info)))
		})
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := parseIPList(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// parseIPList parses a comma-separated list of IPs
func parseIPList(ips string) []string {
	var result []string
	parts := strings.Split(ips, ",")
	for _, ip := range parts {
		trimmed := strings.TrimSpace(ip)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// extractClientID extracts client ID from request context or headers
func extractClientID(r *http.Request) string {
	// Check context first
	if clientID, ok := r.Context().Value("clientID").(string); ok {
		return clientID
	}

	// Check Authorization header
	if auth := r.Header.Get("Authorization"); auth != "" {
		return auth
	}

	// Check API key header
	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
		return apiKey
	}

	return ""
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimitContext adds rate limit information to request context
func RateLimitContext(ctx context.Context, info *RateLimitInfo) context.Context {
	return context.WithValue(ctx, rateLimitInfoKey, info)
}

// GetRateLimitFromContext retrieves rate limit info from context
func GetRateLimitFromContext(ctx context.Context) (*RateLimitInfo, bool) {
	info, ok := ctx.Value(rateLimitInfoKey).(*RateLimitInfo)
	return info, ok
}
