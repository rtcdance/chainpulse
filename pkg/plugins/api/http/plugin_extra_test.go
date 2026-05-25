package http

import (
	"testing"

	"github.com/rtcdance/chainpulse/pkg/observability"
	"github.com/rtcdance/chainpulse/pkg/plugins/api/core"
)

func TestHTTPPlugin_GetProtocolName(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("my-protocol", 9090, apiLayer)

	if got := plugin.GetProtocolName(); got != "my-protocol" {
		t.Errorf("GetProtocolName() = %q, want %q", got, "my-protocol")
	}
}

func TestHTTPPlugin_Handler(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 9091, apiLayer)

	handler := plugin.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
}

func TestHTTPPlugin_SetHTTPSPort_GetHTTPSPort(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 9092, apiLayer)

	if got := plugin.GetHTTPSPort(); got != 9092+443 {
		t.Errorf("default GetHTTPSPort() = %d, want %d", got, 9092+443)
	}

	plugin.SetHTTPSPort(4443)
	if got := plugin.GetHTTPSPort(); got != 4443 {
		t.Errorf("GetHTTPSPort() after SetHTTPSPort = %d, want %d", got, 4443)
	}
}

func TestHTTPPlugin_GetTLSMetrics_NoTLS(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 9093, apiLayer)

	if got := plugin.GetTLSMetrics(); got != nil {
		t.Errorf("GetTLSMetrics() = %v, want nil", got)
	}
}

func TestHTTPPlugin_SetTracer(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 9094, apiLayer)

	plugin.SetTracer(nil)

	tracer := observability.NewDefaultTracer(nil, nil)
	plugin.SetTracer(tracer)
}

func TestClassifyHTTPRuntimePosture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		running    bool
		routeCount int
		expected   string
	}{
		{"stopped", false, 0, "http-stopped"},
		{"stopped with routes", false, 5, "http-stopped"},
		{"running unrouted", true, 0, "http-running-unrouted"},
		{"serving", true, 1, "http-serving"},
		{"serving many", true, 10, "http-serving"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := classifyHTTPRuntimePosture(tt.running, tt.routeCount); got != tt.expected {
				t.Errorf("classifyHTTPRuntimePosture(%v, %d) = %q, want %q", tt.running, tt.routeCount, got, tt.expected)
			}
		})
	}
}

func TestBuildHTTPReliabilityHint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		transportPosture string
		runtimePosture   string
		expected         string
	}{
		{
			name:             "tls serving",
			transportPosture: "http-tls-enabled",
			runtimePosture:   "http-serving",
			expected:         "http runtime is serving registered routes with a TLS-capable transport",
		},
		{
			name:             "plaintext serving",
			transportPosture: "http-plaintext-only",
			runtimePosture:   "http-serving",
			expected:         "http runtime is serving registered routes on a plaintext transport",
		},
		{
			name:             "running unrouted",
			transportPosture: "http-plaintext-only",
			runtimePosture:   "http-running-unrouted",
			expected:         "http runtime is running but no routes are registered yet",
		},
		{
			name:             "tls stopped",
			transportPosture: "http-tls-enabled",
			runtimePosture:   "http-stopped",
			expected:         "http runtime is stopped; restart before relying on TLS-capable route serving",
		},
		{
			name:             "plaintext stopped",
			transportPosture: "http-plaintext-only",
			runtimePosture:   "http-stopped",
			expected:         "http runtime is stopped; restart before relying on route serving",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := buildHTTPReliabilityHint(tt.transportPosture, tt.runtimePosture); got != tt.expected {
				t.Errorf("buildHTTPReliabilityHint(%q, %q) = %q, want %q", tt.transportPosture, tt.runtimePosture, got, tt.expected)
			}
		})
	}
}

func TestHTTPPlugin_GetRuntimeMetrics_RunningUnrouted(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 9095, apiLayer)
	plugin.running = true

	metrics := plugin.GetRuntimeMetrics()
	if metrics["runtime_posture"] != "http-running-unrouted" {
		t.Errorf("expected http-running-unrouted, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "http runtime is running but no routes are registered yet" {
		t.Errorf("unexpected hint: %v", metrics["reliability_hint"])
	}
}

func TestHTTPPlugin_GetRuntimeMetrics_PlaintextServing(t *testing.T) {
	t.Parallel()
	apiLayer := core.NewAPILayer()
	plugin := NewHTTPPlugin("http", 9096, apiLayer)

	handler := core.HandlerFunc(func(req core.Request) (core.Response, error) {
		resp := core.NewBaseResponse(nil)
		resp.SetStatus(200)
		return resp, nil
	})
	_ = plugin.RegisterRoute("/health", handler)
	plugin.running = true

	metrics := plugin.GetRuntimeMetrics()
	if metrics["transport_posture"] != "http-plaintext-only" {
		t.Errorf("expected http-plaintext-only, got %v", metrics["transport_posture"])
	}
	if metrics["runtime_posture"] != "http-serving" {
		t.Errorf("expected http-serving, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "http runtime is serving registered routes on a plaintext transport" {
		t.Errorf("unexpected hint: %v", metrics["reliability_hint"])
	}
}
