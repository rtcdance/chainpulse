package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRedisRateLimiter_AllowRequest(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 5,
		DefaultBurstSize:         5,
		CleanupInterval:          time.Minute,
	}

	rl := NewRedisRateLimiter(nil, logger, metrics, config)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	// Should allow within burst
	allowed, info := rl.AllowRequest(req, "client1")
	if !allowed {
		t.Error("first request should be allowed")
	}
	if info == nil {
		t.Fatal("info should not be nil")
	}
}

func TestRedisRateLimiter_FallbackOnNilRedis(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 2,
		DefaultBurstSize:         2,
		CleanupInterval:          time.Minute,
	}

	// nil Redis client → should use in-memory fallback
	rl := NewRedisRateLimiter(nil, logger, metrics, config)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.RemoteAddr = "10.0.0.1:12345"

	// Should exhaust the bucket
	for i := 0; i < 2; i++ {
		if allowed, _ := rl.AllowRequest(req, "test_client"); !allowed {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Third request should be rate limited
	allowed, _ := rl.AllowRequest(req, "test_client")
	if allowed {
		t.Error("request beyond burst should be rate limited")
	}
}

func TestRedisRateLimiter_SetEndpointLimit(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	rl := NewRedisRateLimiter(nil, logger, metrics, nil)
	rl.SetEndpointLimit("/graphql", 1, 1)

	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	req.RemoteAddr = "10.0.0.2:12345"

	// First request should pass
	if allowed, _ := rl.AllowRequest(req, ""); !allowed {
		t.Error("first request should be allowed")
	}
}

func TestRedisRateLimiter_GetStats(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	rl := NewRedisRateLimiter(nil, logger, metrics, nil)
	stats := rl.GetStats()

	if stats == nil {
		t.Fatal("stats should not be nil")
	}
}

func TestRedisRateLimiter_PerClientIsolation(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	config := &RateLimitConfig{
		DefaultRequestsPerSecond: 2,
		DefaultBurstSize:         2,
		CleanupInterval:          time.Minute,
	}

	rl := NewRedisRateLimiter(nil, logger, metrics, config)

	req1 := httptest.NewRequest(http.MethodGet, "/events", nil)
	req1.RemoteAddr = "10.0.0.10:12345"

	req2 := httptest.NewRequest(http.MethodGet, "/events", nil)
	req2.RemoteAddr = "10.0.0.20:12345"

	// Exhaust client1's bucket
	for i := 0; i < 2; i++ {
		if allowed, _ := rl.AllowRequest(req1, "client_A"); !allowed {
			t.Errorf("client_A request %d should be allowed", i+1)
		}
	}

	// client_A should now be limited
	if allowed, _ := rl.AllowRequest(req1, "client_A"); allowed {
		t.Error("client_A should be rate limited after exhausting bucket")
	}

	// client_B should still be allowed (separate bucket)
	if allowed, _ := rl.AllowRequest(req2, "client_B"); !allowed {
		t.Error("client_B should be allowed (independent of client_A)")
	}
}
