package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/observability"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRequestRouter tests request router initialization
func TestNewRequestRouter(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()

	router := NewRequestRouter(logger, metrics)

	require.NotNil(t, router)
	assert.Equal(t, 0, len(router.routes))
	assert.Equal(t, 0, len(router.loadBalancers))
	assert.False(t, router.initialized)
}

// TestInitialize tests router initialization
func TestInitialize(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	err := router.Initialize(context.Background())

	require.NoError(t, err)
	assert.True(t, router.initialized)
}

// TestInitializeWithoutLogger tests initialization without logger
func TestInitializeWithoutLogger(t *testing.T) {
	t.Parallel()
	metrics := NewMockMetricsCollector()
	router := &RequestRouter{
		routes:          make(map[string]*Route),
		loadBalancers:   make(map[string]*LoadBalancer),
		logger:          nil,
		metrics:         metrics,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	err := router.Initialize(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "logger is required")
}

// TestInitializeWithoutMetrics tests initialization without metrics
func TestInitializeWithoutMetrics(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	router := &RequestRouter{
		routes:          make(map[string]*Route),
		loadBalancers:   make(map[string]*LoadBalancer),
		logger:          logger,
		metrics:         nil,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	err := router.Initialize(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metrics collector is required")
}

// TestRegisterRoute tests route registration
func TestRegisterRoute(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{
		ID:      "route1",
		Pattern: "/api/users",
		Method:  "GET",
	}

	err := router.RegisterRoute(route)

	require.NoError(t, err)
	assert.Equal(t, 1, len(router.routes))
}

// TestRegisterRouteNil tests registering nil route
func TestRegisterRouteNil(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	err := router.RegisterRoute(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be nil")
}

// TestRegisterRouteNoID tests registering route without ID
func TestRegisterRouteNoID(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{
		Pattern: "/api/users",
		Method:  "GET",
	}

	err := router.RegisterRoute(route)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route ID is required")
}

// TestRegisterRouteNoPattern tests registering route without pattern
func TestRegisterRouteNoPattern(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{
		ID:     "route1",
		Method: "GET",
	}

	err := router.RegisterRoute(route)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route pattern is required")
}

// TestRegisterRouteDuplicate tests registering duplicate route
func TestRegisterRouteDuplicate(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{
		ID:      "route1",
		Pattern: "/api/users",
		Method:  "GET",
	}

	err := router.RegisterRoute(route)
	require.NoError(t, err)
	err = router.RegisterRoute(route)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestUnregisterRoute tests route unregistration
func TestUnregisterRoute(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{
		ID:      "route1",
		Pattern: "/api/users",
		Method:  "GET",
	}

	err := router.RegisterRoute(route)
	require.NoError(t, err)
	err = router.UnregisterRoute("route1")

	require.NoError(t, err)
	assert.Equal(t, 0, len(router.routes))
}

// TestUnregisterRouteNotFound tests unregistering nonexistent route
func TestUnregisterRouteNotFound(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	err := router.UnregisterRoute("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetRoute tests getting a route
func TestGetRoute(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{
		ID:      "route1",
		Pattern: "/api/users",
		Method:  "GET",
	}

	err := router.RegisterRoute(route)
	require.NoError(t, err)
	retrieved, err := router.GetRoute("route1")

	require.NoError(t, err)
	assert.Equal(t, "route1", retrieved.ID)
}

// TestGetRouteNotFound tests getting nonexistent route
func TestGetRouteNotFound(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	_, err := router.GetRoute("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestGetRoutes tests getting all routes
func TestGetRoutes(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route1 := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	route2 := &Route{ID: "route2", Pattern: "/api/posts", Method: "GET"}

	err := router.RegisterRoute(route1)
	require.NoError(t, err)
	err = router.RegisterRoute(route2)
	require.NoError(t, err)

	routes := router.GetRoutes()

	assert.Equal(t, 2, len(routes))
}

// TestRequestRouterGetMetrics tests getting router metrics
func TestRequestRouterGetMetrics(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	err := router.RegisterRoute(route)
	require.NoError(t, err)

	routerMetrics := router.GetMetrics()

	assert.Equal(t, 1, routerMetrics["route_count"])
	assert.Equal(t, 1, routerMetrics["load_balancer_count"])
}

// TestNewCircuitBreaker tests circuit breaker initialization
func TestNewCircuitBreaker(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(5, 30*time.Second)

	require.NotNil(t, cb)
	assert.Equal(t, "closed", cb.state)
	assert.Equal(t, 0, cb.failureCount)
	assert.Equal(t, 5, cb.threshold)
}

// TestCircuitBreakerRecordSuccess tests recording success
func TestCircuitBreakerRecordSuccess(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(5, 30*time.Second)

	cb.RecordSuccess()

	assert.Equal(t, "closed", cb.GetState())
}

// TestCircuitBreakerRecordFailure tests recording failure
func TestCircuitBreakerRecordFailure(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(3, 30*time.Second)

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	assert.Equal(t, "open", cb.GetState())
}

// TestCircuitBreakerIsOpen tests checking if circuit breaker is open
func TestCircuitBreakerIsOpen(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(2, 30*time.Second)

	assert.False(t, cb.IsOpen())

	cb.RecordFailure()
	cb.RecordFailure()

	assert.True(t, cb.IsOpen())
}

// TestCircuitBreakerTimeout tests circuit breaker timeout
func TestCircuitBreakerTimeout(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	cb.RecordFailure()
	assert.True(t, cb.IsOpen())

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open
	assert.False(t, cb.IsOpen())
	assert.Equal(t, "half-open", cb.GetState())
}

// TestCircuitBreakerHalfOpenSuccess tests half-open to closed transition
func TestCircuitBreakerHalfOpenSuccess(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(1, 100*time.Millisecond)

	cb.RecordFailure()
	time.Sleep(150 * time.Millisecond)
	cb.IsOpen() // Transition to half-open

	// Record successes
	cb.RecordSuccess()
	cb.RecordSuccess()
	cb.RecordSuccess()

	assert.Equal(t, "closed", cb.GetState())
}

// TestCircuitBreakerGetState tests getting circuit breaker state
func TestCircuitBreakerGetState(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(5, 30*time.Second)

	assert.Equal(t, "closed", cb.GetState())

	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()
	cb.RecordFailure()

	assert.Equal(t, "open", cb.GetState())
}

// TestConcurrentRouteRegistration tests concurrent route registration
func TestConcurrentRouteRegistration(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	var wg sync.WaitGroup
	numRoutes := 10

	for i := 0; i < numRoutes; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			route := &Route{
				ID:      fmt.Sprintf("route%d", id),
				Pattern: fmt.Sprintf("/api/endpoint%d", id),
				Method:  "GET",
			}
			_ = router.RegisterRoute(route)
		}(i)
	}

	wg.Wait()

	assert.Equal(t, numRoutes, len(router.GetRoutes()))
}

// TestConcurrentRouteUnregistration tests concurrent route unregistration
func TestConcurrentRouteUnregistration(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	// Register routes
	for i := 0; i < 10; i++ {
		route := &Route{
			ID:      fmt.Sprintf("route%d", i),
			Pattern: fmt.Sprintf("/api/endpoint%d", i),
			Method:  "GET",
		}
		_ = router.RegisterRoute(route)
	}

	// Unregister concurrently
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_ = router.UnregisterRoute(fmt.Sprintf("route%d", id))
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, len(router.GetRoutes()))
}

// TestForwardRequestNilRoute tests forwarding with nil route
func TestForwardRequestNilRoute(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	req := &ForwardedRequest{
		Method: "GET",
		Path:   "/test",
	}

	_, err := router.ForwardRequest(context.Background(), nil, req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "route cannot be nil")
}

// TestForwardRequestNilRequest tests forwarding with nil request
func TestForwardRequestNilRequest(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}

	_, err := router.ForwardRequest(context.Background(), route, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")
}

func TestForwardRequestInjectsTraceContext(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	require.NoError(t, router.RegisterRoute(route))
	require.NoError(t, router.AttachHandler("route1", NewHandler("handler1", "handler1", "http://upstream.example")))
	router.SetHTTPClient(&http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("traceparent") == "" {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       http.NoBody,
					Header:     make(http.Header),
				}, nil
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       http.NoBody,
				Header:     make(http.Header),
			}, nil
		}),
	})

	tracer := observability.NewDefaultTracer(logger, metrics)
	ctx, span := tracer.StartSpan(context.Background(), "router.forward", observability.SpanKindClient)
	defer tracer.EndSpan(&span)

	response, err := router.ForwardRequest(ctx, route, &ForwardedRequest{
		Method: "GET",
		Path:   "/test",
	})

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusOK, response.Status)
}

// TestMultipleRoutes tests managing multiple routes
func TestMultipleRoutes(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	routes := []*Route{
		{ID: "users", Pattern: "/api/users", Method: "GET"},
		{ID: "posts", Pattern: "/api/posts", Method: "GET"},
		{ID: "comments", Pattern: "/api/comments", Method: "GET"},
	}

	for _, route := range routes {
		err := router.RegisterRoute(route)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, len(router.GetRoutes()))

	// Unregister one
	_ = router.UnregisterRoute("posts")
	assert.Equal(t, 2, len(router.GetRoutes()))
}

// TestRouterMetricsRecording tests metrics recording
func TestRouterMetricsRecording(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	_ = router.RegisterRoute(route)

	assert.Greater(t, metrics.GetCounterValue("router_route_registered"), int64(0))

	_ = router.UnregisterRoute("route1")

	assert.Greater(t, metrics.GetCounterValue("router_route_unregistered"), int64(0))
}

// TestCircuitBreakerConcurrency tests circuit breaker thread safety
func TestCircuitBreakerConcurrency(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(100, 30*time.Second)

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if id%2 == 0 {
				cb.RecordSuccess()
			} else {
				cb.RecordFailure()
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and state should be valid
	state := cb.GetState()
	assert.True(t, state == "closed" || state == "open" || state == "half-open")
}

// TestRouterInitializeIdempotent tests that initialize is idempotent
func TestRouterInitializeIdempotent(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	err1 := router.Initialize(context.Background())
	err2 := router.Initialize(context.Background())

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.True(t, router.initialized)
}

// TestRouteRegistrationWithLoadBalancer tests load balancer creation on route registration
func TestRouteRegistrationWithLoadBalancer(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	err := router.RegisterRoute(route)
	require.NoError(t, err)

	// Verify load balancer was created
	assert.Equal(t, 1, len(router.loadBalancers))
	assert.NotNil(t, router.loadBalancers["route1"])
}

// TestRouteRegistrationWithCircuitBreaker tests circuit breaker creation on route registration
func TestRouteRegistrationWithCircuitBreaker(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	err := router.RegisterRoute(route)
	require.NoError(t, err)

	// Verify circuit breaker was created
	assert.Equal(t, 1, len(router.circuitBreakers))
	assert.NotNil(t, router.circuitBreakers["route1"])
}

// TestRouteUnregistrationCleansUp tests cleanup on route unregistration
func TestRouteUnregistrationCleansUp(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	route := &Route{ID: "route1", Pattern: "/api/users", Method: "GET"}
	err := router.RegisterRoute(route)
	require.NoError(t, err)

	assert.Equal(t, 1, len(router.loadBalancers))
	assert.Equal(t, 1, len(router.circuitBreakers))

	_ = router.UnregisterRoute("route1")

	assert.Equal(t, 0, len(router.loadBalancers))
	assert.Equal(t, 0, len(router.circuitBreakers))
}

// TestCircuitBreakerFailureThreshold tests failure threshold
func TestCircuitBreakerFailureThreshold(t *testing.T) {
	t.Parallel()
	cb := NewCircuitBreaker(3, 30*time.Second)

	assert.False(t, cb.IsOpen())

	cb.RecordFailure()
	assert.False(t, cb.IsOpen())

	cb.RecordFailure()
	assert.False(t, cb.IsOpen())

	cb.RecordFailure()
	assert.True(t, cb.IsOpen())
}

// TestForwardedRequestStructure tests ForwardedRequest structure
func TestForwardedRequestStructure(t *testing.T) {
	t.Parallel()
	req := &ForwardedRequest{
		Method:  "POST",
		Path:    "/api/users",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"name":"test"}`),
		Params:  map[string]string{"id": "123"},
	}

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/api/users", req.Path)
	assert.Equal(t, "application/json", req.Headers["Content-Type"])
	assert.Equal(t, "123", req.Params["id"])
}

// TestAggregatedResponseStructure tests AggregatedResponse structure
func TestAggregatedResponseStructure(t *testing.T) {
	t.Parallel()
	resp := &AggregatedResponse{
		Status:  200,
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    []byte(`{"success":true}`),
	}

	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
}

// TestRouterDefaultTimeout tests default timeout setting
func TestRouterDefaultTimeout(t *testing.T) {
	t.Parallel()
	logger := &MockLogger{}
	metrics := NewMockMetricsCollector()
	router := NewRequestRouter(logger, metrics)

	assert.Equal(t, 30*time.Second, router.defaultTimeout)
}
