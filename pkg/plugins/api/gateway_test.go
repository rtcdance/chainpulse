package api

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
)

func TestAPIGatewayPluginDomainBridgeToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsDomainBridgeEnabled() {
		t.Fatal("expected domain bridge to be disabled by default")
	}

	plugin.SetDomainQueryService(nil)
	if plugin.IsDomainBridgeEnabled() {
		t.Fatal("expected domain bridge to remain disabled when service is nil")
	}
}

func TestAPIGatewayPluginDomainBridgeEnabledByUpstreamQueryEndpoints(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	plugin.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081"})
	if !plugin.IsDomainBridgeEnabled() {
		t.Fatal("expected upstream query endpoints to satisfy domain bridge signal")
	}
}

func TestAPIGatewayPluginEventQueryHandlerToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsEventQueryHandlerEnabled() {
		t.Fatal("expected event query handler to be disabled by default")
	}

	plugin.SetEventQueryHandler(nil)
	if plugin.IsEventQueryHandlerEnabled() {
		t.Fatal("expected event query handler to remain disabled when handler is nil")
	}
}

func TestAPIGatewayPluginRuntimeRoutesEnabledWhenHandlersWired(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	plugin.SetEventQueryHandler(NewEventQueryHandler(nil, logger, metrics))
	plugin.SetEventSubscriptionHandler(NewEventSubscriptionHandler(nil, logger, metrics))
	plugin.SetHealthCheckHandler(NewHealthCheckHandler(nil, logger, metrics))

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	if !plugin.IsRuntimeRoutesEnabled() {
		t.Fatal("expected runtime routes to be enabled when handlers are wired")
	}
}

func TestAPIGatewayPluginEventSubscriptionHandlerToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsEventSubscriptionHandlerEnabled() {
		t.Fatal("expected event subscription handler to be disabled by default")
	}

	plugin.SetEventSubscriptionHandler(nil)
	if plugin.IsEventSubscriptionHandlerEnabled() {
		t.Fatal("expected event subscription handler to remain disabled when handler is nil")
	}
}

func TestAPIGatewayPluginInitializeInjectsRateLimiterIntoSubscriptionHandler(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	limiter := NewRateLimiter(logger, metrics, &RateLimitConfig{
		DefaultRequestsPerSecond: 10,
		DefaultBurstSize:         10,
	})

	plugin.SetEventSubscriptionHandler(subscriptionHandler)
	plugin.SetRateLimitMiddleware(NewRateLimitMiddleware(limiter, logger))

	if err := plugin.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize plugin: %v", err)
	}

	if subscriptionHandler.rateLimiter != limiter {
		t.Fatal("expected gateway initialization to inject the shared rate limiter into subscription handler")
	}
}

func TestAPIGatewayPluginHealthCheckHandlerToggle(t *testing.T) {
	t.Parallel()
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	plugin := NewAPIGatewayPlugin(logger, metrics)

	if plugin.IsHealthCheckHandlerEnabled() {
		t.Fatal("expected health check handler to be disabled by default")
	}

	plugin.SetHealthCheckHandler(nil)
	if plugin.IsHealthCheckHandlerEnabled() {
		t.Fatal("expected health check handler to remain disabled when handler is nil")
	}
}
