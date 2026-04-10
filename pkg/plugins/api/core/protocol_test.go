package core

import "testing"

type testProtocolHandler struct {
	name    string
	running bool
}

func (h *testProtocolHandler) GetProtocolName() string { return h.name }
func (h *testProtocolHandler) Start() error {
	h.running = true
	return nil
}

func (h *testProtocolHandler) Stop() error {
	h.running = false
	return nil
}
func (h *testProtocolHandler) IsRunning() bool                                  { return h.running }
func (h *testProtocolHandler) RegisterRoute(path string, handler Handler) error { return nil }
func (h *testProtocolHandler) Use(middleware ...Middleware) error               { return nil }

func TestProtocolRegistryRuntimeMetricsUnobserved(t *testing.T) {
	registry := NewProtocolRegistry()

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "protocol-registry-empty" {
		t.Fatalf("expected protocol-registry-empty, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "protocol-registry-unobserved" {
		t.Fatalf("expected protocol-registry-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestProtocolRegistryRuntimeMetricsWatch(t *testing.T) {
	registry := NewProtocolRegistry()
	if err := registry.Register("http", &testProtocolHandler{name: "http"}); err != nil {
		t.Fatalf("register handler failed: %v", err)
	}

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "protocol-registry-registered" {
		t.Fatalf("expected protocol-registry-registered, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "protocol-registry-watch" {
		t.Fatalf("expected protocol-registry-watch, got %v", metrics["runtime_posture"])
	}
}

func TestProtocolRegistryRuntimeMetricsReady(t *testing.T) {
	registry := NewProtocolRegistry()
	if err := registry.Register("http", &testProtocolHandler{name: "http", running: true}); err != nil {
		t.Fatalf("register handler failed: %v", err)
	}
	if err := registry.Register("grpc", &testProtocolHandler{name: "grpc", running: true}); err != nil {
		t.Fatalf("register handler failed: %v", err)
	}

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "protocol-registry-active" {
		t.Fatalf("expected protocol-registry-active, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "protocol-registry-ready" {
		t.Fatalf("expected protocol-registry-ready, got %v", metrics["runtime_posture"])
	}
}

func TestDefaultRequestProcessorRuntimeMetricsUnobserved(t *testing.T) {
	processor := NewDefaultRequestProcessor(nil)

	metrics := processor.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-processor-unconfigured" {
		t.Fatalf("expected request-processor-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-processor-unobserved" {
		t.Fatalf("expected request-processor-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestDefaultRequestProcessorRuntimeMetricsWatch(t *testing.T) {
	layer := NewAPILayer()
	layer.RegisterHandlerFunc("/health", func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})
	layer.SetErrorMapper(nil)

	processor := NewDefaultRequestProcessor(layer)
	metrics := processor.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-processor-routes-only" {
		t.Fatalf("expected request-processor-routes-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-processor-watch" {
		t.Fatalf("expected request-processor-watch, got %v", metrics["runtime_posture"])
	}
}

func TestDefaultRequestProcessorRuntimeMetricsReady(t *testing.T) {
	layer := NewAPILayer()
	layer.Use(func(next Handler) Handler { return next })
	layer.RegisterHandlerFunc("/health", func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})

	processor := NewDefaultRequestProcessor(layer)
	metrics := processor.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-processor-guarded" {
		t.Fatalf("expected request-processor-guarded, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-processor-ready" {
		t.Fatalf("expected request-processor-ready, got %v", metrics["runtime_posture"])
	}
}

func TestBaseProtocolHandlerRuntimeMetricsUnobserved(t *testing.T) {
	handler := NewBaseProtocolHandler("http", nil)
	handler.router = NewAPIRouter()

	metrics := handler.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "protocol-handler-unconfigured" {
		t.Fatalf("expected protocol-handler-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "protocol-handler-unobserved" {
		t.Fatalf("expected protocol-handler-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestBaseProtocolHandlerRuntimeMetricsIdle(t *testing.T) {
	handler := NewBaseProtocolHandler("http", NewDefaultRequestProcessor(NewAPILayer()))
	if err := handler.RegisterRoute("/health", HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}

	metrics := handler.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "protocol-handler-routes-only" {
		t.Fatalf("expected protocol-handler-routes-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "protocol-handler-idle" {
		t.Fatalf("expected protocol-handler-idle, got %v", metrics["runtime_posture"])
	}
}

func TestBaseProtocolHandlerRuntimeMetricsReady(t *testing.T) {
	handler := NewBaseProtocolHandler("http", NewDefaultRequestProcessor(NewAPILayer()))
	if err := handler.Use(func(next Handler) Handler { return next }); err != nil {
		t.Fatalf("use middleware failed: %v", err)
	}
	if err := handler.RegisterRoute("/health", HandlerFunc(func(req Request) (Response, error) {
		resp := NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})); err != nil {
		t.Fatalf("register route failed: %v", err)
	}
	handler.SetRunning(true)

	metrics := handler.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "protocol-handler-guarded" {
		t.Fatalf("expected protocol-handler-guarded, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "protocol-handler-ready" {
		t.Fatalf("expected protocol-handler-ready, got %v", metrics["runtime_posture"])
	}
}
