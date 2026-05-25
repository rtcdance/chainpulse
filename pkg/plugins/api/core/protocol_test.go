package core

import (
	"fmt"
	"testing"
)

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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestProtocolRegistry_Get(t *testing.T) {
	t.Parallel()

	registry := NewProtocolRegistry()
	h := &testProtocolHandler{name: "http"}
	_ = registry.Register("http", h)

	got, err := registry.Get("http")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.GetProtocolName() != "http" {
		t.Errorf("expected http, got %s", got.GetProtocolName())
	}

	_, err = registry.Get("unknown")
	if err == nil {
		t.Fatal("expected error for unknown handler")
	}
}

func TestProtocolRegistry_List(t *testing.T) {
	t.Parallel()

	registry := NewProtocolRegistry()

	names := registry.List()
	if len(names) != 0 {
		t.Errorf("expected 0 names, got %d", len(names))
	}

	_ = registry.Register("http", &testProtocolHandler{name: "http"})
	_ = registry.Register("grpc", &testProtocolHandler{name: "grpc"})

	names = registry.List()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestProtocolRegistry_StartAll(t *testing.T) {
	t.Parallel()

	registry := NewProtocolRegistry()
	h1 := &testProtocolHandler{name: "http"}
	h2 := &testProtocolHandler{name: "grpc"}
	_ = registry.Register("http", h1)
	_ = registry.Register("grpc", h2)

	err := registry.StartAll()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !h1.IsRunning() {
		t.Error("expected http handler to be running")
	}
	if !h2.IsRunning() {
		t.Error("expected grpc handler to be running")
	}
}

func TestProtocolRegistry_StopAll(t *testing.T) {
	t.Parallel()

	registry := NewProtocolRegistry()
	h1 := &testProtocolHandler{name: "http", running: true}
	h2 := &testProtocolHandler{name: "grpc", running: true}
	_ = registry.Register("http", h1)
	_ = registry.Register("grpc", h2)

	err := registry.StopAll()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if h1.IsRunning() {
		t.Error("expected http handler to be stopped")
	}
	if h2.IsRunning() {
		t.Error("expected grpc handler to be stopped")
	}
}

func TestDefaultRequestProcessor_ProcessRequest(t *testing.T) {
	t.Parallel()

	t.Run("nil api layer", func(t *testing.T) {
		t.Parallel()
		p := NewDefaultRequestProcessor(nil)
		_, err := p.ProcessRequest(nil, nil)
		if err == nil {
			t.Fatal("expected error for nil api layer")
		}
	})
}

func TestDefaultRequestProcessor_HandleError(t *testing.T) {
	t.Parallel()

	p := NewDefaultRequestProcessor(nil)
	resp := p.HandleError(nil, fmt.Errorf("test error"))
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestBaseProtocolHandler_GetProtocolName(t *testing.T) {
	t.Parallel()

	h := NewBaseProtocolHandler("my-protocol", nil)
	if h.GetProtocolName() != "my-protocol" {
		t.Errorf("expected my-protocol, got %s", h.GetProtocolName())
	}
}

func TestBaseProtocolHandler_IsRunning(t *testing.T) {
	t.Parallel()

	h := NewBaseProtocolHandler("http", nil)
	if h.IsRunning() {
		t.Error("expected false for new handler")
	}

	h.SetRunning(true)
	if !h.IsRunning() {
		t.Error("expected true after SetRunning(true)")
	}
}

func TestBaseProtocolHandler_GetRouter(t *testing.T) {
	t.Parallel()

	h := NewBaseProtocolHandler("http", nil)
	router := h.GetRouter()
	if router == nil {
		t.Fatal("expected non-nil router")
	}
}

func TestBaseProtocolHandler_GetProcessor(t *testing.T) {
	t.Parallel()

	proc := NewDefaultRequestProcessor(nil)
	h := NewBaseProtocolHandler("http", proc)
	got := h.GetProcessor()
	if got == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestClassifyProtocolRegistryCoveragePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		handlerCount int
		runningCount int
		expected     string
	}{
		{"empty", 0, 0, "protocol-registry-empty"},
		{"registered", 2, 0, "protocol-registry-registered"},
		{"partial", 3, 1, "protocol-registry-partial"},
		{"active", 2, 2, "protocol-registry-active"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyProtocolRegistryCoveragePosture(tt.handlerCount, tt.runningCount)
			if got != tt.expected {
				t.Errorf("classifyProtocolRegistryCoveragePosture(%d, %d) = %q, want %q",
					tt.handlerCount, tt.runningCount, got, tt.expected)
			}
		})
	}
}

func TestClassifyProtocolRegistryRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		handlerCount int
		runningCount int
		expected     string
	}{
		{"unobserved", 0, 0, "protocol-registry-unobserved"},
		{"watch", 2, 0, "protocol-registry-watch"},
		{"degraded", 3, 1, "protocol-registry-degraded"},
		{"ready", 2, 2, "protocol-registry-ready"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyProtocolRegistryRuntimePosture(tt.handlerCount, tt.runningCount)
			if got != tt.expected {
				t.Errorf("classifyProtocolRegistryRuntimePosture(%d, %d) = %q, want %q",
					tt.handlerCount, tt.runningCount, got, tt.expected)
			}
		})
	}
}

func TestBuildProtocolRegistryReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coveragePosture string
		runtimePosture  string
		expected        string
	}{
		{
			name:            "unobserved",
			coveragePosture: "protocol-registry-empty",
			runtimePosture:  "protocol-registry-unobserved",
			expected:        "protocol registry has no registered handlers yet",
		},
		{
			name:            "watch",
			coveragePosture: "protocol-registry-registered",
			runtimePosture:  "protocol-registry-watch",
			expected:        "protocol registry has registered handlers but none are running; verify start sequencing before treating it as active",
		},
		{
			name:            "degraded",
			coveragePosture: "protocol-registry-partial",
			runtimePosture:  "protocol-registry-degraded",
			expected:        "protocol registry has only partial running coverage; verify whether all registered handlers are expected to be active",
		},
		{
			name:            "active",
			coveragePosture: "protocol-registry-active",
			runtimePosture:  "protocol-registry-ready",
			expected:        "protocol registry has full running coverage across registered handlers and is ready for protocol dispatch",
		},
		{
			name:            "default",
			coveragePosture: "unknown",
			runtimePosture:  "unknown",
			expected:        "protocol registry has registered handlers and requires runtime observation",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildProtocolRegistryReliabilityHint(tt.coveragePosture, tt.runtimePosture)
			if got != tt.expected {
				t.Errorf("buildProtocolRegistryReliabilityHint(%q, %q) = %q, want %q",
					tt.coveragePosture, tt.runtimePosture, got, tt.expected)
			}
		})
	}
}

func TestClassifyRequestProcessorCoveragePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		apiLayerConfigured   bool
		layerCoveragePosture string
		expected             string
	}{
		{"unconfigured", false, "", "request-processor-unconfigured"},
		{"guarded", true, "layer-guarded", "request-processor-guarded"},
		{"routes only", true, "layer-routes-only", "request-processor-routes-only"},
		{"partial", true, "unknown", "request-processor-partial"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyRequestProcessorCoveragePosture(tt.apiLayerConfigured, tt.layerCoveragePosture)
			if got != tt.expected {
				t.Errorf("classifyRequestProcessorCoveragePosture(%v, %q) = %q, want %q",
					tt.apiLayerConfigured, tt.layerCoveragePosture, got, tt.expected)
			}
		})
	}
}

func TestClassifyRequestProcessorRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		apiLayerConfigured    bool
		routeCount            int
		errorMapperConfigured bool
		expected              string
	}{
		{"unobserved", false, 0, false, "request-processor-unobserved"},
		{"watch no routes", true, 0, true, "request-processor-watch"},
		{"watch no error mapper", true, 5, false, "request-processor-watch"},
		{"ready", true, 5, true, "request-processor-ready"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyRequestProcessorRuntimePosture(tt.apiLayerConfigured, tt.routeCount, tt.errorMapperConfigured)
			if got != tt.expected {
				t.Errorf("classifyRequestProcessorRuntimePosture(%v, %d, %v) = %q, want %q",
					tt.apiLayerConfigured, tt.routeCount, tt.errorMapperConfigured, got, tt.expected)
			}
		})
	}
}

func TestBuildRequestProcessorReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coveragePosture string
		runtimePosture  string
		expected        string
	}{
		{
			name:            "unobserved",
			coveragePosture: "request-processor-unconfigured",
			runtimePosture:  "request-processor-unobserved",
			expected:        "request processor has no API layer configured yet",
		},
		{
			name:            "watch",
			coveragePosture: "request-processor-partial",
			runtimePosture:  "request-processor-watch",
			expected:        "request processor wiring is incomplete; verify API routes and error mapping before relying on runtime behavior",
		},
		{
			name:            "routes only",
			coveragePosture: "request-processor-routes-only",
			runtimePosture:  "request-processor-ready",
			expected:        "request processor has routed API coverage without middleware guardrails; verify whether bare processing is intentional",
		},
		{
			name:            "guarded",
			coveragePosture: "request-processor-guarded",
			runtimePosture:  "request-processor-ready",
			expected:        "request processor is backed by routed, guarded API-layer coverage and is ready for request handling",
		},
		{
			name:            "default",
			coveragePosture: "unknown",
			runtimePosture:  "unknown",
			expected:        "request processor has partial API-layer coverage; continue observing runtime wiring",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildRequestProcessorReliabilityHint(tt.coveragePosture, tt.runtimePosture)
			if got != tt.expected {
				t.Errorf("buildRequestProcessorReliabilityHint(%q, %q) = %q, want %q",
					tt.coveragePosture, tt.runtimePosture, got, tt.expected)
			}
		})
	}
}

func TestClassifyBaseProtocolHandlerCoveragePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		routeCount          int
		middlewareCount     int
		processorConfigured bool
		expected            string
	}{
		{"unconfigured", 0, 0, false, "protocol-handler-unconfigured"},
		{"routes only", 3, 0, true, "protocol-handler-routes-only"},
		{"guarded", 3, 2, true, "protocol-handler-guarded"},
		{"partial", 0, 2, true, "protocol-handler-partial"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyBaseProtocolHandlerCoveragePosture(tt.routeCount, tt.middlewareCount, tt.processorConfigured)
			if got != tt.expected {
				t.Errorf("classifyBaseProtocolHandlerCoveragePosture(%d, %d, %v) = %q, want %q",
					tt.routeCount, tt.middlewareCount, tt.processorConfigured, got, tt.expected)
			}
		})
	}
}

func TestClassifyBaseProtocolHandlerRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		running             bool
		routeCount          int
		processorConfigured bool
		expected            string
	}{
		{"unobserved", false, 0, false, "protocol-handler-unobserved"},
		{"degraded", true, 3, false, "protocol-handler-degraded"},
		{"idle", false, 3, true, "protocol-handler-idle"},
		{"ready", true, 3, true, "protocol-handler-ready"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyBaseProtocolHandlerRuntimePosture(tt.running, tt.routeCount, tt.processorConfigured)
			if got != tt.expected {
				t.Errorf("classifyBaseProtocolHandlerRuntimePosture(%v, %d, %v) = %q, want %q",
					tt.running, tt.routeCount, tt.processorConfigured, got, tt.expected)
			}
		})
	}
}

func TestBuildBaseProtocolHandlerReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		coveragePosture string
		runtimePosture  string
		expected        string
	}{
		{
			name:            "degraded",
			coveragePosture: "protocol-handler-unconfigured",
			runtimePosture:  "protocol-handler-degraded",
			expected:        "protocol handler is missing a request processor; configure processing before relying on runtime behavior",
		},
		{
			name:            "idle",
			coveragePosture: "protocol-handler-routes-only",
			runtimePosture:  "protocol-handler-idle",
			expected:        "protocol handler has wiring but is not running yet; verify start sequencing before treating it as active",
		},
		{
			name:            "routes only",
			coveragePosture: "protocol-handler-routes-only",
			runtimePosture:  "protocol-handler-ready",
			expected:        "protocol handler has routes without middleware guardrails; verify whether bare routing is intentional",
		},
		{
			name:            "guarded ready",
			coveragePosture: "protocol-handler-guarded",
			runtimePosture:  "protocol-handler-ready",
			expected:        "protocol handler has processor, routes, middleware, and running state aligned for request handling",
		},
		{
			name:            "default",
			coveragePosture: "unknown",
			runtimePosture:  "unknown",
			expected:        "protocol handler has not been fully configured yet",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := buildBaseProtocolHandlerReliabilityHint(tt.coveragePosture, tt.runtimePosture)
			if got != tt.expected {
				t.Errorf("buildBaseProtocolHandlerReliabilityHint(%q, %q) = %q, want %q",
					tt.coveragePosture, tt.runtimePosture, got, tt.expected)
			}
		})
	}
}
