package shared

import (
	"testing"
	"time"
)

func TestErrorHandlerRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	handler := NewErrorHandler()

	metrics := handler.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "circuit-ready" {
		t.Fatalf("expected circuit-ready coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["circuit_posture"] != "circuit-ready" {
		t.Fatalf("expected circuit-ready, got %v", metrics["circuit_posture"])
	}
	if metrics["retry_posture"] != "retry-ready" {
		t.Fatalf("expected retry-ready, got %v", metrics["retry_posture"])
	}
}

func TestErrorHandlerMetricsIncludesPostureFields(t *testing.T) {
	t.Parallel()
	handler := NewErrorHandler()

	metrics := handler.GetMetrics()
	if metrics["coverage_posture"] != "circuit-ready" {
		t.Fatalf("expected circuit-ready coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["circuit_posture"] != "circuit-ready" {
		t.Fatalf("expected circuit-ready circuit posture, got %v", metrics["circuit_posture"])
	}
	if metrics["retry_posture"] != "retry-ready" {
		t.Fatalf("expected retry-ready, got %v", metrics["retry_posture"])
	}
	if metrics["reliability_hint"] != "error handler runtime is ready with circuit breaker closed and retry policy available" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestErrorHandlerRuntimeMetricsOpenCircuit(t *testing.T) {
	t.Parallel()
	handler := NewErrorHandler()

	for i := 0; i < 5; i++ {
		handler.circuitBreaker.RecordFailure()
	}

	metrics := handler.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "circuit-open" {
		t.Fatalf("expected circuit-open coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["circuit_posture"] != "circuit-open" {
		t.Fatalf("expected circuit-open, got %v", metrics["circuit_posture"])
	}
	if metrics["retry_posture"] != "retry-exhausted" {
		t.Fatalf("expected retry-exhausted, got %v", metrics["retry_posture"])
	}
}

func TestErrorHandlerRuntimeMetricsProbing(t *testing.T) {
	t.Parallel()
	handler := NewErrorHandler()
	handler.circuitBreaker.state = StateHalfOpen
	handler.circuitBreaker.lastFailureTime = time.Now()
	handler.circuitBreaker.failureCount = 1

	metrics := handler.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "circuit-probing" {
		t.Fatalf("expected circuit-probing coverage, got %v", metrics["coverage_posture"])
	}
	if metrics["circuit_posture"] != "circuit-probing" {
		t.Fatalf("expected circuit-probing, got %v", metrics["circuit_posture"])
	}
	if metrics["retry_posture"] != "retry-engaged" {
		t.Fatalf("expected retry-engaged, got %v", metrics["retry_posture"])
	}
}
