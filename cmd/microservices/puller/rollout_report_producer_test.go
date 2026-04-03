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

func TestPullerRolloutReportProducerSkeleton(t *testing.T) {
	producer := newPullerRolloutReportProducer("puller-1", nil)
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Service; got != "puller" {
		t.Fatalf("expected service puller, got %q", got)
	}
	if got := report.ReportSource; got != "microservice:puller-1" {
		t.Fatalf("expected report source microservice:puller-1, got %q", got)
	}
	if got := report.Mode; got != "unavailable" {
		t.Fatalf("expected mode unavailable, got %q", got)
	}
	if got := report.Progression.State; got != "unknown" {
		t.Fatalf("expected progression unknown, got %q", got)
	}
}

func TestPullerRolloutReportProducerRuntimeDerivedPartial(t *testing.T) {
	producer := newPullerRolloutReportProducer("puller-1", func() pullerRolloutRuntimeState {
		return pullerRolloutRuntimeState{
			BlockchainRPCsConfigured: true,
			DatabaseReady:            true,
			PullerLoopConfigured:     true,
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
	if !strings.Contains(report.Advisory.Reason, "enabled: database_ready,puller_loop_configured,blockchain_rpcs_configured") {
		t.Fatalf("expected enabled signals in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "missing: kafka_ready") {
		t.Fatalf("expected missing signal in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: finish wiring the missing puller runtime dependencies before treating this rollout as ready") {
		t.Fatalf("expected posture hint in reason, got %q", report.Advisory.Reason)
	}
}

func TestPullerRolloutReportProducerRuntimeDerivedFullyWired(t *testing.T) {
	producer := newPullerRolloutReportProducer("puller-1", func() pullerRolloutRuntimeState {
		return pullerRolloutRuntimeState{
			BlockchainRPCsConfigured: true,
			DatabaseReady:            true,
			KafkaReady:               true,
			PullerLoopConfigured:     true,
			DatabaseHealthStatus:     "healthy",
			KafkaHealthStatus:        "healthy",
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if err := api.ValidateMicroserviceRolloutMetadataParity(
		report,
		"puller",
		"puller-ownership-rollout-runtime",
		"microservice:puller-1",
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
		t.Fatal("expected advisory ready true when all runtime dependencies are ready")
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: observe the runtime-wired puller rollout while local ingestion dependencies remain healthy") {
		t.Fatalf("expected runtime-wired posture hint in reason, got %q", report.Advisory.Reason)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(report); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
}

func TestPullerRolloutReportProducerRuntimeDerivedDegraded(t *testing.T) {
	producer := newPullerRolloutReportProducer("puller-1", func() pullerRolloutRuntimeState {
		return pullerRolloutRuntimeState{
			BlockchainRPCsConfigured: true,
			DatabaseReady:            true,
			KafkaReady:               true,
			PullerLoopConfigured:     true,
			DatabaseHealthStatus:     "degraded",
			DatabaseHealthMessage:    "postgres connections near saturation",
			KafkaHealthStatus:        "healthy",
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Advisory.Status; got != "runtime-wired-degraded" {
		t.Fatalf("expected advisory status runtime-wired-degraded, got %q", got)
	}
	if report.Advisory.Ready {
		t.Fatal("expected advisory ready false when an ingestion dependency is degraded")
	}
	if !strings.Contains(report.Advisory.Reason, "database_status: degraded") {
		t.Fatalf("expected degraded database status in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "database_message: postgres connections near saturation") {
		t.Fatalf("expected degraded database message in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: hold the runtime-wired puller rollout and investigate degraded ingestion dependency health before treating it as ready") {
		t.Fatalf("expected degraded posture hint in reason, got %q", report.Advisory.Reason)
	}
}

func TestClassifyPullerRolloutWiringCompleteness(t *testing.T) {
	completeness := classifyPullerRolloutWiringCompleteness(pullerRolloutRuntimeState{
		BlockchainRPCsConfigured: true,
		DatabaseReady:            true,
		PullerLoopConfigured:     true,
	})

	if got := completeness.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if got := completeness.AdvisoryStatus; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
	if completeness.AdvisoryReady {
		t.Fatal("expected advisory ready false")
	}
	if got := completeness.PostureHint; got != "finish wiring the missing puller runtime dependencies before treating this rollout as ready" {
		t.Fatalf("expected posture hint, got %q", got)
	}
}

func TestClassifyPullerRolloutWiringCompletenessFullyWiredDegraded(t *testing.T) {
	completeness := classifyPullerRolloutWiringCompleteness(pullerRolloutRuntimeState{
		BlockchainRPCsConfigured: true,
		DatabaseReady:            true,
		KafkaReady:               true,
		PullerLoopConfigured:     true,
		DatabaseHealthStatus:     "degraded",
		DatabaseHealthMessage:    "postgres connections near saturation",
		KafkaHealthStatus:        "healthy",
	})

	if got := completeness.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := completeness.AdvisoryStatus; got != "runtime-wired-degraded" {
		t.Fatalf("expected advisory status runtime-wired-degraded, got %q", got)
	}
	if completeness.AdvisoryReady {
		t.Fatal("expected advisory ready false for degraded runtime health")
	}
	if got := completeness.PostureHint; got != "hold the runtime-wired puller rollout and investigate degraded ingestion dependency health before treating it as ready" {
		t.Fatalf("expected degraded posture hint, got %q", got)
	}
	if !strings.Contains(completeness.Reason, "database_status: degraded") {
		t.Fatalf("expected degraded database status in reason, got %q", completeness.Reason)
	}
}

func TestClassifyPullerRolloutWiringCompletenessIncludesCheckpointCoveragePosture(t *testing.T) {
	completeness := classifyPullerRolloutWiringCompleteness(pullerRolloutRuntimeState{
		BlockchainRPCsConfigured:  true,
		DatabaseReady:             true,
		KafkaReady:                true,
		PullerLoopConfigured:      true,
		DatabaseHealthStatus:      "healthy",
		KafkaHealthStatus:         "healthy",
		CheckpointCoverageHint:    "tracked=2,recorded=1,reorg_risk=0,reorg_reconciled=1",
		CheckpointCoveragePosture: "coverage-reconciled",
		CheckpointRecoveryHint:    "checkpoint recovery has reconciled recent risk; continue observing fresh checkpoint coverage",
	})

	if !strings.Contains(completeness.Reason, "checkpoint_coverage: tracked=2,recorded=1,reorg_risk=0,reorg_reconciled=1") {
		t.Fatalf("expected checkpoint coverage hint in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "checkpoint_coverage_posture: coverage-reconciled") {
		t.Fatalf("expected checkpoint coverage posture in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "poll_operator_hint: checkpoint recovery has reconciled recent risk; continue observing fresh checkpoint coverage") {
		t.Fatalf("expected checkpoint recovery hint in reason, got %q", completeness.Reason)
	}
}

func TestClassifyPullerRolloutPostureHint(t *testing.T) {
	cases := []struct {
		name string
		mode string
		stat string
		miss int
		want string
	}{
		{"partial", "partially-wired", "partial-runtime-wiring", 1, "finish wiring the missing puller runtime dependencies before treating this rollout as ready"},
		{"wired", "runtime-wired", "runtime-wired", 0, "observe the runtime-wired puller rollout while local ingestion dependencies remain healthy"},
		{"wired degraded", "runtime-wired", "runtime-wired-degraded", 0, "hold the runtime-wired puller rollout and investigate degraded ingestion dependency health before treating it as ready"},
		{"wired unhealthy", "runtime-wired", "runtime-wired-unhealthy", 0, "restore unhealthy puller ingestion dependencies before relying on this runtime-wired rollout"},
		{"wired unknown", "runtime-wired", "runtime-wired-health-unknown", 0, "verify puller ingestion dependency health before treating this runtime-wired rollout as ready"},
		{"fallback", "unavailable", "unavailable", 0, ""},
	}

	for _, tc := range cases {
		if got := classifyPullerRolloutPostureHint(tc.mode, tc.stat, tc.miss); got != tc.want {
			t.Fatalf("%s: expected posture hint %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestPullerRolloutReportHandlerRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.SetRolloutReportProducer(newPullerRolloutReportProducer("puller-1", func() pullerRolloutRuntimeState {
		return pullerRolloutRuntimeState{
			BlockchainRPCsConfigured: true,
			DatabaseReady:            true,
			PullerLoopConfigured:     true,
		}
	}))
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
	if got := payload.Details.Service; got != "puller" {
		t.Fatalf("expected service puller, got %q", got)
	}
	if got := payload.Details.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if got := payload.Details.Advisory.Status; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
}
