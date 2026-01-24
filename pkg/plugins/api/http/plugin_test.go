package http

import (
	"testing"

	"chainpulse/pkg/plugins/api/core"
)

func TestNewHTTPPlugin(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8080, apiLayer)

	if plugin == nil {
		t.Fatal("expected plugin, got nil")
	}

	if plugin.GetName() != "http" {
		t.Errorf("expected name 'http', got %s", plugin.GetName())
	}

	if plugin.port != 8080 {
		t.Errorf("expected port 8080, got %d", plugin.port)
	}
}

func TestHTTPPluginStart(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8081, apiLayer)

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

func TestHTTPPluginStartAlreadyRunning(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8082, apiLayer)

	_ = plugin.Start()
	defer func() { _ = plugin.Stop() }()

	// Try to start again
	err := plugin.Start()
	if err == nil {
		t.Error("expected error when starting already running plugin")
	}
}

func TestHTTPPluginStop(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8083, apiLayer)

	_ = plugin.Start()

	err := plugin.Stop()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if plugin.IsRunning() {
		t.Error("expected plugin to be stopped")
	}
}

func TestHTTPPluginStopNotRunning(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8084, apiLayer)

	err := plugin.Stop()
	if err == nil {
		t.Error("expected error when stopping non-running plugin")
	}
}

func TestHTTPPluginRegisterRoute(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8085, apiLayer)

	handler := core.HandlerFunc(func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	_ = plugin.RegisterRoute("/api/test", handler)

	// Verify route is registered
	route := plugin.router.Route("/api/test")
	if route == nil {
		t.Error("expected route to be registered")
	}
}

func TestHTTPPluginUseMiddleware(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8086, apiLayer)

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

func TestHTTPPluginGetName(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("test-http", 8087, apiLayer)

	if plugin.GetName() != "test-http" {
		t.Errorf("expected name 'test-http', got %s", plugin.GetName())
	}
}

func TestHTTPPluginIsRunning(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8088, apiLayer)

	if plugin.IsRunning() {
		t.Error("expected plugin to not be running initially")
	}

	_ = plugin.Start()
	defer func() { _ = plugin.Stop() }()

	if !plugin.IsRunning() {
		t.Error("expected plugin to be running after start")
	}
}
