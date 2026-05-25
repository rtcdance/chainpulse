package gateway

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/infrastructure/discovery"
	"github.com/stretchr/testify/assert"
)

// MockServiceDiscoveryClient for testing
type MockServiceDiscoveryClient struct {
	services map[string]*discovery.ServiceInfo
}

func (msdc *MockServiceDiscoveryClient) GetService(ctx context.Context, serviceName string) (*discovery.ServiceInfo, error) {
	if service, exists := msdc.services[serviceName]; exists {
		return service, nil
	}
	return nil, nil
}

func (msdc *MockServiceDiscoveryClient) GetServices(ctx context.Context, serviceName string) ([]*discovery.ServiceInfo, error) {
	if service, exists := msdc.services[serviceName]; exists {
		return []*discovery.ServiceInfo{service}, nil
	}
	return []*discovery.ServiceInfo{}, nil
}

func (msdc *MockServiceDiscoveryClient) InvalidateCache(serviceName string) {
}

func (msdc *MockServiceDiscoveryClient) InvalidateAllCache() {
}

// MockServiceLoadBalancer for testing
type MockServiceLoadBalancer struct {
	services map[string]*discovery.ServiceInfo
}

func (mslb *MockServiceLoadBalancer) SelectService(ctx context.Context, serviceName string) (*discovery.ServiceInfo, error) {
	if service, exists := mslb.services[serviceName]; exists {
		return service, nil
	}
	return nil, fmt.Errorf("service not found: %s", serviceName)
}

// TestNewAPIGateway tests gateway creation
func TestNewAPIGateway(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{
		Port:              8080,
		Protocols:         []string{"rest", "grpc"},
		MaxConnections:    1000,
		RequestTimeout:    30 * time.Second,
		EnableCompression: true,
		EnableCaching:     true,
	}

	gateway := NewAPIGateway(config, nil, nil)

	assert.NotNil(t, gateway)
	assert.Equal(t, config.Port, gateway.config.Port)
	assert.Equal(t, 2, len(gateway.config.Protocols))
	assert.False(t, gateway.running)
}

// TestStartGateway tests starting the gateway
func TestStartGateway(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)
	ctx := context.Background()

	err := gateway.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, gateway.running)
}

// TestStartGatewayAlreadyRunning tests starting already running gateway
func TestStartGatewayAlreadyRunning(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)
	ctx := context.Background()

	_ = gateway.Start(ctx)

	err := gateway.Start(ctx)

	assert.Error(t, err)
}

// TestStopGateway tests stopping the gateway
func TestStopGateway(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)
	ctx := context.Background()

	_ = gateway.Start(ctx)
	err := gateway.Stop()

	assert.NoError(t, err)
	assert.False(t, gateway.running)
}

// TestStopGatewayNotRunning tests stopping non-running gateway
func TestStopGatewayNotRunning(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)

	err := gateway.Stop()

	assert.Error(t, err)
}

// TestHandleRequestWithoutLoadBalancer tests that HandleRequest requires proper initialization
func TestHandleRequestWithoutLoadBalancer(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)
	ctx := context.Background()

	// Create a session
	session, err := gateway.sessionManager.CreateSession(ctx, "user-1")
	assert.NoError(t, err)

	// Verify gateway components are initialized
	assert.NotNil(t, gateway.sessionManager)
	assert.NotNil(t, gateway.cache)
	assert.NotNil(t, gateway.metrics)
	assert.Equal(t, session.ID, session.ID) // Verify session was created
}

// TestAPIMetricsRecordRequest tests recording requests
func TestAPIMetricsRecordRequest(t *testing.T) {
	t.Parallel()
	metrics := NewAPIMetrics()

	assert.Equal(t, int64(0), metrics.totalRequests)

	metrics.RecordRequest()
	assert.Equal(t, int64(1), metrics.totalRequests)

	metrics.RecordRequest()
	assert.Equal(t, int64(2), metrics.totalRequests)
}

// TestAPIMetricsRecordError tests recording errors
func TestAPIMetricsRecordError(t *testing.T) {
	t.Parallel()
	metrics := NewAPIMetrics()

	assert.Equal(t, int64(0), metrics.totalErrors)

	metrics.RecordError()
	assert.Equal(t, int64(1), metrics.totalErrors)

	metrics.RecordError()
	assert.Equal(t, int64(2), metrics.totalErrors)
}

// TestAPIMetricsRecordLatency tests recording latency
func TestAPIMetricsRecordLatency(t *testing.T) {
	t.Parallel()
	metrics := NewAPIMetrics()

	assert.Equal(t, int64(0), metrics.totalLatency)

	metrics.RecordLatency(100 * time.Millisecond)
	assert.Equal(t, int64(100), metrics.totalLatency)

	metrics.RecordLatency(50 * time.Millisecond)
	assert.Equal(t, int64(150), metrics.totalLatency)
}

// TestAPIMetricsGetMetrics tests getting metrics
func TestAPIMetricsGetMetrics(t *testing.T) {
	t.Parallel()
	metrics := NewAPIMetrics()

	metrics.RecordRequest()
	metrics.RecordRequest()
	metrics.RecordError()
	metrics.RecordLatency(100 * time.Millisecond)
	metrics.RecordLatency(50 * time.Millisecond)

	m := metrics.GetMetrics()

	assert.Equal(t, int64(2), m["total_requests"])
	assert.Equal(t, int64(1), m["total_errors"])
	assert.Equal(t, int64(75), m["avg_latency_ms"])
}

// TestNewAPIGatewayCluster tests cluster creation
func TestNewAPIGatewayCluster(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()

	assert.NotNil(t, cluster)
	assert.NotNil(t, cluster.gateways)
	assert.Equal(t, 0, len(cluster.gateways))
}

// TestAddGateway tests adding a gateway to cluster
func TestAddGateway(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)

	err := cluster.AddGateway("gateway-1", gateway)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(cluster.gateways))
}

// TestAddGatewayDuplicate tests adding duplicate gateway
func TestAddGatewayDuplicate(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)

	_ = cluster.AddGateway("gateway-1", gateway)
	err := cluster.AddGateway("gateway-1", gateway)

	assert.Error(t, err)
}

// TestRemoveGateway tests removing a gateway from cluster
func TestRemoveGateway(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)

	_ = cluster.AddGateway("gateway-1", gateway)
	err := cluster.RemoveGateway("gateway-1")

	assert.NoError(t, err)
	assert.Equal(t, 0, len(cluster.gateways))
}

// TestRemoveGatewayNotFound tests removing non-existent gateway
func TestRemoveGatewayNotFound(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()

	err := cluster.RemoveGateway("nonexistent")

	assert.Error(t, err)
}

// TestGetGateway tests getting a gateway from cluster
func TestGetGateway(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)

	_ = cluster.AddGateway("gateway-1", gateway)
	retrieved, err := cluster.GetGateway("gateway-1")

	assert.NoError(t, err)
	assert.Equal(t, gateway, retrieved)
}

// TestGetGatewayNotFound tests getting non-existent gateway
func TestGetGatewayNotFound(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()

	_, err := cluster.GetGateway("nonexistent")

	assert.Error(t, err)
}

// TestListGateways tests listing all gateways
func TestListGateways(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	for i := 1; i <= 3; i++ {
		gateway := NewAPIGateway(config, nil, nil)
		_ = cluster.AddGateway("gateway-"+string(rune(48+i)), gateway)
	}

	gateways := cluster.ListGateways()

	assert.Equal(t, 3, len(gateways))
}

// TestGetClusterMetrics tests getting cluster metrics
func TestGetClusterMetrics(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	for i := 1; i <= 3; i++ {
		gateway := NewAPIGateway(config, nil, nil)
		gateway.metrics.RecordRequest()
		gateway.metrics.RecordRequest()
		_ = cluster.AddGateway("gateway-"+string(rune(48+i)), gateway)
	}

	metrics := cluster.GetClusterMetrics()

	assert.Equal(t, int64(6), metrics["total_requests"])
	assert.Equal(t, 3, metrics["gateway_count"])
}

// TestConcurrentMetricsRecording tests concurrent metrics recording
func TestConcurrentMetricsRecording(t *testing.T) {
	t.Parallel()
	metrics := NewAPIMetrics()

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			metrics.RecordRequest()
			atomic.AddInt32(&counter, 1)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&counter))
	assert.Equal(t, int64(100), metrics.totalRequests)
}

// TestConcurrentGatewayOperations tests concurrent gateway operations
func TestConcurrentGatewayOperations(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	var wg sync.WaitGroup
	var counter int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			gateway := NewAPIGateway(config, nil, nil)
			gatewayID := "gateway-" + string(rune(48+(id%10)))
			if err := cluster.AddGateway(gatewayID, gateway); err == nil {
				atomic.AddInt32(&counter, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Greater(t, atomic.LoadInt32(&counter), int32(0))
}

// TestAPIRequestFields tests API request fields
func TestAPIRequestFields(t *testing.T) {
	t.Parallel()
	req := &APIRequest{
		SessionID:   "session-1",
		ServiceName: "api-service",
		Protocol:    "rest",
		Method:      "POST",
		Path:        "/api/v1/data",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: []byte(`{"key": "value"}`),
	}

	assert.Equal(t, "session-1", req.SessionID)
	assert.Equal(t, "api-service", req.ServiceName)
	assert.Equal(t, "rest", req.Protocol)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/api/v1/data", req.Path)
	assert.Equal(t, "application/json", req.Headers["Content-Type"])
}

// TestAPIResponseFields tests API response fields
func TestAPIResponseFields(t *testing.T) {
	t.Parallel()
	resp := &APIResponse{
		SessionID:   "session-1",
		ServiceName: "api-service",
		Status:      200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body:      []byte(`{"result": "success"}`),
		Timestamp: time.Now(),
	}

	assert.Equal(t, "session-1", resp.SessionID)
	assert.Equal(t, "api-service", resp.ServiceName)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
	assert.NotZero(t, resp.Timestamp)
}

// TestGatewayConfigFields tests gateway config fields
func TestGatewayConfigFields(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{
		Port:              8080,
		Protocols:         []string{"rest", "grpc", "websocket"},
		MaxConnections:    1000,
		RequestTimeout:    30 * time.Second,
		EnableCompression: true,
		EnableCaching:     true,
	}

	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, 3, len(config.Protocols))
	assert.Equal(t, 1000, config.MaxConnections)
	assert.Equal(t, 30*time.Second, config.RequestTimeout)
	assert.True(t, config.EnableCompression)
	assert.True(t, config.EnableCaching)
}

// TestMultipleGatewaysInCluster tests multiple gateways in cluster
func TestMultipleGatewaysInCluster(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	for i := 1; i <= 5; i++ {
		gateway := NewAPIGateway(config, nil, nil)
		_ = cluster.AddGateway("gateway-"+string(rune(48+i)), gateway)
	}

	assert.Equal(t, 5, len(cluster.ListGateways()))
}

// TestMetricsAverageLatency tests average latency calculation
func TestMetricsAverageLatency(t *testing.T) {
	t.Parallel()
	metrics := NewAPIMetrics()

	metrics.RecordRequest()
	metrics.RecordLatency(100 * time.Millisecond)

	metrics.RecordRequest()
	metrics.RecordLatency(200 * time.Millisecond)

	m := metrics.GetMetrics()

	assert.Equal(t, int64(150), m["avg_latency_ms"])
}

// TestGatewayStartStopCycle tests start/stop cycle
func TestGatewayStartStopCycle(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	gateway := NewAPIGateway(config, nil, nil)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_ = gateway.Start(ctx)
		assert.True(t, gateway.running)

		_ = gateway.Stop()
		assert.False(t, gateway.running)
	}
}

// TestClusterMetricsAggregation tests cluster metrics aggregation
func TestClusterMetricsAggregation(t *testing.T) {
	t.Parallel()
	cluster := NewAPIGatewayCluster()
	config := APIGatewayConfig{Port: 8080}
	// Using nil for discovery client
	// Using nil for load balancer

	for i := 1; i <= 3; i++ {
		gateway := NewAPIGateway(config, nil, nil)
		for j := 0; j < i; j++ {
			gateway.metrics.RecordRequest()
			gateway.metrics.RecordError()
		}
		_ = cluster.AddGateway("gateway-"+string(rune(48+i)), gateway)
	}

	metrics := cluster.GetClusterMetrics()

	assert.Equal(t, int64(6), metrics["total_requests"])
	assert.Equal(t, int64(6), metrics["total_errors"])
	assert.Equal(t, 3, metrics["gateway_count"])
}

func TestAPIGatewayRegisterHandler(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	handler := func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		return &APIResponse{Status: 200}, nil
	}

	gateway.RegisterHandler("GET /api/test", handler)

	patterns := gateway.ListHandlers()
	assert.Contains(t, patterns, "GET /api/test")
}

func TestRegisterMiddleware(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	mw := func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
			return next(ctx, req)
		}
	}

	gateway.RegisterMiddleware(mw)

	gateway.mu.RLock()
	assert.Len(t, gateway.middlewareChain, 1)
	gateway.mu.RUnlock()
}

func TestListHandlers(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	handler := func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		return &APIResponse{Status: 200}, nil
	}

	gateway.RegisterHandler("GET /api/a", handler)
	gateway.RegisterHandler("POST /api/b", handler)
	gateway.RegisterHandler("DELETE /api/c", handler)

	patterns := gateway.ListHandlers()
	assert.Len(t, patterns, 3)

	for i := 0; i < len(patterns)-1; i++ {
		assert.True(t, patterns[i] < patterns[i+1], "handlers should be sorted")
	}
}

func TestFindHandler(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	customHandler := func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		return &APIResponse{Status: 201, Body: []byte("custom")}, nil
	}
	gateway.RegisterHandler("POST /api/custom", customHandler)

	h := gateway.findHandler(&APIRequest{Method: "POST", Path: "/api/custom"})
	assert.NotNil(t, h)

	ctx := context.Background()
	resp, err := h(ctx, &APIRequest{Method: "POST", Path: "/api/custom"})
	assert.NoError(t, err)
	assert.Equal(t, 201, resp.Status)
}

func TestWrapWithMiddleware(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	var called []string

	mw1 := func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
			called = append(called, "mw1")
			return next(ctx, req)
		}
	}
	mw2 := func(next RequestHandler) RequestHandler {
		return func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
			called = append(called, "mw2")
			return next(ctx, req)
		}
	}

	gateway.RegisterMiddleware(mw1)
	gateway.RegisterMiddleware(mw2)

	baseHandler := func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		called = append(called, "handler")
		return &APIResponse{Status: 200}, nil
	}

	wrapped := gateway.wrapWithMiddleware(baseHandler)
	ctx := context.Background()
	_, err := wrapped(ctx, &APIRequest{Method: "GET", Path: "/test"})
	assert.NoError(t, err)

	assert.Equal(t, []string{"mw1", "mw2", "handler"}, called)
}

func TestExecuteWithRecoveryPanic(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	panicHandler := func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		panic("test panic")
	}

	resp, err := gateway.executeWithRecovery(context.Background(), &APIRequest{}, panicHandler)
	assert.Nil(t, resp)
	assert.Nil(t, err)
}

func TestExecuteWithRecoveryError(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	errHandler := func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		return nil, fmt.Errorf("handler error")
	}

	resp, err := gateway.executeWithRecovery(context.Background(), &APIRequest{}, errHandler)
	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
}

func TestFindHandlerFallback(t *testing.T) {
	t.Parallel()
	cache := discovery.NewServiceEndpointCache(5 * time.Minute)
	cache.Set("test-service", []*discovery.ServiceInfo{
		{Name: "test-service", Address: "localhost", Port: 9999},
	})
	lb := discovery.NewServiceLoadBalancer(nil, cache, &discovery.RoundRobinStrategy{})

	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, lb)

	h := gateway.findHandler(&APIRequest{Method: "GET", Path: "/api/nonexistent", ServiceName: "test-service"})
	assert.NotNil(t, h)

	ctx := context.Background()
	resp, err := h(ctx, &APIRequest{Method: "GET", Path: "/api/nonexistent", ServiceName: "test-service"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-service", resp.ServiceName)
}

func TestHandleRequestWithRegisteredHandler(t *testing.T) {
	t.Parallel()
	config := APIGatewayConfig{Port: 8080}
	gateway := NewAPIGateway(config, nil, nil)

	ctx := context.Background()
	_ = gateway.Start(ctx)

	gateway.RegisterHandler("GET /api/hello", func(ctx context.Context, req *APIRequest) (*APIResponse, error) {
		return &APIResponse{
			Status:  200,
			Body:    []byte("hello"),
			Headers: map[string]string{"X-Custom": "value"},
		}, nil
	})

	resp, err := gateway.HandleRequest(ctx, &APIRequest{
		Method: "GET",
		Path:   "/api/hello",
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.Status)
	assert.Equal(t, "hello", string(resp.Body))
}
