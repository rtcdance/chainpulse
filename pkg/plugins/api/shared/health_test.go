package shared

import (
	"testing"
	"time"
)

func TestHealthCheckRuntimeSummaryHealthy(t *testing.T) {
	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RecordSuccess("db")

	summary := hc.GetRuntimeSummary()
	if summary["coverage_posture"] != "health-complete" {
		t.Fatalf("expected health-complete, got %v", summary["coverage_posture"])
	}
	if summary["runtime_posture"] != "health-healthy" {
		t.Fatalf("expected health-healthy, got %v", summary["runtime_posture"])
	}
	if summary["reliability_hint"] != "health runtime is healthy across all registered components" {
		t.Fatalf("unexpected reliability hint: %v", summary["reliability_hint"])
	}
}

func TestHealthCheckRuntimeSummaryDegraded(t *testing.T) {
	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 3)
	hc.RecordError("db")

	summary := hc.GetRuntimeSummary()
	if summary["coverage_posture"] != "health-partial" {
		t.Fatalf("expected health-partial, got %v", summary["coverage_posture"])
	}
	if summary["runtime_posture"] != "health-degraded" {
		t.Fatalf("expected health-degraded, got %v", summary["runtime_posture"])
	}
}

func TestHealthCheckRuntimeSummaryUnhealthy(t *testing.T) {
	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 1)
	hc.RecordError("db")

	summary := hc.GetRuntimeSummary()
	if summary["coverage_posture"] != "health-failing" {
		t.Fatalf("expected health-failing, got %v", summary["coverage_posture"])
	}
	if summary["runtime_posture"] != "health-unhealthy" {
		t.Fatalf("expected health-unhealthy, got %v", summary["runtime_posture"])
	}
	if summary["reliability_hint"] != "health runtime is unhealthy; prioritize recovery of failing components" {
		t.Fatalf("unexpected reliability hint: %v", summary["reliability_hint"])
	}
}

func TestHealthCheckRuntimeSummaryUnobserved(t *testing.T) {
	hc := NewHealthCheck()

	summary := hc.GetRuntimeSummary()
	if summary["coverage_posture"] != "health-unconfigured" {
		t.Fatalf("expected health-unconfigured, got %v", summary["coverage_posture"])
	}
	if summary["runtime_posture"] != "health-unobserved" {
		t.Fatalf("expected health-unobserved, got %v", summary["runtime_posture"])
	}
}

func TestHealthCheckHealthSummaryIncludesPostureFields(t *testing.T) {
	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RecordSuccess("db")

	summary := hc.GetHealthSummary()
	if summary["coverage_posture"] != "health-complete" {
		t.Fatalf("expected health-complete coverage, got %v", summary["coverage_posture"])
	}
	if summary["runtime_posture"] != "health-healthy" {
		t.Fatalf("expected health-healthy runtime posture, got %v", summary["runtime_posture"])
	}
	if summary["reliability_hint"] != "health runtime is healthy across all registered components" {
		t.Fatalf("unexpected reliability hint: %v", summary["reliability_hint"])
	}
}

func TestHealthCheckRuntimeMetricsUnconfigured(t *testing.T) {
	hc := NewHealthCheck()

	metrics := hc.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "health-unconfigured" {
		t.Fatalf("expected health-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "health-unobserved" {
		t.Fatalf("expected health-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestHealthCheckRuntimeMetricsPartial(t *testing.T) {
	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 3)
	hc.RegisterComponent("cache", time.Minute, 3)
	hc.RecordSuccess("db")
	hc.RecordError("cache")

	metrics := hc.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "health-partial" {
		t.Fatalf("expected health-partial, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "health-degraded" {
		t.Fatalf("expected health-degraded, got %v", metrics["runtime_posture"])
	}
}

func TestHealthCheckRuntimeMetricsComplete(t *testing.T) {
	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RegisterComponent("cache", time.Minute, 2)
	hc.RecordSuccess("db")
	hc.RecordSuccess("cache")

	metrics := hc.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "health-complete" {
		t.Fatalf("expected health-complete, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "health-healthy" {
		t.Fatalf("expected health-healthy, got %v", metrics["runtime_posture"])
	}
}
