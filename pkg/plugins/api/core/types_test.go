package core

import "testing"

func TestRequestMetadataRuntimeMetricsUnobserved(t *testing.T) {
	t.Parallel()
	metadata := RequestMetadata{
		Protocol: ProtocolUnknown,
	}

	metrics := metadata.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-metadata-unconfigured" {
		t.Fatalf("expected request-metadata-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-metadata-unobserved" {
		t.Fatalf("expected request-metadata-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestRequestMetadataRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	metadata := RequestMetadata{
		Protocol:      ProtocolHTTP,
		ClientIP:      "127.0.0.1",
		UserAgent:     "unit-test",
		RequestID:     "req-1",
		Timestamp:     1711900800,
		ContentLength: 32,
	}

	metrics := metadata.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "request-metadata-attributed" {
		t.Fatalf("expected request-metadata-attributed, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "request-metadata-ready" {
		t.Fatalf("expected request-metadata-ready, got %v", metrics["runtime_posture"])
	}
}

func TestResponseMetadataRuntimeMetricsWatch(t *testing.T) {
	t.Parallel()
	metadata := ResponseMetadata{
		Protocol:      ProtocolHTTP,
		ContentLength: 128,
		Duration:      0,
		Timestamp:     0,
	}

	metrics := metadata.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "response-metadata-partial" {
		t.Fatalf("expected response-metadata-partial, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "response-metadata-watch" {
		t.Fatalf("expected response-metadata-watch, got %v", metrics["runtime_posture"])
	}
}

func TestResponseMetadataRuntimeMetricsReady(t *testing.T) {
	t.Parallel()
	metadata := ResponseMetadata{
		Protocol:      ProtocolGRPC,
		ContentLength: 512,
		Duration:      18,
		Timestamp:     1711900818,
	}

	metrics := metadata.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "response-metadata-profiled" {
		t.Fatalf("expected response-metadata-profiled, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "response-metadata-ready" {
		t.Fatalf("expected response-metadata-ready, got %v", metrics["runtime_posture"])
	}
}
