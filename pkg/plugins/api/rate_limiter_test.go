package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRateLimiter tests rate limiter initialization
func TestNewRateLimiter(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
		CleanupInterval:          5 * time.Minute,
	}

	limiter := NewRateLimiter(logger, metrics, config)

	require.NotNil(t, limiter)
	assert.Equal(t, 100.0, limiter.defaultRequestsPerSecond)
	assert.Equal(t, 10, limiter.defaultBurstSize)
}

// TestNewRateLimiterWithNilConfig tests rate limiter with nil config
func TestNewRateLimiterWithNilConfig(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	limiter := NewRateLimiter(logger, metrics, nil)

	require.NotNil(t, limiter)
	assert.Equal(t, 100.0, limiter.defaultRequestsPerSecond)
	assert.Equal(t, 10, limiter.defaultBurstSize)
}

// TestNewTokenBucket tests token bucket initialization
func TestNewTokenBucket(t *testing.T) {
	t.Parallel()
	tb := NewTokenBucket(10.0, 5.0)

	require.NotNil(t, tb)
	assert.Equal(t, 5.0, tb.tokens)
	assert.Equal(t, 5.0, tb.maxTokens)
	assert.Equal(t, 10.0, tb.refillRate)
	assert.Equal(t, int64(0), tb.requestCount)
	assert.Equal(t, int64(0), tb.rejectedCount)
}

// TestTokenBucketAllow tests token bucket allow
func TestTokenBucketAllow(t *testing.T) {
	t.Parallel()
	tb := NewTokenBucket(10.0, 5.0)

	// Should allow first 5 requests
	for i := 0; i < 5; i++ {
		assert.True(t, tb.Allow())
	}

	// Should reject 6th request
	assert.False(t, tb.Allow())

	assert.Equal(t, int64(5), tb.requestCount)
	assert.Equal(t, int64(1), tb.rejectedCount)
}

// TestTokenBucketRefill tests token bucket refill
func TestTokenBucketRefill(t *testing.T) {
	t.Parallel()
	tb := NewTokenBucket(10.0, 5.0)

	// Use all tokens
	for i := 0; i < 5; i++ {
		tb.Allow()
	}

	// Wait for refill
	time.Sleep(200 * time.Millisecond)

	// Should have tokens again
	assert.True(t, tb.Allow())
}

// TestTokenBucketGetAvailableTokens tests getting available tokens
func TestTokenBucketGetAvailableTokens(t *testing.T) {
	t.Parallel()
	tb := NewTokenBucket(10.0, 5.0)

	available := tb.GetAvailableTokens()
	assert.InDelta(t, 5.0, available, 0.0001)

	tb.Allow()
	available = tb.GetAvailableTokens()
	assert.InDelta(t, 4.0, available, 0.0001)
}

// TestTokenBucketGetStats tests token bucket statistics
func TestTokenBucketGetStats(t *testing.T) {
	t.Parallel()
	tb := NewTokenBucket(10.0, 5.0)

	tb.Allow()
	tb.Allow()
	tb.Allow()
	tb.Allow()
	tb.Allow()
	tb.Allow() // This should fail

	stats := tb.GetStats()

	assert.Equal(t, int64(5), stats["request_count"])
	assert.Equal(t, int64(1), stats["rejected_count"])
	assert.Equal(t, 10.0, stats["refill_rate"])
	assert.Equal(t, 5.0, stats["max_tokens"])
}

// TestAllowRequest tests allow request
func TestAllowRequest(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
	}

	limiter := NewRateLimiter(logger, metrics, config)
	req := httptest.NewRequest("GET", "/test", nil)

	allowed, info := limiter.AllowRequest(req, "client1")

	assert.True(t, allowed)
	assert.True(t, info.Allowed)
	assert.Greater(t, info.RequestsRemaining, 0)
}

// TestAllowRequestExceeded tests rate limit exceeded
func TestAllowRequestExceeded(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 0.01,
		DefaultBurstSize:         2,
		CleanupInterval:          5 * time.Minute,
	}

	limiter := NewRateLimiter(logger, metrics, config)
	req := httptest.NewRequest("GET", "/test", nil)

	// Use up all tokens
	limiter.AllowRequest(req, "client1")
	limiter.AllowRequest(req, "client1")

	// This should be rejected
	allowed, info := limiter.AllowRequest(req, "client1")

	assert.False(t, allowed)
	assert.False(t, info.Allowed)
}

// TestSetEndpointLimit tests setting endpoint limit
func TestSetEndpointLimit(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, nil)

	limiter.SetEndpointLimit("/api/users", 50.0, 5)

	assert.NotNil(t, limiter.endpointLimiters["/api/users"])
}

// TestSetClientLimit tests setting client limit
func TestSetClientLimit(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, nil)

	limiter.SetClientLimit("client1", 50.0, 5)

	assert.NotNil(t, limiter.clientLimiters["client1"])
}

// TestSetIPLimit tests setting IP limit
func TestSetIPLimit(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, nil)

	limiter.SetIPLimit("192.168.1.1", 50.0, 5)

	assert.NotNil(t, limiter.ipLimiters["192.168.1.1"])
}

// TestRateLimiterGetStats tests getting rate limiter statistics
func TestRateLimiterGetStats(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, nil)

	limiter.SetEndpointLimit("/api/users", 50.0, 5)
	limiter.SetClientLimit("client1", 50.0, 5)
	limiter.SetIPLimit("192.168.1.1", 50.0, 5)

	stats := limiter.GetStats()

	assert.Equal(t, 1, stats["total_endpoints"])
	assert.Equal(t, 1, stats["total_clients"])
	assert.Equal(t, 1, stats["total_ips"])
}

// TestGetClientIP tests extracting client IP
func TestGetClientIP(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		headers  map[string]string
		remoteIP string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			remoteIP: "127.0.0.1:8080",
			// Without trustedProxies, X-Forwarded-For is ignored; falls back to RemoteAddr
			expected: "127.0.0.1",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteIP: "127.0.0.1:8080",
			expected: "192.168.1.2",
		},
		{
			name:     "RemoteAddr",
			headers:  map[string]string{},
			remoteIP: "192.168.1.3:8080",
			expected: "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteIP

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := getClientIP(req)
			assert.Equal(t, tt.expected, ip)
		})
	}
}

// TestExtractClientID tests extracting client ID
func TestExtractClientID(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "Authorization header",
			headers:  map[string]string{"Authorization": "Bearer token123"},
			expected: "Bearer token123",
		},
		{
			name:     "X-API-Key header",
			headers:  map[string]string{"X-API-Key": "api-key-123"},
			expected: "api-key-123",
		},
		{
			name:     "No headers",
			headers:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			clientID := extractClientID(req)
			assert.Equal(t, tt.expected, clientID)
		})
	}
}

// TestConcurrentRequests tests concurrent requests
func TestConcurrentRequests(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         50,
	}

	limiter := NewRateLimiter(logger, metrics, config)

	var wg sync.WaitGroup
	allowedCount := 0
	rejectedCount := 0
	mu := sync.Mutex{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/test", nil)
			allowed, _ := limiter.AllowRequest(req, fmt.Sprintf("client%d", id%10))

			mu.Lock()
			if allowed {
				allowedCount++
			} else {
				rejectedCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	assert.Greater(t, allowedCount, 0)
	assert.Equal(t, 100, allowedCount+rejectedCount)
}

// TestEndpointSpecificLimit tests endpoint-specific rate limiting
func TestEndpointSpecificLimit(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
		EndpointLimits: map[string]EndpointLimit{
			"/api/expensive": {
				Path:              "/api/expensive",
				RequestsPerSecond: 1.0,
				BurstSize:         2,
			},
		},
	}

	limiter := NewRateLimiter(logger, metrics, config)

	// Make requests to the limited endpoint
	req1 := httptest.NewRequest("GET", "/api/expensive", nil)
	req2 := httptest.NewRequest("GET", "/api/expensive", nil)
	req3 := httptest.NewRequest("GET", "/api/expensive", nil)

	allowed1, _ := limiter.AllowRequest(req1, "client1")
	allowed2, _ := limiter.AllowRequest(req2, "client1")
	allowed3, _ := limiter.AllowRequest(req3, "client1")

	assert.True(t, allowed1)
	assert.True(t, allowed2)
	assert.False(t, allowed3)
}

// TestClientSpecificLimit tests client-specific rate limiting
func TestClientSpecificLimit(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
		ClientLimits: map[string]ClientLimit{
			"premium-client": {
				ClientID:          "premium-client",
				RequestsPerSecond: 1000.0,
				BurstSize:         100,
			},
		},
	}

	limiter := NewRateLimiter(logger, metrics, config)

	req := httptest.NewRequest("GET", "/test", nil)

	// Premium client should have higher limit
	allowed, _ := limiter.AllowRequest(req, "premium-client")
	assert.True(t, allowed)
}

// TestRateLimitContext tests rate limit context
func TestRateLimitContext(t *testing.T) {
	t.Parallel()
	info := &RateLimitInfo{
		Allowed:           true,
		RequestsRemaining: 100,
		ResetTime:         time.Now().Add(1 * time.Second),
	}

	ctx := RateLimitContext(context.Background(), info)
	retrievedInfo, ok := GetRateLimitFromContext(ctx)

	assert.True(t, ok)
	assert.Equal(t, info, retrievedInfo)
}

func TestRateLimitMiddlewareInjectsContext(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, &RateLimitConfig{
		DefaultRequestsPerSecond: 10,
		DefaultBurstSize:         10,
	})
	middleware := NewRateLimitMiddleware(limiter, logger)

	var observed bool
	handler := middleware.Middleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, observed = GetRateLimitFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !observed {
		t.Fatal("expected rate limit info to be injected into request context")
	}
}

// TestRateLimitHeaders tests rate limit headers in response
func TestRateLimitHeaders(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
	}

	limiter := NewRateLimiter(logger, metrics, config)
	middleware := NewRateLimitMiddleware(limiter, logger)

	handler := middleware.Middleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

// TestRateLimitExceededResponse tests rate limit exceeded response
func TestRateLimitExceededResponse(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 0.01,
		DefaultBurstSize:         1,
		CleanupInterval:          5 * time.Minute,
	}

	limiter := NewRateLimiter(logger, metrics, config)
	middleware := NewRateLimitMiddleware(limiter, logger)

	handler := middleware.Middleware(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

// TestCleanup tests cleanup of stale limiters
func TestCleanup(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
		CleanupInterval:          1 * time.Millisecond,
	}

	limiter := NewRateLimiter(logger, metrics, config)

	// Add some IP limiters
	limiter.SetIPLimit("192.168.1.1", 100.0, 10)
	limiter.SetIPLimit("192.168.1.2", 100.0, 10)

	// Wait for cleanup interval
	time.Sleep(10 * time.Millisecond)

	// Make a request to trigger cleanup
	req := httptest.NewRequest("GET", "/test", nil)
	limiter.AllowRequest(req, "client1")

	// Verify cleanup occurred
	assert.Less(t, len(limiter.ipLimiters), 3)
}

// TestTokenBucketConcurrency tests token bucket thread safety
func TestTokenBucketConcurrency(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping concurrency stress test in short mode")
	}

	tb := NewTokenBucket(1000.0, 100.0)

	var wg sync.WaitGroup
	allowedCount := 0
	rejectedCount := 0
	mu := sync.Mutex{}

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if tb.Allow() {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			} else {
				mu.Lock()
				rejectedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, allowedCount)
	assert.Equal(t, 100, rejectedCount)
}

// TestRateLimiterConcurrency tests rate limiter thread safety
func TestRateLimiterConcurrency(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 1000.0,
		DefaultBurstSize:         100,
	}

	limiter := NewRateLimiter(logger, metrics, config)

	var wg sync.WaitGroup
	allowedCount := 0
	rejectedCount := 0
	mu := sync.Mutex{}

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/test", nil)
			allowed, _ := limiter.AllowRequest(req, fmt.Sprintf("client%d", id%10))

			mu.Lock()
			if allowed {
				allowedCount++
			} else {
				rejectedCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	assert.Greater(t, allowedCount, 0)
	assert.Equal(t, 200, allowedCount+rejectedCount)
}

// TestMultipleEndpointLimits tests multiple endpoint limits
func TestMultipleEndpointLimits(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 100.0,
		DefaultBurstSize:         10,
		EndpointLimits: map[string]EndpointLimit{
			"/api/users":    {Path: "/api/users", RequestsPerSecond: 50.0, BurstSize: 5},
			"/api/posts":    {Path: "/api/posts", RequestsPerSecond: 100.0, BurstSize: 10},
			"/api/comments": {Path: "/api/comments", RequestsPerSecond: 200.0, BurstSize: 20},
		},
	}

	limiter := NewRateLimiter(logger, metrics, config)

	assert.Equal(t, 3, len(limiter.endpointLimiters))
}

// TestDynamicLimitAdjustment tests dynamic limit adjustment
func TestDynamicLimitAdjustment(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	limiter := NewRateLimiter(logger, metrics, nil)

	// Set initial limit
	limiter.SetClientLimit("client1", 10.0, 5)

	// Adjust limit
	limiter.SetClientLimit("client1", 50.0, 10)

	stats := limiter.GetStats()
	clientStats := stats["client_limiters"].(map[string]any)
	client1Stats := clientStats["client1"].(map[string]any)

	assert.Equal(t, 50.0, client1Stats["refill_rate"])
	assert.Equal(t, 10.0, client1Stats["max_tokens"])
}

func TestRequestsPerMinuteToPerSecond(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, RequestsPerMinuteToPerSecond(0))
	assert.InDelta(t, 2.0, RequestsPerMinuteToPerSecond(120), 0.0001)
	assert.InDelta(t, 1000.0/60.0, RequestsPerMinuteToPerSecond(1000), 0.0001)
}

func TestBurstSizeFromRequestsPerMinute(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 10, BurstSizeFromRequestsPerMinute(1))
	assert.Equal(t, 10, BurstSizeFromRequestsPerMinute(42))
	assert.Equal(t, 20, BurstSizeFromRequestsPerMinute(120))
}
