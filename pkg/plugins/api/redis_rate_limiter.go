package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rtcdance/chainpulse/pkg/core"

	redisv9 "github.com/redis/go-redis/v9"
)

var errRedisUnavailable = fmt.Errorf("redis unavailable")

// RedisRateLimiter implements distributed rate limiting backed by Redis.
// It uses a sliding-window counter algorithm executed atomically via a Lua
// script, making it safe for multi-pod deployments behind a shared Redis.
//
// If Redis becomes unreachable the limiter falls back to a per-process
// in-memory token bucket so the service degrades gracefully rather than
// rejecting all traffic or allowing unlimited access.
type RedisRateLimiter struct {
	client *redisv9.Client

	logger  core.Logger
	metrics core.MetricsCollector

	defaultRequestsPerSecond float64
	defaultBurstSize         int

	// Fallback in-memory limiter used when Redis is unavailable.
	fallback *RateLimiter

	// windowSize is the sliding window duration (1 second by default).
	windowSize time.Duration
}

// NewRedisRateLimiter creates a Redis-backed rate limiter.
// If redisClient is nil, it falls back to a pure in-memory limiter.
func NewRedisRateLimiter(
	redisClient *redisv9.Client,
	logger core.Logger,
	metrics core.MetricsCollector,
	config *RateLimitConfig,
) *RedisRateLimiter {
	if config == nil {
		config = &RateLimitConfig{
			DefaultRequestsPerSecond: 100.0,
			DefaultBurstSize:         10,
			CleanupInterval:          5 * time.Minute,
		}
	}

	rl := &RedisRateLimiter{
		client:                   redisClient,
		logger:                   logger,
		metrics:                  metrics,
		defaultRequestsPerSecond: config.DefaultRequestsPerSecond,
		defaultBurstSize:         config.DefaultBurstSize,
		windowSize:               time.Second,
		fallback:                 NewRateLimiter(logger, metrics, config),
	}

	return rl
}

// slidingWindowLua atomically increments a sliding-window counter and returns
// the current count. Keys:
//
//	KEYS[1] = current window key
//	KEYS[2] = previous window key
//
// Argv:
//
//	ARGV[1] = window size in milliseconds
//	ARGV[2] = current timestamp in milliseconds
//	ARGV[3] = max requests allowed in the window
//
// Returns: count (integer) or -1 on error.
var slidingWindowLua = redisv9.NewScript(`
local current_key   = KEYS[1]
local previous_key  = KEYS[2]
local window_ms     = tonumber(ARGV[1])
local now_ms        = tonumber(ARGV[2])
local max_requests  = tonumber(ARGV[3])

local current_count = tonumber(redis.call("GET", current_key) or "0")
local elapsed       = now_ms % window_ms
local weight        = (window_ms - elapsed) / window_ms

local previous_count = tonumber(redis.call("GET", previous_key) or "0")
local estimated      = math.floor(previous_count * weight + 0.5)
local total          = current_count + estimated

if total >= max_requests then
    return total
end

local new_count = redis.call("INCR", current_key)
if new_count == 1 then
    redis.call("PEXPIRE", current_key, window_ms * 2)
end

return new_count + estimated
`)

// AllowRequest checks if a request is allowed based on distributed rate limits.
// When Redis is unavailable, it gracefully degrades to the in-memory token-bucket
// limiter instead of allowing all traffic through (fail-open).
func (rl *RedisRateLimiter) AllowRequest(r *http.Request, clientID string) (bool, *RateLimitInfo) {
	if rl.client == nil {
		return rl.fallback.AllowRequest(r, clientID)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 50*time.Millisecond)
	defer cancel()

	info := &RateLimitInfo{
		Allowed:           true,
		RequestsRemaining: rl.defaultBurstSize,
		ResetTime:         time.Now().Add(rl.windowSize),
	}

	endpoint := r.URL.Path
	clientIP := getClientIP(r)

	// Check endpoint limit
	allowed, remaining, err := rl.checkSlidingWindow(ctx, "endpoint", endpoint, rl.defaultRequestsPerSecond)
	if err != nil {
		return rl.fallback.AllowRequest(r, clientID)
	}
	if !allowed {
		info.Allowed = false
		info.RetryAfter = rl.windowSize
		rl.metrics.RecordCounter("rate_limit_exceeded_endpoint", 1, nil)
		rl.logger.Warn("Rate limit exceeded for endpoint", "path", endpoint, "clientID", clientID)
		return false, info
	}
	info.RequestsRemaining = remaining

	// Check IP limit
	allowed, remaining, err = rl.checkSlidingWindow(ctx, "ip", clientIP, rl.defaultRequestsPerSecond)
	if err != nil {
		return rl.fallback.AllowRequest(r, clientID)
	}
	if !allowed {
		info.Allowed = false
		info.RetryAfter = rl.windowSize
		rl.metrics.RecordCounter("rate_limit_exceeded_ip", 1, nil)
		rl.logger.Warn("Rate limit exceeded for IP", "ip", clientIP, "clientID", clientID)
		return false, info
	}
	if remaining < info.RequestsRemaining {
		info.RequestsRemaining = remaining
	}

	// Check client limit
	if clientID != "" {
		allowed, remaining, err = rl.checkSlidingWindow(ctx, "client", clientID, rl.defaultRequestsPerSecond)
		if err != nil {
			return rl.fallback.AllowRequest(r, clientID)
		}
		if !allowed {
			info.Allowed = false
			info.RetryAfter = rl.windowSize
			rl.metrics.RecordCounter("rate_limit_exceeded_client", 1, nil)
			rl.logger.Warn("Rate limit exceeded for client", "clientID", clientID)
			return false, info
		}
		if remaining < info.RequestsRemaining {
			info.RequestsRemaining = remaining
		}
	}

	rl.metrics.RecordCounter("rate_limit_allowed", 1, nil)
	return true, info
}

// checkSlidingWindow runs the Lua sliding-window script for a given key.
// Returns (allowed, remaining, error). When error is errRedisUnavailable,
// the caller should fall back to the in-memory rate limiter.
func (rl *RedisRateLimiter) checkSlidingWindow(ctx context.Context, namespace, key string, rps float64) (bool, int, error) {
	now := time.Now()
	nowMs := now.UnixMilli()
	windowMs := rl.windowSize.Milliseconds()

	currentWindow := now.Truncate(rl.windowSize).UnixMilli()
	previousWindow := currentWindow - windowMs

	currentKey := fmt.Sprintf("rl:%s:%s:%d", namespace, key, currentWindow)
	previousKey := fmt.Sprintf("rl:%s:%s:%d", namespace, key, previousWindow)

	maxRequests := int64(rps * rl.windowSize.Seconds())
	if maxRequests < 1 {
		maxRequests = 1
	}

	result, err := slidingWindowLua.Run(ctx, rl.client, []string{currentKey, previousKey},
		windowMs, nowMs, maxRequests,
	).Int64()

	if err != nil {
		rl.metrics.RecordCounter("rate_limit_redis_error", 1, nil)
		rl.logger.Warn("Redis rate limit check failed, falling back to in-memory limiter", "error", err.Error())
		return false, 0, errRedisUnavailable
	}

	remaining := int(maxRequests - result)
	if remaining < 0 {
		remaining = 0
	}

	return result <= maxRequests, remaining, nil
}

// SetEndpointLimit configures a per-endpoint rate limit.
// The limit is stored in the fallback in-memory limiter; Redis limits
// use the default RPS unless overridden via Redis key conventions.
func (rl *RedisRateLimiter) SetEndpointLimit(path string, requestsPerSecond float64, burstSize int) {
	rl.fallback.SetEndpointLimit(path, requestsPerSecond, burstSize)
}

// SetClientLimit configures a per-client rate limit.
func (rl *RedisRateLimiter) SetClientLimit(clientID string, requestsPerSecond float64, burstSize int) {
	rl.fallback.SetClientLimit(clientID, requestsPerSecond, burstSize)
}

// SetIPLimit configures a per-IP rate limit.
func (rl *RedisRateLimiter) SetIPLimit(ip string, requestsPerSecond float64, burstSize int) {
	rl.fallback.SetIPLimit(ip, requestsPerSecond, burstSize)
}

// GetStats returns rate limiter statistics from the fallback in-memory limiter.
func (rl *RedisRateLimiter) GetStats() map[string]any {
	return rl.fallback.GetStats()
}
