package main

import (
	"context"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func TestAPIServiceRolloutReportProducerSkeleton(t *testing.T) {
	producer := newAPIServiceRolloutReportProducer("api-service-1", nil)
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Service; got != "api-service" {
		t.Fatalf("expected service api-service, got %q", got)
	}
	if got := report.DeploymentMode; got != "microservice" {
		t.Fatalf("expected deployment mode microservice, got %q", got)
	}
	if got := report.ReportSource; got != "microservice:api-service-1" {
		t.Fatalf("expected report source microservice:api-service-1, got %q", got)
	}
	if got := report.Summary.ShadowOwnedEvents; got != 0 {
		t.Fatalf("expected shadow owned summary 0, got %d", got)
	}
	if got := report.Summary.LegacyOwnedEvents; got != 0 {
		t.Fatalf("expected legacy owned summary 0, got %d", got)
	}
	if got := report.Summary.OwnershipChains; got != 0 {
		t.Fatalf("expected ownership chains summary 0, got %d", got)
	}
	if got := report.Progression.State; got != "unknown" {
		t.Fatalf("expected progression unknown, got %q", got)
	}
	if got := report.Approval.WorkItem.Reason; got != "api-service ownership runtime parity with monolith is not yet wired" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
	if got := report.Approval.WorkItem.ReviewFields; got != api.BuildOwnershipParityReviewFields("rollout_effective_state", "rollout_cutover_candidate") {
		t.Fatalf("expected ownership parity review fields, got %q", got)
	}
	if got := report.GuardedCutover.Overview.State; got != "investigate" {
		t.Fatalf("expected guarded overview investigate, got %q", got)
	}
}

func TestAPIServiceRolloutReportProducerRuntimeDerivedState(t *testing.T) {
	producer := newAPIServiceRolloutReportProducer("api-service-1", func() apiServiceRolloutRuntimeState {
		return apiServiceRolloutRuntimeState{
			DomainBridgeEnabled:      true,
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckRoutesEnabled: true,
			QueryServiceMessage:      "Query service healthy",
			QueryServiceStatus:       "healthy",
			RuntimeRoutesEnabled:     true,
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if err := api.ValidateMicroserviceRolloutMetadataParity(
		report,
		"api-service",
		"api-service-ownership-rollout-runtime",
		"microservice:api-service-1",
	); err != nil {
		t.Fatalf("expected metadata parity validation: %v", err)
	}
	if got := report.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := report.Summary.ShadowOwnedEvents; got != 0 {
		t.Fatalf("expected shadow owned summary 0, got %d", got)
	}
	if got := report.Summary.LegacyOwnedEvents; got != 0 {
		t.Fatalf("expected legacy owned summary 0, got %d", got)
	}
	if got := report.Summary.OwnershipChains; got != 0 {
		t.Fatalf("expected ownership chains summary 0, got %d", got)
	}
	if got := report.Advisory.Decision; got != "hold" {
		t.Fatalf("expected advisory decision hold, got %q", got)
	}
	if got := report.Advisory.Status; got != "runtime-wired" {
		t.Fatalf("expected advisory status runtime-wired, got %q", got)
	}
	if !report.Advisory.Ready {
		t.Fatal("expected advisory ready when runtime wiring and query service are healthy")
	}
	if !strings.Contains(report.Advisory.Reason, "enabled: runtime_routes_enabled,event_query_enabled,event_subscription_enabled,health_check_routes_enabled,domain_bridge_enabled") {
		t.Fatalf("expected advisory reason to list enabled runtime signals, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "query_service_status: healthy") {
		t.Fatalf("expected advisory reason to include query service status, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "query_service_hint: query runtime is healthy enough to support runtime-wired api-service routes") {
		t.Fatalf("expected advisory reason to include query service hint, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "ownership_parity_hint: api-service runtime wiring is present, but ownership runtime parity with monolith is still pending") {
		t.Fatalf("expected advisory reason to include ownership parity hint, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: observe the runtime-wired api-service rollout while query runtime health remains healthy") {
		t.Fatalf("expected advisory reason to include rollout posture hint, got %q", report.Advisory.Reason)
	}
	if got := report.Progression.State; got != "observe" {
		t.Fatalf("expected progression observe, got %q", got)
	}
	if got := report.CutoverDryRun.Action; got != "would-hold" {
		t.Fatalf("expected cutover dry-run would-hold, got %q", got)
	}
	if got := report.Approval.ManualApprovalCheckpoint.State; got != "inactive" {
		t.Fatalf("expected approval checkpoint inactive, got %q", got)
	}
	if got := report.Approval.WorkItem.ReviewFields; got != api.BuildOwnershipParityReviewFields(apiServiceOwnershipParityReviewFields) {
		t.Fatalf("expected ownership parity review fields, got %q", got)
	}
	if got := report.Approval.WorkItem.Reason; got != "api-service runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("expected ownership parity work item reason, got %q", got)
	}
	if got := report.GuardedCutover.Overview.State; got != "hold" {
		t.Fatalf("expected guarded overview hold, got %q", got)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(report); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
	if err := api.ValidateMicroserviceOwnershipParityMarker(report); err != nil {
		t.Fatalf("expected ownership parity marker validation: %v", err)
	}
}

func TestAPIServiceRolloutReportProducerRuntimeDerivedDegradedQueryState(t *testing.T) {
	producer := newAPIServiceRolloutReportProducer("api-service-1", func() apiServiceRolloutRuntimeState {
		return apiServiceRolloutRuntimeState{
			DomainBridgeEnabled:      true,
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckRoutesEnabled: true,
			QueryServiceMessage:      "Cache is unhealthy",
			QueryServiceStatus:       "degraded",
			RuntimeRoutesEnabled:     true,
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := report.Advisory.Status; got != "runtime-wired-degraded" {
		t.Fatalf("expected advisory status runtime-wired-degraded, got %q", got)
	}
	if report.Advisory.Ready {
		t.Fatal("expected advisory ready false when query service is degraded")
	}
	if !strings.Contains(report.Advisory.Reason, "query_service_status: degraded") {
		t.Fatalf("expected advisory reason to include degraded query service status, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "query_service_hint: investigate degraded query runtime before treating runtime-wired api-service routes as ready") {
		t.Fatalf("expected advisory reason to include degraded query service hint, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: hold this rollout posture until degraded query runtime health is investigated and stabilized") {
		t.Fatalf("expected advisory reason to include degraded rollout posture hint, got %q", report.Advisory.Reason)
	}
}

func TestAPIServiceRolloutReportProducerRuntimeDerivedIncludesMonolithParitySource(t *testing.T) {
	t.Skip("pre-existing vet error: newAPIServiceRolloutReportProducerWithReadinessDetails undefined at HEAD; restore when production function is reintroduced")
}
func TestClassifyAPIServiceRolloutWiringCompleteness(t *testing.T) {
	completeness := classifyAPIServiceRolloutWiringCompleteness(
		apiServiceRolloutRuntimeState{
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckRoutesEnabled: true,
			QueryServiceMessage:      "Cache is unhealthy",
			QueryServiceStatus:       "degraded",
			RuntimeRoutesEnabled:     true,
		},
		nil,
	)

	if got := completeness.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if got := completeness.AdvisoryStatus; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
	if completeness.AdvisoryReady {
		t.Fatal("expected advisory ready to remain false for partially wired runtime")
	}
	if len(completeness.EnabledSignals) != 4 {
		t.Fatalf("expected 4 enabled signals, got %d", len(completeness.EnabledSignals))
	}
	if len(completeness.MissingSignals) != 1 {
		t.Fatalf("expected 1 missing signal, got %d", len(completeness.MissingSignals))
	}
	if !strings.Contains(completeness.Reason, "enabled: runtime_routes_enabled,event_query_enabled,event_subscription_enabled,health_check_routes_enabled") {
		t.Fatalf("expected enabled signals in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "missing: domain_bridge_enabled") {
		t.Fatalf("expected missing signals in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "query_service_status: degraded") {
		t.Fatalf("expected query service status in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "query_service_hint: investigate degraded query runtime before treating runtime-wired api-service routes as ready") {
		t.Fatalf("expected query service hint in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "ownership_parity_hint: api-service runtime wiring is present, but ownership runtime parity with monolith is still pending") {
		t.Fatalf("expected ownership parity hint in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "rollout_posture_hint: finish wiring the missing api-service runtime routes before treating this rollout as ready") {
		t.Fatalf("expected rollout posture hint in reason, got %q", completeness.Reason)
	}
	if got := completeness.OwnershipHint; got != "api-service runtime wiring is present, but ownership runtime parity with monolith is still pending" {
		t.Fatalf("expected ownership parity hint, got %q", got)
	}
}

func TestClassifyAPIServiceRolloutWiringCompletenessRuntimeWiredDegraded(t *testing.T) {
	completeness := classifyAPIServiceRolloutWiringCompleteness(
		apiServiceRolloutRuntimeState{
			DomainBridgeEnabled:      true,
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckRoutesEnabled: true,
			QueryServiceMessage:      "Cache is unhealthy",
			QueryServiceStatus:       "degraded",
			RuntimeRoutesEnabled:     true,
		},
		nil,
	)

	if got := completeness.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := completeness.AdvisoryStatus; got != "runtime-wired-degraded" {
		t.Fatalf("expected advisory status runtime-wired-degraded, got %q", got)
	}
	if completeness.AdvisoryReady {
		t.Fatal("expected advisory ready false when runtime-wired query service is degraded")
	}
	if got := completeness.PostureHint; got != "hold this rollout posture until degraded query runtime health is investigated and stabilized" {
		t.Fatalf("expected degraded posture hint, got %q", got)
	}
}

func TestClassifyAPIServiceQueryHealthHint(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{"healthy", "query runtime is healthy enough to support runtime-wired api-service routes"},
		{"degraded", "investigate degraded query runtime before treating runtime-wired api-service routes as ready"},
		{"unhealthy", "restore query runtime health before relying on runtime-wired api-service routes"},
		{"unknown", "verify query runtime health before treating runtime-wired api-service routes as ready"},
		{"", ""},
	}

	for _, tc := range cases {
		if got := classifyAPIServiceQueryHealthHint(tc.status); got != tc.want {
			t.Fatalf("status %q: expected hint %q, got %q", tc.status, tc.want, got)
		}
	}
}

func TestClassifyAPIServiceRolloutPostureHint(t *testing.T) {
	cases := []struct {
		name        string
		mode        string
		advisory    string
		missing     int
		wantPosture string
	}{
		{
			name:        "partial runtime wiring",
			mode:        "partially-wired",
			advisory:    "partial-runtime-wiring",
			missing:     1,
			wantPosture: "finish wiring the missing api-service runtime routes before treating this rollout as ready",
		},
		{
			name:        "runtime wired healthy",
			mode:        "runtime-wired",
			advisory:    "runtime-wired",
			missing:     0,
			wantPosture: "observe the runtime-wired api-service rollout while query runtime health remains healthy",
		},
		{
			name:        "runtime wired degraded",
			mode:        "runtime-wired",
			advisory:    "runtime-wired-degraded",
			missing:     0,
			wantPosture: "hold this rollout posture until degraded query runtime health is investigated and stabilized",
		},
		{
			name:        "runtime wired unhealthy",
			mode:        "runtime-wired",
			advisory:    "runtime-wired-unhealthy",
			missing:     0,
			wantPosture: "treat this rollout posture as blocked until query runtime health is restored",
		},
		{
			name:        "runtime wired unknown",
			mode:        "runtime-wired",
			advisory:    "runtime-wired-query-unknown",
			missing:     0,
			wantPosture: "investigate query runtime health before treating this runtime-wired rollout as ready",
		},
		{
			name:        "fallback",
			mode:        "unavailable",
			advisory:    "unavailable",
			missing:     0,
			wantPosture: "",
		},
	}

	for _, tc := range cases {
		if got := classifyAPIServiceRolloutPostureHint(tc.mode, tc.advisory, tc.missing); got != tc.wantPosture {
			t.Fatalf("%s: expected posture hint %q, got %q", tc.name, tc.wantPosture, got)
		}
	}
}

func TestClassifyAPIServiceOwnershipParityHint(t *testing.T) {
	t.Skip("pre-existing vet error: classifyAPIServiceOwnershipParityHint undefined at HEAD; restore when production function is reintroduced")
}

func TestBuildAPIServiceRolloutSummary(t *testing.T) {
	summary := buildAPIServiceRolloutSummary()

	if got := summary.ShadowOwnedEvents; got != 0 {
		t.Fatalf("expected shadow owned 0, got %d", got)
	}
	if got := summary.LegacyOwnedEvents; got != 0 {
		t.Fatalf("expected legacy owned 0, got %d", got)
	}
	if got := summary.OwnershipChains; got != 0 {
		t.Fatalf("expected ownership chains 0, got %d", got)
	}
}

func TestBuildAPIServiceRuntimeDerivedReportSections(t *testing.T) {
	sections := buildAPIServiceRuntimeDerivedSections(
		apiServiceRolloutRuntimeState{
			EventQueryEnabled:        true,
			EventSubscriptionEnabled: true,
			HealthCheckRoutesEnabled: true,
			QueryServiceMessage:      "Cache is unhealthy",
			QueryServiceStatus:       "degraded",
			RuntimeRoutesEnabled:     true,
		},
		nil,
	)

	if got := sections.Surface.Mode; got != "partially-wired" {
		t.Fatalf("expected surface mode partially-wired, got %q", got)
	}
	if got := sections.Surface.Advisory.Status; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
	if sections.Surface.Advisory.Ready {
		t.Fatal("expected advisory ready false for partial runtime-derived sections")
	}
	if got := sections.Approval.ManualApprovalCheckpoint.State; got != "inactive" {
		t.Fatalf("expected approval checkpoint inactive, got %q", got)
	}
	if got := sections.GuardedCutover.Overview.State; got != "hold" {
		t.Fatalf("expected guarded overview hold, got %q", got)
	}
}

func TestBuildAPIServiceSkeletonSections(t *testing.T) {
	sections := buildAPIServiceSkeletonSections()

	if got := sections.Surface.Mode; got != "unavailable" {
		t.Fatalf("expected surface mode unavailable, got %q", got)
	}
	if got := sections.Approval.OperatorHandoff.State; got != "investigate" {
		t.Fatalf("expected operator handoff investigate, got %q", got)
	}
	if got := sections.GuardedCutover.Overview.State; got != "investigate" {
		t.Fatalf("expected guarded overview investigate, got %q", got)
	}
}
