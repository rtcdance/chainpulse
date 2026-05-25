package shared

import (
	"testing"
	"time"
)

func TestHealthCheckRuntimeSummaryHealthy(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestGetComponentStatus(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RegisterComponent("cache", time.Minute, 3)

	if status := hc.GetComponentStatus("db"); status != StatusHealthy {
		t.Fatalf("expected StatusHealthy, got %v", status)
	}

	hc.RecordError("db")
	if status := hc.GetComponentStatus("db"); status != StatusDegraded {
		t.Fatalf("expected StatusDegraded, got %v", status)
	}

	hc.RecordError("db")
	if status := hc.GetComponentStatus("db"); status != StatusUnhealthy {
		t.Fatalf("expected StatusUnhealthy, got %v", status)
	}

	if status := hc.GetComponentStatus("nonexistent"); status != StatusUnhealthy {
		t.Fatalf("expected StatusUnhealthy for nonexistent, got %v", status)
	}
}

func TestGetComponentMetrics(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RecordSuccess("db")

	metrics := hc.GetComponentMetrics("db")
	if metrics["component"] != "db" {
		t.Fatalf("expected component db, got %v", metrics["component"])
	}
	if metrics["status"] != "healthy" {
		t.Fatalf("expected healthy status, got %v", metrics["status"])
	}
	if metrics["error_threshold"] != 2 {
		t.Fatalf("expected error_threshold 2, got %v", metrics["error_threshold"])
	}

	notFound := hc.GetComponentMetrics("nonexistent")
	if notFound["error"] != "component not found" {
		t.Fatalf("expected 'component not found' error, got %v", notFound["error"])
	}
}

func TestGetComponentMetricsDegraded(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RecordError("db")

	metrics := hc.GetComponentMetrics("db")
	if metrics["status"] != "degraded" {
		t.Fatalf("expected degraded status, got %v", metrics["status"])
	}
	if metrics["consecutive_errors"] != 1 {
		t.Fatalf("expected 1 consecutive error, got %v", metrics["consecutive_errors"])
	}
}

func TestGetAllMetrics(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Minute, 2)
	hc.RegisterComponent("cache", time.Second*30, 3)
	hc.RecordSuccess("db")
	hc.RecordError("cache")

	all := hc.GetAllMetrics()
	if len(all) != 2 {
		t.Fatalf("expected 2 components, got %d", len(all))
	}
	if all["db"]["status"] != "healthy" {
		t.Fatalf("expected db healthy, got %v", all["db"]["status"])
	}
	if all["cache"]["status"] != "degraded" {
		t.Fatalf("expected cache degraded, got %v", all["cache"]["status"])
	}
}

func TestGetAllMetricsEmpty(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	all := hc.GetAllMetrics()
	if len(all) != 0 {
		t.Fatalf("expected 0 components, got %d", len(all))
	}
}

func TestNeedsHealthCheck(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Millisecond, 2)

	if hc.NeedsHealthCheck("nonexistent") {
		t.Fatal("nonexistent component should not need health check")
	}

	if !hc.NeedsHealthCheck("db") {
		t.Fatal("db should need health check immediately after registration")
	}

	hc.RecordSuccess("db")
	if hc.NeedsHealthCheck("db") {
		t.Fatal("db should not need health check right after success")
	}
}

func TestGetComponentsNeedingCheck(t *testing.T) {
	t.Parallel()

	hc := NewHealthCheck()
	hc.RegisterComponent("db", time.Millisecond, 2)
	hc.RegisterComponent("cache", time.Millisecond, 3)

	needing := hc.GetComponentsNeedingCheck()
	if len(needing) != 2 {
		t.Fatalf("expected 2 components needing check, got %d", len(needing))
	}

	hc.RecordSuccess("db")
	needing = hc.GetComponentsNeedingCheck()
	if len(needing) != 1 || needing[0] != "cache" {
		t.Fatalf("expected only cache needing check, got %v", needing)
	}
}

func TestClassifyHealthCoveragePosture(t *testing.T) {
	t.Parallel()

	if p := classifyHealthCoveragePosture(0, 0, 0, 0); p != "health-unconfigured" {
		t.Fatalf("expected health-unconfigured, got %s", p)
	}
	if p := classifyHealthCoveragePosture(2, 0, 0, 1); p != "health-failing" {
		t.Fatalf("expected health-failing, got %s", p)
	}
	if p := classifyHealthCoveragePosture(2, 1, 1, 0); p != "health-partial" {
		t.Fatalf("expected health-partial, got %s", p)
	}
	if p := classifyHealthCoveragePosture(2, 2, 0, 0); p != "health-complete" {
		t.Fatalf("expected health-complete, got %s", p)
	}
	if p := classifyHealthCoveragePosture(2, 0, 0, 0); p != "health-partial" {
		t.Fatalf("expected health-partial for all unknown, got %s", p)
	}
}

func TestClassifyHealthRuntimePosture(t *testing.T) {
	t.Parallel()

	if p := classifyHealthRuntimePosture("healthy", 0, 0, 0); p != "health-unobserved" {
		t.Fatalf("expected health-unobserved, got %s", p)
	}
	if p := classifyHealthRuntimePosture("unhealthy", 2, 0, 1); p != "health-unhealthy" {
		t.Fatalf("expected health-unhealthy with overallStatus=unhealthy, got %s", p)
	}
	if p := classifyHealthRuntimePosture("healthy", 2, 0, 1); p != "health-unhealthy" {
		t.Fatalf("expected health-unhealthy with unhealthyCount>0, got %s", p)
	}
	if p := classifyHealthRuntimePosture("degraded", 2, 1, 0); p != "health-degraded" {
		t.Fatalf("expected health-degraded with overallStatus=degraded, got %s", p)
	}
	if p := classifyHealthRuntimePosture("healthy", 2, 1, 0); p != "health-degraded" {
		t.Fatalf("expected health-degraded with degradedCount>0, got %s", p)
	}
	if p := classifyHealthRuntimePosture("healthy", 2, 0, 0); p != "health-healthy" {
		t.Fatalf("expected health-healthy, got %s", p)
	}
}

func TestBuildHealthReliabilityHint(t *testing.T) {
	t.Parallel()

	if h := buildHealthReliabilityHint("health-healthy"); h != "health runtime is healthy across all registered components" {
		t.Fatalf("unexpected hint: %s", h)
	}
	if h := buildHealthReliabilityHint("health-degraded"); h != "health runtime is degraded; inspect components with consecutive errors before conditions worsen" {
		t.Fatalf("unexpected hint: %s", h)
	}
	if h := buildHealthReliabilityHint("health-unhealthy"); h != "health runtime is unhealthy; prioritize recovery of failing components" {
		t.Fatalf("unexpected hint: %s", h)
	}
	if h := buildHealthReliabilityHint("unknown"); h != "health runtime has not observed any registered components yet" {
		t.Fatalf("unexpected default hint: %s", h)
	}
}
