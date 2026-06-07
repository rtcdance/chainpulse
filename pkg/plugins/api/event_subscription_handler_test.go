package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
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

func TestEventSubscriptionHandler_SetTokenValidator(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	handler.SetTokenValidator(&TokenValidator{})
	if handler.tokenValidator == nil {
		t.Error("expected tokenValidator to be set")
	}
}

func TestEventSubscriptionHandler_SetAllowedOrigins(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	origins := []string{"https://example.com", "https://app.example.com"}
	handler.SetAllowedOrigins(origins)
	if len(handler.allowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(handler.allowedOrigins))
	}
}

func TestEventSubscriptionHandler_Initialize(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	t.Run("not initialized", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		if err := handler.Initialize(t.Context()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !handler.initialized {
			t.Error("expected handler to be initialized")
		}
	})

	t.Run("already initialized", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		handler.initialized = true
		if err := handler.Initialize(t.Context()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestEventSubscriptionHandler_GetConnectionCount(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	if count := handler.GetConnectionCount(); count != 0 {
		t.Errorf("expected 0 connections, got %d", count)
	}
}

func TestEventSubscriptionHandler_GetSubscriptionCount(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	if count := handler.GetSubscriptionCount(); count != 0 {
		t.Errorf("expected 0 subscriptions, got %d", count)
	}
}

func TestEventSubscriptionHandler_matchesSubscription(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)

	contractAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")

	tests := []struct {
		name      string
		event     *blockchain.BlockchainEvent
		sub       *Subscription
		wantMatch bool
	}{
		{
			name: "all matches anything",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "all"},
			wantMatch: true,
		},
		{
			name: "chain matches",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "chain", FilterValue: "1"},
			wantMatch: true,
		},
		{
			name: "chain does not match",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "chain", FilterValue: "56"},
			wantMatch: false,
		},
		{
			name: "contract matches",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "contract", FilterValue: contractAddr.Hex()},
			wantMatch: true,
		},
		{
			name: "contract does not match",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "contract", FilterValue: "0xDIFFERENT"},
			wantMatch: false,
		},
		{
			name: "name matches",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "name", FilterValue: "Transfer"},
			wantMatch: true,
		},
		{
			name: "name does not match",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "name", FilterValue: "Approval"},
			wantMatch: false,
		},
		{
			name: "unknown subscription type",
			event: &blockchain.BlockchainEvent{
				ChainID:         "1",
				ContractAddress: contractAddr,
				EventName:       "Transfer",
			},
			sub:       &Subscription{SubscriptionType: "unknown"},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := handler.matchesSubscription(tt.event, tt.sub)
			if got != tt.wantMatch {
				t.Errorf("matchesSubscription() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestEventSubscriptionHandler_checkOrigin(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	t.Run("no allowed origins, any origin ok", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://evil.com")
		if !handler.checkOrigin(req) {
			t.Error("expected origin to be allowed when no restrictions set")
		}
	})

	t.Run("allowed origin matches", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		handler.SetAllowedOrigins([]string{"https://example.com"})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		if !handler.checkOrigin(req) {
			t.Error("expected matching origin to be allowed")
		}
	})

	t.Run("disallowed origin rejected", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		handler.SetAllowedOrigins([]string{"https://example.com"})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://evil.com")
		if handler.checkOrigin(req) {
			t.Error("expected non-matching origin to be rejected")
		}
	})

	t.Run("no origin header with restrictions", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		handler.SetAllowedOrigins([]string{"https://example.com"})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if !handler.checkOrigin(req) {
			t.Error("expected missing origin to be allowed")
		}
	})
}

func TestEventSubscriptionHandler_Health(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	t.Run("not initialized", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		status := handler.Health(context.Background())
		if status.Status != "unhealthy" {
			t.Errorf("expected unhealthy status, got %q", status.Status)
		}
	})

	t.Run("initialized nil retrieval service", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		handler.initialized = true
		status := handler.Health(context.Background())
		if status.Status != "unhealthy" {
			t.Errorf("expected unhealthy status for nil service, got %q", status.Status)
		}
	})
}

func TestEventSubscriptionHandler_Close(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	t.Run("not initialized", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		if err := handler.Close(context.Background()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("initialized", func(t *testing.T) {
		t.Parallel()
		handler := NewEventSubscriptionHandler(nil, logger, metrics)
		handler.initialized = true
		if err := handler.Close(context.Background()); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if handler.initialized {
			t.Error("expected handler to be marked as not initialized after close")
		}
	})
}

func TestEventSubscriptionHandler_HandleSubscribeChain_InvalidChainID(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	handler.initialized = true

	req := httptest.NewRequest(http.MethodGet, "/events/subscribe/chain/invalid", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec := httptest.NewRecorder()
	handler.HandleSubscribeChain(rec, req, "invalid")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid chain ID, got %d", rec.Code)
	}
}

func TestEventSubscriptionHandler_HandleSubscribeContract_EmptyAddress(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	handler.initialized = true

	req := httptest.NewRequest(http.MethodGet, "/events/subscribe/contract/", nil)
	rec := httptest.NewRecorder()
	handler.HandleSubscribeContract(rec, req, "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty contract address, got %d", rec.Code)
	}
}

func TestEventSubscriptionHandler_HandleSubscribeName_EmptyName(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)
	handler.initialized = true

	req := httptest.NewRequest(http.MethodGet, "/events/subscribe/name/", nil)
	rec := httptest.NewRecorder()
	handler.HandleSubscribeName(rec, req, "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty event name, got %d", rec.Code)
	}
}

func TestEventSubscriptionHandler_HandleSubscribeAll_NotInitialized(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	handler := NewEventSubscriptionHandler(nil, logger, metrics)

	req := httptest.NewRequest(http.MethodGet, "/events/subscribe", nil)
	rec := httptest.NewRecorder()
	handler.HandleSubscribeAll(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for not initialized handler, got %d", rec.Code)
	}
}
