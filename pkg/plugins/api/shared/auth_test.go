package shared

import (
	"testing"
	"time"
)

func TestAuthenticationRuntimeMetricsUnconfigured(t *testing.T) {
	auth := NewAuthentication()

	metrics := auth.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "auth-unconfigured" {
		t.Fatalf("expected auth-unconfigured, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-unobserved" {
		t.Fatalf("expected auth-unobserved, got %v", metrics["runtime_posture"])
	}
}

func TestAuthenticationMetricsIncludesPostureFields(t *testing.T) {
	auth := NewAuthentication()
	err := auth.RegisterToken("token-1", "user-1", time.Now().Add(time.Hour), []string{"read"})
	if err != nil {
		t.Fatalf("failed to register token: %v", err)
	}

	metrics := auth.GetMetrics()
	if metrics["coverage_posture"] != "auth-active-only" {
		t.Fatalf("expected auth-active-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-ready" {
		t.Fatalf("expected auth-ready, got %v", metrics["runtime_posture"])
	}
	if metrics["reliability_hint"] != "authentication runtime has active tokens available and no expired-token drift" {
		t.Fatalf("unexpected reliability hint: %v", metrics["reliability_hint"])
	}
}

func TestAuthenticationRuntimeMetricsReady(t *testing.T) {
	auth := NewAuthentication()
	err := auth.RegisterToken("token-1", "user-1", time.Now().Add(time.Hour), []string{"read"})
	if err != nil {
		t.Fatalf("failed to register token: %v", err)
	}

	metrics := auth.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "auth-active-only" {
		t.Fatalf("expected auth-active-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-ready" {
		t.Fatalf("expected auth-ready, got %v", metrics["runtime_posture"])
	}
}

func TestAuthenticationRuntimeMetricsDegraded(t *testing.T) {
	auth := NewAuthentication()
	err := auth.RegisterToken("token-1", "user-1", time.Now().Add(-time.Hour), []string{"read"})
	if err != nil {
		t.Fatalf("failed to register token: %v", err)
	}

	metrics := auth.GetRuntimeMetrics()
	if metrics["coverage_posture"] != "auth-expired-only" {
		t.Fatalf("expected auth-expired-only, got %v", metrics["coverage_posture"])
	}
	if metrics["runtime_posture"] != "auth-degraded" {
		t.Fatalf("expected auth-degraded, got %v", metrics["runtime_posture"])
	}
}
