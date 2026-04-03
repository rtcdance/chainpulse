package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
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
	producer := newAPIGatewayRolloutReportProducerWithReadinessDetails(
		"api-gateway-1",
		func() apiGatewayRolloutRuntimeState {
			return apiGatewayRolloutRuntimeState{
				DomainBridgeEnabled:      true,
				EventQueryEnabled:        true,
				EventSubscriptionEnabled: true,
				HealthCheckEnabled:       true,
				RuntimeRoutesEnabled:     true,
			}
		},
		func() map[string]interface{} {
			return map[string]interface{}{
				"ownership_mode":                  "runtime-owned",
				"rollout_ready_for_runtime_owned": true,
				"rollout_status":                  "ready",
				"rollout_reason":                  "shared runtime owns observed writes",
			}
		},
	)

	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if err := api.ValidateRouteMonolithOwnershipParityReason(
		report,
		"monolith-runtime-owned-ready",
		"monolith ownership rollout is runtime-owned and ready; use this as the target parity posture",
		"target-ready",
		"use the monolith runtime-owned rollout as the current route parity target",
	); err != nil {
		t.Fatalf("expected monolith parity reason validation: %v", err)
	}
	if err := api.ValidateRouteMonolithOwnershipParityRecommendationBundle(report, api.MonolithOwnershipParityRecommendationBundle{
		Posture:        "monolith-runtime-owned-ready",
		Hint:           "monolith ownership rollout is runtime-owned and ready; use this as the target parity posture",
		TargetDecision: "target-ready",
		ActionGuidance: "use the monolith runtime-owned rollout as the current route parity target",
	}); err != nil {
		t.Fatalf("expected monolith parity bundle validation: %v", err)
	}
}

func TestClassifyAPIGatewayOwnershipParityHint(t *testing.T) {
	cases := []struct {
		name    string
		present bool
		want    string
	}{
		{"absent", false, "api-gateway ownership runtime parity with monolith is not yet wired"},
		{"present", true, "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending"},
	}

	for _, tc := range cases {
		if got := classifyAPIGatewayOwnershipParityHint(tc.present); got != tc.want {
			t.Fatalf("%s: expected ownership hint %q, got %q", tc.name, tc.want, got)
		}
	}
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
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.SetRolloutReportProducer(newAPIGatewayRolloutReportProducerWithReadinessDetails(
		"api-gateway-1",
		func() apiGatewayRolloutRuntimeState {
			return apiGatewayRolloutRuntimeState{
				EventQueryEnabled:        true,
				EventSubscriptionEnabled: true,
				HealthCheckEnabled:       true,
				RuntimeRoutesEnabled:     true,
			}
		},
		func() map[string]interface{} {
			return map[string]interface{}{
				"ownership_mode":                  "shadow",
				"rollout_ready_for_runtime_owned": false,
				"rollout_status":                  "shadow-observe",
				"rollout_reason":                  "shared runtime still coexists with legacy writes",
			}
		},
	))
	handler.InitializedForTests()

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()
	handler.HandleRollout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload api.RolloutReportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if !payload.Available || payload.Details == nil {
		t.Fatal("expected rollout details")
	}
	if got := payload.Details.Service; got != "api-gateway" {
		t.Fatalf("expected service api-gateway, got %q", got)
	}
	if got := payload.Details.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if err := api.ValidateMicroserviceOwnershipParityMarker(payload.Details); err != nil {
		t.Fatalf("expected ownership parity marker validation: %v", err)
	}
	if !strings.Contains(payload.Details.Advisory.Reason, "ownership_parity_hint: api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending") {
		t.Fatalf("expected ownership parity hint in reason, got %q", payload.Details.Advisory.Reason)
	}
	if err := api.ValidateRouteMonolithOwnershipParityReason(
		payload.Details,
		"monolith-shadow-observe",
		"monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet",
		"target-shadow",
		"keep route parity in observe mode until the monolith exits shadow posture",
	); err != nil {
		t.Fatalf("expected monolith parity reason validation: %v", err)
	}
	if err := api.ValidateRouteMonolithOwnershipParityRecommendationBundle(payload.Details, api.MonolithOwnershipParityRecommendationBundle{
		Posture:        "monolith-shadow-observe",
		Hint:           "monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet",
		TargetDecision: "target-shadow",
		ActionGuidance: "keep route parity in observe mode until the monolith exits shadow posture",
	}); err != nil {
		t.Fatalf("expected monolith parity bundle validation: %v", err)
	}
	if !strings.Contains(payload.Details.Advisory.Reason, "monolith_parity_target_decision: target-shadow") {
		t.Fatalf("expected monolith parity target decision in reason, got %q", payload.Details.Advisory.Reason)
	}
	if got := payload.Details.Approval.WorkItem.Reason; got != "api-gateway runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
}
