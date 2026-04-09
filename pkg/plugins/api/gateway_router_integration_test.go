package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeGatewayAPIV1Request(t *testing.T) {
	t.Run("strips prefix for nested route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/events?limit=5", nil)
		got := normalizeGatewayAPIV1Request(req)
		if got.URL.Path != "/events" {
			t.Fatalf("expected normalized path /events, got %q", got.URL.Path)
		}
		if got.URL.RawQuery != "limit=5" {
			t.Fatalf("expected query string to be preserved, got %q", got.URL.RawQuery)
		}
	})

	t.Run("keeps original when prefix not present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/events", nil)
		got := normalizeGatewayAPIV1Request(req)
		if got != req {
			t.Fatal("expected request pointer to be reused when no normalization is needed")
		}
	})
}

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
		"websocket-subscribe",
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
		path     string
		routeID  string
		paramKey string
		paramVal string
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

	if gatewayMetrics["route_count"] != 15 {
		t.Errorf("Expected 15 routes, got %v", gatewayMetrics["route_count"])
	}
}

func TestGatewayRouterIntegrationRuntimeSummaryRoute(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(
		logger,
		metrics,
		queryHandler,
		subscriptionHandler,
		healthHandler,
		func(r *http.Request) interface{} {
			_ = r
			return map[string]interface{}{
				"service":         "api-service",
				"runtime_mode":    "runtime-wired",
				"component_state": "healthy",
			}
		},
	)
	_ = integration.Initialize(context.Background())

	gatewayMetrics := integration.GetMetrics()
	if gatewayMetrics["route_count"] != 16 {
		t.Errorf("Expected 16 routes with runtime summary, got %v", gatewayMetrics["route_count"])
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	w := httptest.NewRecorder()
	integration.HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGatewayRouterIntegrationRejectsWrongMethodForRuntimeSummary(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(
		logger,
		metrics,
		queryHandler,
		subscriptionHandler,
		healthHandler,
		func(r *http.Request) interface{} {
			_ = r
			return map[string]interface{}{"service": "monolithic"}
		},
	)
	_ = integration.Initialize(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/runtime/summary", nil)
	w := httptest.NewRecorder()
	integration.HandleRequest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("expected Allow GET, got %q", got)
	}
}

func TestGatewayRouterIntegrationRejectsWrongMethodForEventQuery(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	_ = integration.Initialize(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/events", nil)
	w := httptest.NewRecorder()
	integration.HandleRequest(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("expected Allow GET, got %q", got)
	}
}

func TestGatewayRouterIntegrationRuntimeRouteInventory(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(
		logger,
		metrics,
		queryHandler,
		subscriptionHandler,
		healthHandler,
		func(r *http.Request) interface{} {
			_ = r
			return map[string]interface{}{"service": "monolithic"}
		},
		func(r *http.Request) interface{} {
			_ = r
			return map[string]interface{}{"counters": map[string]interface{}{}}
		},
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		},
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("replayed"))
		},
	)
	_ = integration.Initialize(context.Background())

	inventory := integration.GetRuntimeRouteInventory()
	if inventory.RegisteredRouteCount != 20 {
		t.Fatalf("expected 20 registered routes, got %d", inventory.RegisteredRouteCount)
	}
	if inventory.RuntimeRouteCount != 9 {
		t.Fatalf("expected 9 runtime routes, got %d", inventory.RuntimeRouteCount)
	}
	if inventory.RuntimeSurfaceCount != 5 {
		t.Fatalf("expected 5 runtime surfaces, got %d", inventory.RuntimeSurfaceCount)
	}
	if !inventory.HealthRoutesEnabled || !inventory.SummaryRouteEnabled || !inventory.MetricsRouteEnabled || !inventory.ControlRouteEnabled || !inventory.ReplayRouteEnabled {
		t.Fatalf("expected full runtime route inventory, got %+v", inventory)
	}
}

func TestGatewayRouterIntegrationRuntimeReplayRoute(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(
		logger,
		metrics,
		queryHandler,
		subscriptionHandler,
		healthHandler,
		nil,
		nil,
		nil,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"status":"accepted"}`))
		},
	)
	_ = integration.Initialize(context.Background())

	req := httptest.NewRequest(http.MethodPost, "/runtime/indexing/dlq/replay", strings.NewReader(`{"chain_id":"ethereum"}`))
	w := httptest.NewRecorder()
	integration.HandleRequest(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestGatewayRouterIntegrationSkipsBusinessRoutesWithoutHandlers(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, nil, nil, healthHandler)
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	router := integration.GetRouter()
	if _, err := router.GetRoute("event-query"); err == nil {
		t.Fatal("expected event-query route to be absent when no query handler or upstreams are configured")
	}
	if _, err := router.GetRoute("subscribe"); err == nil {
		t.Fatal("expected subscribe route to be absent when no subscription handler is configured")
	}
	if _, err := router.GetRoute("health"); err != nil {
		t.Fatalf("expected health route to remain registered: %v", err)
	}
}

func TestGatewayRouterIntegrationForwardsEventQueryToUpstream(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	integration.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081"})
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}
	integration.GetRouter().SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "http://api-service-1:8081/events?limit=5" {
				t.Fatalf("expected forwarded url http://api-service-1:8081/events?limit=5, got %s", r.URL.String())
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"data":[],"meta":{"source":"upstream-api-service"}}`)),
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	w := httptest.NewRecorder()
	integration.HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", got)
	}
	if body := w.Body.String(); body != `{"data":[],"meta":{"source":"upstream-api-service"}}` {
		t.Fatalf("unexpected forwarded body %q", body)
	}
}

func TestGatewayRouterIntegrationReturnsStructuredBridgeErrorWhenUpstreamFails(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	integration.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081"})
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}
	integration.GetRouter().SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp upstream unavailable")
		}),
	})

	req := httptest.NewRequest(http.MethodGet, "/events?limit=5", nil)
	w := httptest.NewRecorder()
	integration.HandleRequest(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", got)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode bridge error payload: %v", err)
	}
	if got := payload["error"]; got != "query_upstream_unavailable" {
		t.Fatalf("expected query_upstream_unavailable, got %v", got)
	}
	meta, ok := payload["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object, got %#v", payload["meta"])
	}
	if got := meta["bridgePosture"]; got != "query-bridge-unavailable" {
		t.Fatalf("expected query-bridge-unavailable, got %v", got)
	}
}

func TestGatewayRouterIntegrationRefreshesUpstreamQueryHealth(t *testing.T) {
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	queryHandler := NewEventQueryHandler(nil, logger, metrics)
	subscriptionHandler := NewEventSubscriptionHandler(nil, logger, metrics)
	healthHandler := NewHealthCheckHandler(nil, logger, metrics)

	integration := NewGatewayRouterIntegration(logger, metrics, queryHandler, subscriptionHandler, healthHandler)
	integration.SetUpstreamQueryEndpoints([]string{"http://api-service-1:8081", "http://api-service-2:8081"})
	integration.SetUpstreamQueryHealthHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}
			if strings.Contains(r.URL.Host, "api-service-2") {
				resp.StatusCode = http.StatusServiceUnavailable
			}
			return resp, nil
		}),
	})
	if err := integration.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize integration: %v", err)
	}

	integration.RefreshUpstreamQueryBridgeHealth()
	configured, attached, available := integration.GetUpstreamQueryBridgeStatus()
	if configured != 2 || attached != 2 || available != 1 {
		t.Fatalf("expected bridge status 2/2/1, got %d/%d/%d", configured, attached, available)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
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
	if testing.Short() {
		t.Skip("skipping concurrent integration test in short mode")
	}

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
