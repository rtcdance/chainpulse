package gateway

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewRESTHandler tests REST handler creation
func TestNewRESTHandler(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)

	assert.NotNil(t, handler)
	assert.Equal(t, 8080, handler.port)
	assert.False(t, handler.running)
}

// TestRESTHandlerStart tests starting REST handler
func TestRESTHandlerStart(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	err := handler.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, handler.running)
}

// TestRESTHandlerStartAlreadyRunning tests starting already running handler
func TestRESTHandlerStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_ = handler.Start(ctx)
	err := handler.Start(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

// TestRESTHandlerStop tests stopping REST handler
func TestRESTHandlerStop(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_ = handler.Start(ctx)
	err := handler.Stop()

	assert.NoError(t, err)
	assert.False(t, handler.running)
}

// TestRESTHandlerStopNotRunning tests stopping non-running handler
func TestRESTHandlerStopNotRunning(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)

	err := handler.Stop()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestRESTHandlerHandleRequest tests handling REST request
func TestRESTHandlerHandleRequest(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_ = handler.Start(ctx)
	resp, err := handler.HandleRequest(ctx, nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "REST", resp.(map[string]any)["protocol"])
}

// TestRESTHandlerHandleRequestNotRunning tests handling request when not running
func TestRESTHandlerHandleRequestNotRunning(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_, err := handler.HandleRequest(ctx, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

// TestRESTHandlerHealth tests REST handler health
func TestRESTHandlerHealth(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_ = handler.Start(ctx)
	health := handler.Health()

	assert.Equal(t, "healthy", health.Status)
	assert.False(t, health.Timestamp.IsZero())
}

// TestRESTHandlerHealthNotRunning tests health when not running
func TestRESTHandlerHealthNotRunning(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)

	health := handler.Health()

	assert.Equal(t, "unhealthy", health.Status)
}

// TestNewgRPCHandler tests gRPC handler creation
func TestNewgRPCHandler(t *testing.T) {
	t.Parallel()
	handler := NewgRPCHandler(9090)

	assert.NotNil(t, handler)
	assert.Equal(t, 9090, handler.port)
	assert.False(t, handler.running)
}

// TestGRPCHandlerStart tests starting gRPC handler
func TestGRPCHandlerStart(t *testing.T) {
	t.Parallel()
	handler := NewgRPCHandler(9090)
	ctx := context.Background()

	err := handler.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, handler.running)
}

// TestGRPCHandlerStop tests stopping gRPC handler
func TestGRPCHandlerStop(t *testing.T) {
	t.Parallel()
	handler := NewgRPCHandler(9090)
	ctx := context.Background()

	_ = handler.Start(ctx)
	err := handler.Stop()

	assert.NoError(t, err)
	assert.False(t, handler.running)
}

// TestGRPCHandlerHandleRequest tests handling gRPC request
func TestGRPCHandlerHandleRequest(t *testing.T) {
	t.Parallel()
	handler := NewgRPCHandler(9090)
	ctx := context.Background()

	_ = handler.Start(ctx)
	resp, err := handler.HandleRequest(ctx, nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "gRPC", resp.(map[string]any)["protocol"])
}

// TestGRPCHandlerHealth tests gRPC handler health
func TestGRPCHandlerHealth(t *testing.T) {
	t.Parallel()
	handler := NewgRPCHandler(9090)
	ctx := context.Background()

	_ = handler.Start(ctx)
	health := handler.Health()

	assert.Equal(t, "healthy", health.Status)
}

// TestNewWebSocketHandler tests WebSocket handler creation
func TestNewWebSocketHandler(t *testing.T) {
	t.Parallel()
	handler := NewWebSocketHandler(8081)

	assert.NotNil(t, handler)
	assert.Equal(t, 8081, handler.port)
	assert.False(t, handler.running)
}

// TestWebSocketHandlerStart tests starting WebSocket handler
func TestWebSocketHandlerStart(t *testing.T) {
	t.Parallel()
	handler := NewWebSocketHandler(8081)
	ctx := context.Background()

	err := handler.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, handler.running)
}

// TestWebSocketHandlerStop tests stopping WebSocket handler
func TestWebSocketHandlerStop(t *testing.T) {
	t.Parallel()
	handler := NewWebSocketHandler(8081)
	ctx := context.Background()

	_ = handler.Start(ctx)
	err := handler.Stop()

	assert.NoError(t, err)
	assert.False(t, handler.running)
}

// TestWebSocketHandlerHandleRequest tests handling WebSocket request
func TestWebSocketHandlerHandleRequest(t *testing.T) {
	t.Parallel()
	handler := NewWebSocketHandler(8081)
	ctx := context.Background()

	_ = handler.Start(ctx)
	resp, err := handler.HandleRequest(ctx, nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "WebSocket", resp.(map[string]any)["protocol"])
}

// TestWebSocketHandlerHealth tests WebSocket handler health
func TestWebSocketHandlerHealth(t *testing.T) {
	t.Parallel()
	handler := NewWebSocketHandler(8081)
	ctx := context.Background()

	_ = handler.Start(ctx)
	health := handler.Health()

	assert.Equal(t, "healthy", health.Status)
}

// TestNewGraphQLHandler tests GraphQL handler creation
func TestNewGraphQLHandler(t *testing.T) {
	t.Parallel()
	handler := NewGraphQLHandler(8082)

	assert.NotNil(t, handler)
	assert.Equal(t, 8082, handler.port)
	assert.False(t, handler.running)
}

// TestGraphQLHandlerStart tests starting GraphQL handler
func TestGraphQLHandlerStart(t *testing.T) {
	t.Parallel()
	handler := NewGraphQLHandler(8082)
	ctx := context.Background()

	err := handler.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, handler.running)
}

// TestGraphQLHandlerStop tests stopping GraphQL handler
func TestGraphQLHandlerStop(t *testing.T) {
	t.Parallel()
	handler := NewGraphQLHandler(8082)
	ctx := context.Background()

	_ = handler.Start(ctx)
	err := handler.Stop()

	assert.NoError(t, err)
	assert.False(t, handler.running)
}

// TestGraphQLHandlerHandleRequest tests handling GraphQL request
func TestGraphQLHandlerHandleRequest(t *testing.T) {
	t.Parallel()
	handler := NewGraphQLHandler(8082)
	ctx := context.Background()

	_ = handler.Start(ctx)
	resp, err := handler.HandleRequest(ctx, nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "GraphQL", resp.(map[string]any)["protocol"])
}

// TestGraphQLHandlerHealth tests GraphQL handler health
func TestGraphQLHandlerHealth(t *testing.T) {
	t.Parallel()
	handler := NewGraphQLHandler(8082)
	ctx := context.Background()

	_ = handler.Start(ctx)
	health := handler.Health()

	assert.Equal(t, "healthy", health.Status)
}

// TestNewMultiProtocolAPI tests multi-protocol API creation
func TestNewMultiProtocolAPI(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()

	assert.NotNil(t, api)
	assert.NotNil(t, api.handlers)
	assert.Equal(t, 0, len(api.handlers))
}

// TestRegisterHandler tests registering a handler
func TestRegisterHandler(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	handler := NewRESTHandler(8080)

	err := api.RegisterHandler("rest", handler)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(api.handlers))
}

// TestRegisterHandlerDuplicate tests registering duplicate handler
func TestRegisterHandlerDuplicate(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	handler1 := NewRESTHandler(8080)
	handler2 := NewRESTHandler(8081)

	_ = api.RegisterHandler("rest", handler1)
	err := api.RegisterHandler("rest", handler2)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

// TestStartAll tests starting all handlers
func TestStartAll(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	ctx := context.Background()

	_ = api.RegisterHandler("rest", NewRESTHandler(8080))
	_ = api.RegisterHandler("grpc", NewgRPCHandler(9090))

	err := api.StartAll(ctx)

	assert.NoError(t, err)
}

// TestStopAll tests stopping all handlers
func TestStopAll(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	ctx := context.Background()

	_ = api.RegisterHandler("rest", NewRESTHandler(8080))
	_ = api.RegisterHandler("grpc", NewgRPCHandler(9090))

	_ = api.StartAll(ctx)
	err := api.StopAll()

	assert.NoError(t, err)
}

// TestGetHandler tests getting a handler
func TestGetHandler(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	handler := NewRESTHandler(8080)

	_ = api.RegisterHandler("rest", handler)
	retrieved, err := api.GetHandler("rest")

	assert.NoError(t, err)
	assert.Equal(t, handler, retrieved)
}

// TestGetHandlerNotFound tests getting non-existent handler
func TestGetHandlerNotFound(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()

	_, err := api.GetHandler("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestHealthAll tests getting health of all handlers
func TestHealthAll(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	ctx := context.Background()

	_ = api.RegisterHandler("rest", NewRESTHandler(8080))
	_ = api.RegisterHandler("grpc", NewgRPCHandler(9090))

	_ = api.StartAll(ctx)
	health := api.HealthAll()

	assert.Equal(t, 2, len(health))
	assert.Equal(t, "healthy", health["rest"].Status)
	assert.Equal(t, "healthy", health["grpc"].Status)
}

// TestMultipleProtocols tests multiple protocol handlers
func TestMultipleProtocols(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	ctx := context.Background()

	_ = api.RegisterHandler("rest", NewRESTHandler(8080))
	_ = api.RegisterHandler("grpc", NewgRPCHandler(9090))
	_ = api.RegisterHandler("websocket", NewWebSocketHandler(8081))
	_ = api.RegisterHandler("graphql", NewGraphQLHandler(8082))

	err := api.StartAll(ctx)

	assert.NoError(t, err)
	assert.Equal(t, 4, len(api.handlers))
}

// TestConcurrentHandlerOperations tests concurrent handler operations
func TestConcurrentHandlerOperations(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	ctx := context.Background()

	_ = api.RegisterHandler("rest", NewRESTHandler(8080))
	_ = api.StartAll(ctx)

	var wg sync.WaitGroup
	var requestCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler, err := api.GetHandler("rest")
			if err == nil {
				_, err := handler.HandleRequest(ctx, nil)
				if err == nil {
					atomic.AddInt32(&requestCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	assert.Greater(t, atomic.LoadInt32(&requestCount), int32(0))
}

// TestHandlerStartStopCycle tests handler start/stop cycle
func TestHandlerStartStopCycle(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_ = handler.Start(ctx)
		assert.True(t, handler.running)

		_ = handler.Stop()
		assert.False(t, handler.running)
	}
}

// TestProtocolHandlerInterface tests protocol handler interface
func TestProtocolHandlerInterface(t *testing.T) {
	t.Parallel()
	var handler ProtocolHandler = NewRESTHandler(8080)

	assert.NotNil(t, handler)
}

// TestHandlerPortConfiguration tests handler port configuration
func TestHandlerPortConfiguration(t *testing.T) {
	t.Parallel()
	restHandler := NewRESTHandler(8080)
	grpcHandler := NewgRPCHandler(9090)
	wsHandler := NewWebSocketHandler(8081)
	graphqlHandler := NewGraphQLHandler(8082)

	assert.Equal(t, 8080, restHandler.port)
	assert.Equal(t, 9090, grpcHandler.port)
	assert.Equal(t, 8081, wsHandler.port)
	assert.Equal(t, 8082, graphqlHandler.port)
}

// TestMultiProtocolAPIHealthEmpty tests health of empty API
func TestMultiProtocolAPIHealthEmpty(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()

	health := api.HealthAll()

	assert.Equal(t, 0, len(health))
}

// TestHandlerResponseFormat tests handler response format
func TestHandlerResponseFormat(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_ = handler.Start(ctx)
	resp, _ := handler.HandleRequest(ctx, nil)

	respMap := resp.(map[string]any)
	assert.Equal(t, "REST", respMap["protocol"])
	assert.Equal(t, "ok", respMap["status"])
}

// TestConcurrentHandlerRegistration tests concurrent handler registration
func TestConcurrentHandlerRegistration(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()

	var wg sync.WaitGroup
	var registerCount int32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			protocol := fmt.Sprintf("protocol-%d", id)
			handler := NewRESTHandler(8000 + id)
			if err := api.RegisterHandler(protocol, handler); err == nil {
				atomic.AddInt32(&registerCount, 1)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(10), atomic.LoadInt32(&registerCount))
}

// TestHandlerHealthTimestamp tests handler health timestamp
func TestHandlerHealthTimestamp(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	before := time.Now()
	_ = handler.Start(ctx)
	health := handler.Health()
	after := time.Now()

	assert.True(t, health.Timestamp.After(before) || health.Timestamp.Equal(before))
	assert.True(t, health.Timestamp.Before(after) || health.Timestamp.Equal(after))
}

// TestMultiProtocolAPIStartAllFailure tests StartAll with handler failure
func TestMultiProtocolAPIStartAllFailure(t *testing.T) {
	t.Parallel()
	api := NewMultiProtocolAPI()
	ctx := context.Background()

	handler := NewRESTHandler(8080)
	_ = api.RegisterHandler("rest", handler)

	// Start first time
	_ = api.StartAll(ctx)

	// Try to start again (should fail)
	err := api.StartAll(ctx)

	assert.Error(t, err)
}

// TestHandlerConcurrentRequests tests concurrent requests to handler
func TestHandlerConcurrentRequests(t *testing.T) {
	t.Parallel()
	handler := NewRESTHandler(8080)
	ctx := context.Background()

	_ = handler.Start(ctx)

	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := handler.HandleRequest(ctx, nil)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(100), atomic.LoadInt32(&successCount))
}
