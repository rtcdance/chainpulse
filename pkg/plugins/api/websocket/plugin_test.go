package websocket

import (
	"testing"
	"time"

	"chainpulse/pkg/plugins/api/core"
	"chainpulse/pkg/plugins/api/shared"
	"github.com/gorilla/websocket"
)

func TestNewWebSocketPlugin(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8090, apiLayer)

	if plugin == nil {
		t.Fatal("expected plugin, got nil")
	}

	if plugin.GetName() != "websocket" {
		t.Errorf("expected name 'websocket', got %s", plugin.GetName())
	}

	if plugin.port != 8090 {
		t.Errorf("expected port 8090, got %d", plugin.port)
	}
}

func TestWebSocketPluginStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8091, apiLayer)

	err := plugin.Start()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running")
	}

	// Clean up
	_ = plugin.Stop()
	time.Sleep(50 * time.Millisecond) // Wait for port to be released
}

func TestWebSocketPluginStartAlreadyRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8092, apiLayer)

	_ = plugin.Start()
	defer func() { 
		_ = plugin.Stop()
		time.Sleep(50 * time.Millisecond) // Wait for port to be released
	}()

	// Try to start again
	err := plugin.Start()
	if err == nil {
		t.Error("expected error when starting already running plugin")
	}
}

func TestWebSocketPluginStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8093, apiLayer)

	_ = plugin.Start()

	err := plugin.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if plugin.IsRunning() {
		t.Error("expected plugin to be stopped")
	}

	time.Sleep(50 * time.Millisecond) // Wait for port to be released
}

func TestWebSocketPluginStopNotRunning(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8094, apiLayer)

	err := plugin.Stop()
	if err == nil {
		t.Error("expected error when stopping non-running plugin")
	}
}

func TestWebSocketPluginGetName(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("test-ws", 8095, apiLayer)

	if plugin.GetName() != "test-ws" {
		t.Errorf("expected name 'test-ws', got %s", plugin.GetName())
	}
}

func TestWebSocketPluginIsRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8096, apiLayer)

	if plugin.IsRunning() {
		t.Error("expected plugin to not be running initially")
	}

	_ = plugin.Start()
	defer func() { 
		_ = plugin.Stop()
		time.Sleep(50 * time.Millisecond) // Wait for port to be released
	}()

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running after start")
	}
}

func TestWebSocketPluginGetClientCount(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8097, apiLayer)

	count := plugin.GetClientCount()
	if count != 0 {
		t.Errorf("expected 0 clients initially, got %d", count)
	}
}

func TestWebSocketPluginGetConnectionMetricsPlaintextIdle(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8097, apiLayer)

	metrics := plugin.GetConnectionMetrics()
	if metrics["transport_posture"] != "ws-plaintext-only" {
		t.Errorf("expected ws-plaintext-only, got %v", metrics["transport_posture"])
	}
	if metrics["connection_posture"] != "ws-stopped" {
		t.Errorf("expected ws-stopped, got %v", metrics["connection_posture"])
	}
}

func TestWebSocketPluginGetConnectionMetricsTLSIdle(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := &WebSocketPlugin{
		name:       "tls-ws",
		port:       8098,
		apiLayer:   apiLayer,
		tlsManager: &shared.TLSManager{},
		clients:    make(map[*websocket.Conn]bool),
		router:     core.NewAPIRouter(),
	}

	metrics := plugin.GetConnectionMetrics()
	if metrics["transport_posture"] != "ws-tls-enabled" {
		t.Errorf("expected ws-tls-enabled, got %v", metrics["transport_posture"])
	}
	if metrics["connection_posture"] != "ws-stopped" {
		t.Errorf("expected ws-stopped, got %v", metrics["connection_posture"])
	}
}

func TestWebSocketPluginGetConnectionMetricsActiveHint(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8099, apiLayer)
	plugin.running = true
	plugin.clients = map[*websocket.Conn]bool{
		{}: true,
	}

	metrics := plugin.GetConnectionMetrics()
	if metrics["connection_posture"] != "ws-active" {
		t.Errorf("expected ws-active, got %v", metrics["connection_posture"])
	}
	if metrics["reliability_hint"] != "websocket transport is active with connected clients on a plaintext runtime" {
		t.Errorf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestWebSocketPluginUseMiddleware(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8098, apiLayer)

	middleware := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware", "applied")
			}
			return resp, err
		})
	}

	if err := plugin.Use(middleware); err != nil {
		t.Fatalf("failed to use middleware: %v", err)
	}

	if len(plugin.middleware) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(plugin.middleware))
	}
}

func TestWebSocketPluginMultipleMiddleware(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8099, apiLayer)

	middleware1 := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			return next.Handle(req)
		})
	}

	middleware2 := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			return next.Handle(req)
		})
	}

	if err := plugin.Use(middleware1, middleware2); err != nil {
		t.Fatalf("failed to use middleware: %v", err)
	}

	if len(plugin.middleware) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(plugin.middleware))
	}
}

func TestWebSocketPluginConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	apiLayer := core.NewAPILayer()
	plugin := NewWebSocketPlugin("websocket", 8100, apiLayer)

	// Start plugin
	err := plugin.Start()
	if err != nil {
		t.Fatalf("failed to start plugin: %v", err)
	}
	defer func() {
		_ = plugin.Stop()
		time.Sleep(50 * time.Millisecond) // Wait for port to be released
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Verify running state from multiple goroutines
	done := make(chan bool, 3)

	go func() {
		if !plugin.IsRunning() {
			t.Error("expected plugin to be running (goroutine 1)")
		}
		done <- true
	}()

	go func() {
		if !plugin.IsRunning() {
			t.Error("expected plugin to be running (goroutine 2)")
		}
		done <- true
	}()

	go func() {
		count := plugin.GetClientCount()
		if count < 0 {
			t.Error("expected non-negative client count")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
