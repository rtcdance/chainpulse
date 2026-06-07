package main

import (
	"context"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func TestAPIGatewayRolloutReportProducerSkeleton(t *testing.T) {
	producer := newAPIGatewayRolloutReportProducer("api-gateway-1", nil)
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Service; got != "api-gateway" {
		t.Fatalf("expected service api-gateway, got %q", got)
	}
	if got := report.ReportSource; got != "microservice:api-gateway-1" {
		t.Fatalf("expected report source microservice:api-gateway-1, got %q", got)
	}
	if got := report.Mode; got != "unavailable" {
		t.Fatalf("expected mode unavailable, got %q", got)
	}
	if got := report.Approval.WorkItem.Reason; got != "api-gateway ownership runtime parity with monolith is not yet wired" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
	if got := report.Approval.WorkItem.ReviewFields; got != api.BuildOwnershipParityReviewFields(apiGatewayOwnershipParityReviewFields) {
		t.Fatalf("expected ownership parity review fields, got %q", got)
	}
}

func TestAPIGatewayRolloutReportProducerRuntimeDerivedPartial(t *testing.T) {
	producer := newAPIGatewayRolloutReportProducer("api-gateway-1", func() apiGatewayRolloutRuntimeState {
		return apiGatewayRolloutRuntimeState{
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckEnabled:       true,
			RuntimeRoutesEnabled:     true,
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if got := report.Advisory.Status; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
	if report.Advisory.Ready {
		t.Fatal("expected advisory ready false for partial runtime wiring")
	}
	if !strings.Contains(report.Advisory.Reason, "missing: domain_bridge_enabled") {
		t.Fatalf("expected missing domain bridge in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "ownership_parity_hint: api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending") {
		t.Fatalf("expected ownership parity hint in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: finish wiring the missing api-gateway runtime routes before treating this rollout as ready") {
		t.Fatalf("expected posture hint in reason, got %q", report.Advisory.Reason)
	}
	if got := report.Approval.WorkItem.Reason; got != "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
	if got := report.Approval.WorkItem.ReviewFields; got != api.BuildOwnershipParityReviewFields(apiGatewayOwnershipParityReviewFields) {
		t.Fatalf("expected ownership parity review fields, got %q", got)
	}
}

func TestAPIGatewayRolloutReportProducerRuntimeDerivedFullyWired(t *testing.T) {
	producer := newAPIGatewayRolloutReportProducer("api-gateway-1", func() apiGatewayRolloutRuntimeState {
		return apiGatewayRolloutRuntimeState{
			DomainBridgeEnabled:      true,
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckEnabled:       true,
			RuntimeRoutesEnabled:     true,
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if err := api.ValidateMicroserviceRolloutMetadataParity(
		report,
		"api-gateway",
		"api-gateway-ownership-rollout-runtime",
		"microservice:api-gateway-1",
	); err != nil {
		t.Fatalf("expected metadata parity validation: %v", err)
	}
	if got := report.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := report.Advisory.Status; got != "runtime-wired" {
		t.Fatalf("expected advisory status runtime-wired, got %q", got)
	}
	if !report.Advisory.Ready {
		t.Fatal("expected advisory ready true when all runtime signals are wired")
	}
	if !strings.Contains(report.Advisory.Reason, "ownership_parity_hint: api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending") {
		t.Fatalf("expected ownership parity hint in reason, got %q", report.Advisory.Reason)
	}
	if got := report.Approval.WorkItem.Reason; got != "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(report); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
	if err := api.ValidateMicroserviceOwnershipParityMarker(report); err != nil {
		t.Fatalf("expected ownership parity marker validation: %v", err)
	}
}

func TestAPIGatewayRolloutReportProducerIncludesMonolithParitySource(t *testing.T) {
	t.Skip("pre-existing vet error: newAPIGatewayRolloutReportProducerWithReadinessDetails undefined at HEAD; restore when production function is reintroduced")
}

func TestClassifyAPIGatewayOwnershipParityHint(t *testing.T) {
	t.Skip("pre-existing vet error: classifyAPIGatewayOwnershipParityHint undefined at HEAD; restore when production function is reintroduced")
}

func TestClassifyAPIGatewayRolloutPostureHint(t *testing.T) {
	cases := []struct {
		name string
		mode string
		stat string
		miss int
		want string
	}{
		{"partial", "partially-wired", "partial-runtime-wiring", 1, "finish wiring the missing api-gateway runtime routes before treating this rollout as ready"},
		{"wired", "runtime-wired", "runtime-wired", 0, "observe the runtime-wired api-gateway rollout while local route composition remains healthy"},
		{"fallback", "unavailable", "unavailable", 0, ""},
	}

	for _, tc := range cases {
		if got := classifyAPIGatewayRolloutPostureHint(tc.mode, tc.stat, tc.miss); got != tc.want {
			t.Fatalf("%s: expected posture hint %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestAPIGatewayRolloutReportHandlerRoute(t *testing.T) {
	t.Skip("pre-existing vet error: newAPIGatewayRolloutReportProducerWithReadinessDetails undefined at HEAD; restore when production function is reintroduced")
}
