package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"chainpulse/pkg/plugins/api/core"
	"chainpulse/pkg/plugins/api/shared"

	oteltrace "go.opentelemetry.io/otel/trace"
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

func TestHTTPPluginGetRuntimeMetricsPlaintextStopped(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8089, apiLayer)

	metrics := plugin.GetRuntimeMetrics()
	if metrics["transport_posture"] != "http-plaintext-only" {
		t.Errorf("expected http-plaintext-only, got %v", metrics["transport_posture"])
	}
	if metrics["runtime_posture"] != "http-stopped" {
		t.Errorf("expected http-stopped, got %v", metrics["runtime_posture"])
	}
	if metrics["route_posture"] != "http-routes-empty" {
		t.Errorf("expected http-routes-empty, got %v", metrics["route_posture"])
	}
}

func TestHTTPPluginGetRuntimeMetricsTLSServing(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8090, apiLayer)
	plugin.tlsManager = &shared.TLSManager{}

	handler := core.HandlerFunc(func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})
	_ = plugin.RegisterRoute("/health", handler)
	plugin.running = true

	metrics := plugin.GetRuntimeMetrics()
	if metrics["transport_posture"] != "http-tls-enabled" {
		t.Errorf("expected http-tls-enabled, got %v", metrics["transport_posture"])
	}
	if metrics["runtime_posture"] != "http-serving" {
		t.Errorf("expected http-serving, got %v", metrics["runtime_posture"])
	}
	if metrics["route_count"] != 1 {
		t.Errorf("expected route_count 1, got %v", metrics["route_count"])
	}
	if metrics["reliability_hint"] != "http runtime is serving registered routes with a TLS-capable transport" {
		t.Errorf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestHTTPPluginPropagatesTraceContextToNativeHandler(t *testing.T) {
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 8091, apiLayer)

	called := false
	plugin.SetNativeHandler(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if !oteltrace.SpanFromContext(r.Context()).SpanContext().IsValid() {
			t.Fatal("expected inbound request to carry an active OTel span")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	plugin.handleRequest(rr, req)

	if !called {
		t.Fatal("expected native handler to be called")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}
