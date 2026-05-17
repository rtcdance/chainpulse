package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/rtcdance/chainpulse/pkg/plugins/api/core"
)

func skipGRPCPluginLifecycleTestsInShortMode(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping gRPC plugin lifecycle test in short mode")
	}
}

func TestNewGRPCPlugin(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9090, apiLayer)

	if plugin == nil {
		t.Fatal("expected plugin, got nil")
	}

	if plugin.GetName() != "grpc" {
		t.Errorf("expected name 'grpc', got %s", plugin.GetName())
	}

	if plugin.port != 9090 {
		t.Errorf("expected port 9090, got %d", plugin.port)
	}
}

func TestGRPCPluginStart(t *testing.T) {
	t.Parallel()
	skipGRPCPluginLifecycleTestsInShortMode(t)

	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9091, apiLayer)

	err := plugin.Start()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running")
	}

	// Clean up
	_ = plugin.Stop()
}

func TestGRPCPluginStartAlreadyRunning(t *testing.T) {
	t.Parallel()
	skipGRPCPluginLifecycleTestsInShortMode(t)

	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9092, apiLayer)

	_ = plugin.Start()
	defer func() { _ = plugin.Stop() }()

	// Try to start again
	err := plugin.Start()
	if err == nil {
		t.Error("expected error when starting already running plugin")
	}
}

func TestGRPCPluginStop(t *testing.T) {
	t.Parallel()
	skipGRPCPluginLifecycleTestsInShortMode(t)

	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9093, apiLayer)

	_ = plugin.Start()

	err := plugin.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if plugin.IsRunning() {
		t.Error("expected plugin to be stopped")
	}
}

func TestGRPCPluginStopNotRunning(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9094, apiLayer)

	err := plugin.Stop()
	if err == nil {
		t.Error("expected error when stopping non-running plugin")
	}
}

func TestGRPCPluginGetName(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("test-grpc", 9095, apiLayer)

	if plugin.GetName() != "test-grpc" {
		t.Errorf("expected name 'test-grpc', got %s", plugin.GetName())
	}
}

func TestGRPCPluginIsRunning(t *testing.T) {
	t.Parallel()
	skipGRPCPluginLifecycleTestsInShortMode(t)

	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9096, apiLayer)

	if plugin.IsRunning() {
		t.Error("expected plugin to not be running initially")
	}

	_ = plugin.Start()
	defer func() { _ = plugin.Stop() }()

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running after start")
	}
}

func TestGRPCPluginUseMiddleware(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9097, apiLayer)

	middleware := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-Middleware", "applied")
			}
			return resp, err
		})
	}

	_ = plugin.Use(middleware)

	if len(plugin.middleware) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(plugin.middleware))
	}
}

func TestGRPCPluginMultipleMiddleware(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9098, apiLayer)

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

	_ = plugin.Use(middleware1, middleware2)

	if len(plugin.middleware) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(plugin.middleware))
	}
}

func TestGRPCPluginProcessRequestExecutesMiddleware(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9101, apiLayer)

	if err := plugin.RegisterRoute("/grpc.test", core.HandlerFunc(func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		resp.SetBody([]byte("ok"))
		return resp, nil
	})); err != nil {
		t.Fatalf("register route: %v", err)
	}

	middleware := func(next core.Handler) core.Handler {
		return core.HandlerFunc(func(req core.Request) (core.Response, error) {
			resp, err := next.Handle(req)
			if err == nil {
				resp.SetHeader("X-GRPC-Middleware", "applied")
			}
			return resp, err
		})
	}

	if err := plugin.Use(middleware); err != nil {
		t.Fatalf("use middleware: %v", err)
	}

	resp, err := plugin.ProcessRequest(context.Background(), "POST", "/grpc.test", nil, []byte(`{}`))
	if err != nil {
		t.Fatalf("process request: %v", err)
	}
	if resp.Status() != 200 {
		t.Fatalf("expected status 200, got %d", resp.Status())
	}
	if got := resp.Header("X-GRPC-Middleware"); got != "applied" {
		t.Fatalf("expected middleware header applied, got %q", got)
	}
}

func TestGRPCPluginConcurrentOperations(t *testing.T) {
	t.Parallel()
	skipGRPCPluginLifecycleTestsInShortMode(t)

	apiLayer := core.NewAPILayer()
	plugin := NewGRPCPlugin("grpc", 9099, apiLayer)

	// Start plugin
	_ = plugin.Start()
	defer func() { _ = plugin.Stop() }()

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
		server := plugin.GetServer()
		if server == nil {
			t.Error("expected server to be non-nil")
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
