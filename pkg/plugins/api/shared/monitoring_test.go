package shared

import (
	"testing"
	"time"
)

func TestMonitoringMetricsIncludesPostureFields(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()
	monitoring.RecordRequest("http", 50*time.Millisecond, true)
	monitoring.RecordRequest("http", 70*time.Millisecond, true)

	metrics := monitoring.GetMetrics("http")
	if metrics["coverage_posture"] != "monitoring-success-only" {
		t.Fatalf("expected monitoring-success-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "monitoring-healthy" {
		t.Fatalf("expected monitoring-healthy, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "protocol monitoring is observing successful traffic with healthy runtime posture" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestMonitoringRuntimeMetricsUnobserved(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()

	monitoring.RecordRequest("http", 0, true)
	monitoring.ResetMetrics("http")

	metrics := monitoring.GetProtocolRuntimeMetrics("http")
	if metrics["coverage_posture"] != "monitoring-unobserved" {
		t.Fatalf("expected monitoring-unobserved, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "monitoring-unobserved" {
		t.Fatalf("expected monitoring-unobserved runtime, got %v", metrics["runtime_posture"])
	}
}

func TestMonitoringRuntimeMetricsHealthy(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()

	monitoring.RecordRequest("http", 50*time.Millisecond, true)
	monitoring.RecordRequest("http", 70*time.Millisecond, true)

	metrics := monitoring.GetProtocolRuntimeMetrics("http")
	if metrics["coverage_posture"] != "monitoring-success-only" {
		t.Fatalf("expected monitoring-success-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "monitoring-healthy" {
		t.Fatalf("expected monitoring-healthy, got %v", metrics["runtime_posture"])
	}
}

func TestMonitoringRuntimeMetricsDegraded(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()

	monitoring.RecordRequest("grpc", 200*time.Millisecond, false)
	monitoring.RecordRequest("grpc", 300*time.Millisecond, false)

	metrics := monitoring.GetProtocolRuntimeMetrics("grpc")
	if metrics["coverage_posture"] != "monitoring-fail-only" {
		t.Fatalf("expected monitoring-fail-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "monitoring-degraded" {
		t.Fatalf("expected monitoring-degraded, got %v", metrics["runtime_posture"])
	}
}

func TestMonitoringAggregateRuntimeMetricsUnobserved(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()

	metrics := monitoring.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "monitoring-unobserved" {
		t.Fatalf("expected monitoring-unobserved, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "monitoring-unobserved" {
		t.Fatalf("expected monitoring-unobserved runtime, got %v", metrics["runtime_posture"])
	}
}

func TestMonitoringAggregateRuntimeMetricsHealthy(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()
	monitoring.RecordRequest("http", 80*time.Millisecond, true)
	monitoring.RecordRequest("grpc", 120*time.Millisecond, true)

	metrics := monitoring.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "monitoring-success-only" {
		t.Fatalf("expected monitoring-success-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "monitoring-healthy" {
		t.Fatalf("expected monitoring-healthy, got %v", metrics["runtime_posture"])
	}
}

func TestMonitoring_GetAllMetrics(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()
	monitoring.RecordRequest("http", 50*time.Millisecond, true)
	monitoring.RecordRequest("grpc", 100*time.Millisecond, true)

	all := monitoring.GetAllMetrics()
	if len(all) != 2 {
		t.Fatalf("expected 2 protocols, got %d", len(all))
	}
	if _, ok := all["http"]; !ok {
		t.Fatal("expected http protocol in all metrics")
	}
	if _, ok := all["grpc"]; !ok {
		t.Fatal("expected grpc protocol in all metrics")
	}
}

func TestMonitoringAggregateRuntimeMetricsDegraded(t *testing.T) {
	t.Parallel()
	monitoring := NewMonitoring()
	monitoring.RecordRequest("http", 100*time.Millisecond, false)
	monitoring.RecordRequest("grpc", 120*time.Millisecond, false)
	monitoring.RecordRequest("ws", 140*time.Millisecond, true)

	metrics := monitoring.GetRuntimeMetrics()
	if metrics["runtime_posture"] != "monitoring-degraded" {
		t.Fatalf("expected monitoring-degraded, got %v", metrics["runtime_posture"])
	}
}
