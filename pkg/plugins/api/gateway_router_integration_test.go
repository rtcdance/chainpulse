package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test Gateway Router Integration Initialization
func TestGatewayRouterIntegrationInitialization(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)

	if integration == nil {
		t.Errorf("Expected integration to be created")
	}

	err := integration.Initialize(context.Background())
	if err != nil {
		t.Errorf("Failed to initialize integration: %v", err)
	}

	// Initialize again should not error
	err = integration.Initialize(context.Background())
	if err != nil {
		t.Errorf("Failed to initialize integration again: %v", err)
	}
}

// Test Route Registration
func TestGatewayRouterIntegrationRouteRegistration(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		_ = err // Log but continue
	}

	router := integration.GetRouter()

	// Verify routes are registered
	routes := router.GetRoutes()
	if len(routes) == 0 {
		t.Errorf("Expected routes to be registered")
	}

	// Verify specific routes
	expectedRoutes := []string{
		"event-query",
		"event-by-id",
		"event-by-chain",
		"event-by-contract",
		"event-by-name",
		"subscribe",
		"subscribe-chain",
		"subscribe-contract",
		"subscribe-name",
		"health",
		"ready",
		"live",
		"components",
	}

	for _, expectedRoute := range expectedRoutes {
		route, err := router.GetRoute(expectedRoute)
		if err != nil {
			t.Errorf("Expected route %s to be registered", expectedRoute)
		}

		if route.ID != expectedRoute {
			t.Errorf("Expected route ID %s, got %s", expectedRoute, route.ID)
		}
	}
}

// Test Request Handling
func TestGatewayRouterIntegrationRequestHandling(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		_ = err // Log but continue
	}

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	integration.HandleRequest(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// Test Route Matching
func TestGatewayRouterIntegrationRouteMatching(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		_ = err // Log but continue
	}

	router := integration.GetRouter()

	tests := []struct {
		path      string
		routeID   string
		paramKey  string
		paramVal  string
	}{
		{"/events", "event-query", "", ""},
		{"/events/123", "event-by-id", "id", "123"},
		{"/events/chain/1", "event-by-chain", "chainId", "1"},
		{"/events/contract/0x123", "event-by-contract", "address", "0x123"},
		{"/events/name/Transfer", "event-by-name", "eventName", "Transfer"},
		{"/health", "health", "", ""},
		{"/health/ready", "ready", "", ""},
		{"/health/live", "live", "", ""},
		{"/health/components", "components", "", ""},
	}

	for _, test := range tests {
		route, params, err := router.MatchRoute(test.path)
		if err != nil {
			t.Errorf("Failed to match route for path %s: %v", test.path, err)
		}

		if route.ID != test.routeID {
			t.Errorf("Expected route %s, got %s", test.routeID, route.ID)
		}

		if test.paramKey != "" {
			if params[test.paramKey] != test.paramVal {
				t.Errorf("Expected param %s=%s, got %s", test.paramKey, test.paramVal, params[test.paramKey])
			}
		}
	}
}

// Test 404 Handling
func TestGatewayRouterIntegration404Handling(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	_ = integration.Initialize(context.Background())

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	integration.HandleRequest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test Metrics Collection
func TestGatewayRouterIntegrationMetrics(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	_ = integration.Initialize(context.Background())

	gatewayMetrics := integration.GetMetrics()

	if gatewayMetrics == nil {
		t.Errorf("Expected metrics to be returned")
	}

	if gatewayMetrics["route_count"] != 13 {
		t.Errorf("Expected 13 routes, got %v", gatewayMetrics["route_count"])
	}
}

// Test Response Writer
func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.WriteHeader(http.StatusOK)
	if _, err := rw.Write([]byte("test")); err != nil {
		_ = err // Log but continue
	}

	if rw.GetStatusCode() != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rw.GetStatusCode())
	}

	if string(rw.GetBody()) != "test" {
		t.Errorf("Expected body 'test', got '%s'", string(rw.GetBody()))
	}
}

// Test Request Logger
func TestRequestLogger(t *testing.T) {
	logger := &MockLogger{}
	rl := NewRequestLogger(logger)

	req := httptest.NewRequest("GET", "/test", nil)
	rl.LogRequest(req, http.StatusOK, 100*time.Millisecond)

	if len(logger.messages) == 0 {
		t.Errorf("Expected log message to be recorded")
	}

	rl.LogError(req, fmt.Errorf("test error"))

	if len(logger.messages) < 2 {
		t.Errorf("Expected error log message to be recorded")
	}
}

// Test Request Router Middleware
func TestRequestRouterMiddleware(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	router := NewRequestRouter(logger, metrics)
	if err := router.Initialize(context.Background()); err != nil {
		_ = err // Log but continue
	}

	route := NewRoute("test", "/test", "GET")
	if err := router.RegisterRoute(route); err != nil {
		_ = err // Log but continue
	}

	middleware := NewRequestRouterMiddleware(router, logger)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, ok := GetRouteFromContext(r)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if route.ID != "test" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	wrappedHandler := middleware.Middleware(testHandler)

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test Close
func TestGatewayRouterIntegrationClose(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	_ = integration.Initialize(context.Background())

	err := integration.Close(context.Background())
	if err != nil {
		t.Errorf("Failed to close integration: %v", err)
	}

	// Close again should not error
	err = integration.Close(context.Background())
	if err != nil {
		t.Errorf("Failed to close integration again: %v", err)
	}
}

// Test Concurrent Requests
func TestGatewayRouterIntegrationConcurrentRequests(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("failed to initialize integration: %v", err)
	}

	// Send concurrent requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			integration.HandleRequest(w, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test Route Not Found
func TestGatewayRouterIntegrationRouteNotFound(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("failed to initialize integration: %v", err)
	}

	req := httptest.NewRequest("GET", "/invalid/path", nil)
	w := httptest.NewRecorder()

	integration.HandleRequest(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test Uninitialized Gateway
func TestGatewayRouterIntegrationUninitialized(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	integration.HandleRequest(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
