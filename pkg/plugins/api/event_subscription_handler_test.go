package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"chainpulse/pkg/core"
)

func TestEventSubscriptionHandlerRateLimitsHandshakeWithoutContext(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	handler.initialized = true
	handler.SetRateLimiter(NewRateLimiter(logger, metrics, &RateLimitConfig{
		DefaultRequestsPerSecond: 1,
		DefaultBurstSize:         1,
		CleanupInterval:          5 * time.Minute,
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/events/subscribe", nil)
	req1.Header.Set("Connection", "Upgrade")
	req1.Header.Set("Upgrade", "websocket")
	req1.Header.Set("Sec-WebSocket-Version", "13")
	req1.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec1 := httptest.NewRecorder()
	handler.HandleSubscribeAll(rec1, req1)
	if rec1.Code == http.StatusTooManyRequests {
		t.Fatalf("expected first handshake attempt to reach upgrade path, got %d", rec1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/events/subscribe", nil)
	req2.Header.Set("Connection", "Upgrade")
	req2.Header.Set("Upgrade", "websocket")
	req2.Header.Set("Sec-WebSocket-Version", "13")
	req2.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec2 := httptest.NewRecorder()
	handler.HandleSubscribeAll(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second handshake to be rate limited, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestEventSubscriptionHandlerSkipsDirectRateLimitWhenContextAlreadyLimited(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	handler.initialized = true
	handler.SetRateLimiter(NewRateLimiter(logger, metrics, &RateLimitConfig{
		DefaultRequestsPerSecond: 1,
		DefaultBurstSize:         1,
		CleanupInterval:          5 * time.Minute,
	}))

	req := httptest.NewRequest(http.MethodGet, "/events/subscribe", nil)
	req = req.WithContext(RateLimitContext(req.Context(), &RateLimitInfo{Allowed: true}))
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec := httptest.NewRecorder()
	handler.HandleSubscribeAll(rec, req)

	if rec.Code == http.StatusTooManyRequests {
		t.Fatalf("expected context-limited handshake to skip direct limiter, got %d", rec.Code)
	}
}
