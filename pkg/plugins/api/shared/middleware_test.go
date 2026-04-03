package shared

import (
	"testing"

	"chainpulse/pkg/plugins/api/core"
)

func noopMiddleware(next core.Handler) core.Handler {
	return next
}

func TestMiddlewareRegistryRuntimeMetricsUnconfigured(t *testing.T) {
	registry := NewMiddlewareRegistry()
	registry.security = nil
	registry.observability = nil
	registry.performance = nil

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "middleware-unconfigured" {
		t.Fatalf("expected middleware-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "middleware-unobserved" {
		t.Fatalf("expected middleware-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestMiddlewareRegistryRuntimeMetricsPartial(t *testing.T) {
	registry := NewMiddlewareRegistry()
	registry.GetSecurityGroup().SetAuthMiddleware(noopMiddleware)
	registry.GetObservabilityGroup().SetHealthMiddleware(noopMiddleware)

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "middleware-partial" {
		t.Fatalf("expected middleware-partial, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "middleware-watch" {
		t.Fatalf("expected middleware-watch, got %v", metrics["runtime_posture"])
	}
}

func TestMiddlewareRegistryRuntimeMetricsReady(t *testing.T) {
	registry := NewMiddlewareRegistry()
	registry.GetSecurityGroup().SetAuthMiddleware(noopMiddleware)
	registry.GetSecurityGroup().SetTLSMiddleware(noopMiddleware)
	registry.GetObservabilityGroup().SetHealthMiddleware(noopMiddleware)
	registry.GetObservabilityGroup().SetMonitoringMiddleware(noopMiddleware)
	registry.GetPerformanceGroup().SetCompressionMiddleware(noopMiddleware)
	registry.GetPerformanceGroup().SetBatchingMiddleware(noopMiddleware)
	registry.GetPerformanceGroup().SetPoolMiddleware(noopMiddleware)
	registry.SetErrorHandling(NewErrorHandlingMiddleware(nil))

	metrics := registry.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "middleware-full-stack" {
		t.Fatalf("expected middleware-full-stack, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "middleware-ready" {
		t.Fatalf("expected middleware-ready, got %v", metrics["runtime_posture"])
	}
}
