package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

func TestEventProcessorRolloutReportProducerSkeleton(t *testing.T) {
	producer := newEventProcessorRolloutReportProducer("event-processor-1", nil)
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.Service; got != "event-processor" {
		t.Fatalf("expected service event-processor, got %q", got)
	}
	if got := report.ReportSource; got != "microservice:event-processor-1" {
		t.Fatalf("expected report source microservice:event-processor-1, got %q", got)
	}
	if got := report.Mode; got != "unavailable" {
		t.Fatalf("expected mode unavailable, got %q", got)
	}
	if got := report.Progression.State; got != "unknown" {
		t.Fatalf("expected progression unknown, got %q", got)
	}
	if got := report.GuardedCutover.Overview.State; got != "investigate" {
		t.Fatalf("expected guarded overview investigate, got %q", got)
	}
}

func TestEventProcessorRolloutReportProducerRuntimeDerivedPartial(t *testing.T) {
	producer := newEventProcessorRolloutReportProducer("event-processor-1", func() eventProcessorRolloutRuntimeState {
		return eventProcessorRolloutRuntimeState{
			DatabaseReady:      true,
			EventStoreReady:    true,
			MetadataStoreReady: true,
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
	if !strings.Contains(report.Advisory.Reason, "enabled: database_ready,event_store_ready,metadata_store_ready") {
		t.Fatalf("expected enabled runtime signals in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "missing: kafka_ready") {
		t.Fatalf("expected missing runtime signals in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: finish wiring the missing event-processor runtime dependencies before treating this rollout as ready") {
		t.Fatalf("expected rollout posture hint in reason, got %q", report.Advisory.Reason)
	}
	if got := report.Progression.State; got != "observe" {
		t.Fatalf("expected progression observe, got %q", got)
	}
	if got := report.CutoverDryRun.Action; got != "would-hold" {
		t.Fatalf("expected cutover dry-run would-hold, got %q", got)
	}
}

func TestEventProcessorRolloutReportProducerRuntimeDerivedFullyWired(t *testing.T) {
	producer := newEventProcessorRolloutReportProducer("event-processor-1", func() eventProcessorRolloutRuntimeState {
		return eventProcessorRolloutRuntimeState{
			DatabaseReady:             true,
			EventStoreReady:           true,
			KafkaReady:                true,
			MetadataStoreReady:        true,
			EventStoreHealthStatus:    "healthy",
			MetadataStoreHealthStatus: "healthy",
			KafkaHealthStatus:         "healthy",
			KafkaMessageCount:         12,
			KafkaActivityState:        "active",
			ActiveConsumers:           2,
			ConsumerOffset:            144,
			ConsumerProgressState:     "active",
		}
	})
	report := producer.BuildRolloutReport(context.Background())

	if report == nil {
		t.Fatal("expected rollout report")
	}
	if err := api.ValidateMicroserviceRolloutMetadataParity(
		report,
		"event-processor",
		"event-processor-ownership-rollout-runtime",
		"microservice:event-processor-1",
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
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: observe the runtime-wired event-processor rollout while local processing dependencies remain healthy") {
		t.Fatalf("expected runtime-wired posture hint in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "kafka_activity_state: active") {
		t.Fatalf("expected kafka activity state in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "consumer_progress_posture: consumer-advancing") {
		t.Fatalf("expected consumer progress posture in reason, got %q", report.Advisory.Reason)
	}
	if err := api.ValidateRolloutExecutionProgressReasonCoverage(
		report.Advisory.Reason,
		api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Consumer: api.RolloutConsumerProgressSnapshot{
				ActiveConsumers: 2,
				CurrentOffset:   144,
				ProgressState:   "active",
			},
		}),
	); err != nil {
		t.Fatalf("expected execution progress reason coverage validation: %v", err)
	}
	if err := api.ValidateRolloutExecutionProgressPostureReasonCoverage(
		report.Advisory.Reason,
		api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Consumer: api.RolloutConsumerProgressSnapshot{
				ActiveConsumers: 2,
				CurrentOffset:   144,
				ProgressState:   "active",
			},
		}),
	); err != nil {
		t.Fatalf("expected execution progress posture reason coverage validation: %v", err)
	}
	if got := report.Approval.ManualApprovalCheckpoint.State; got != "inactive" {
		t.Fatalf("expected approval checkpoint inactive, got %q", got)
	}
	if got := report.GuardedCutover.Overview.State; got != "hold" {
		t.Fatalf("expected guarded overview hold, got %q", got)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(report); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
}

func TestEventProcessorRolloutReportProducerRuntimeDerivedDegraded(t *testing.T) {
	producer := newEventProcessorRolloutReportProducer("event-processor-1", func() eventProcessorRolloutRuntimeState {
		return eventProcessorRolloutRuntimeState{
			DatabaseReady:             true,
			EventStoreReady:           true,
			KafkaReady:                true,
			MetadataStoreReady:        true,
			EventStoreHealthStatus:    "degraded",
			EventStoreHealthMessage:   "mongo write latency elevated",
			MetadataStoreHealthStatus: "healthy",
			KafkaHealthStatus:         "healthy",
			KafkaActivityState:        "stalled",
			ActiveConsumers:           1,
			ConsumerLag:               15,
			ConsumerOffset:            144,
			ConsumerProgressState:     "lagging",
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
		t.Fatal("expected advisory ready false when a processing dependency is degraded")
	}
	if !strings.Contains(report.Advisory.Reason, "event_store_status: degraded") {
		t.Fatalf("expected event store degraded status in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "event_store_message: mongo write latency elevated") {
		t.Fatalf("expected event store degraded message in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "rollout_posture_hint: hold the runtime-wired event-processor rollout and investigate degraded processing dependency health before treating it as ready") {
		t.Fatalf("expected degraded posture hint in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "kafka_activity_state: stalled") {
		t.Fatalf("expected kafka activity state in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "consumer_lag: 15") {
		t.Fatalf("expected consumer lag in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "consumer_progress_posture: consumer-backlog") {
		t.Fatalf("expected consumer progress posture in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "consumer_lag_severity: backlog-low") {
		t.Fatalf("expected consumer lag severity in reason, got %q", report.Advisory.Reason)
	}
	if !strings.Contains(report.Advisory.Reason, "consumer_operator_hint: consumer backlog is present but small; continue observing drain progression") {
		t.Fatalf("expected consumer backlog hint in reason, got %q", report.Advisory.Reason)
	}
	if err := api.ValidateRolloutExecutionProgressReasonCoverage(
		report.Advisory.Reason,
		api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Consumer: api.RolloutConsumerProgressSnapshot{
				ActiveConsumers: 1,
				Lag:             15,
				CurrentOffset:   144,
				ProgressState:   "lagging",
			},
		}),
	); err != nil {
		t.Fatalf("expected execution progress reason coverage validation: %v", err)
	}
	if err := api.ValidateRolloutExecutionProgressPostureReasonCoverage(
		report.Advisory.Reason,
		api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Consumer: api.RolloutConsumerProgressSnapshot{
				ActiveConsumers: 1,
				Lag:             15,
				CurrentOffset:   144,
				ProgressState:   "lagging",
			},
		}),
	); err != nil {
		t.Fatalf("expected execution progress posture reason coverage validation: %v", err)
	}
}

func TestClassifyEventProcessorRolloutWiringCompleteness(t *testing.T) {
	completeness := classifyEventProcessorRolloutWiringCompleteness(eventProcessorRolloutRuntimeState{
		DatabaseReady:      true,
		EventStoreReady:    true,
		MetadataStoreReady: true,
	})

	if got := completeness.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if got := completeness.AdvisoryStatus; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
	if completeness.AdvisoryReady {
		t.Fatal("expected advisory ready false for partially wired state")
	}
	if len(completeness.EnabledSignals) != 3 {
		t.Fatalf("expected 3 enabled signals, got %d", len(completeness.EnabledSignals))
	}
	if len(completeness.MissingSignals) != 1 {
		t.Fatalf("expected 1 missing signal, got %d", len(completeness.MissingSignals))
	}
	if got := completeness.PostureHint; got != "finish wiring the missing event-processor runtime dependencies before treating this rollout as ready" {
		t.Fatalf("expected posture hint for partial wiring, got %q", got)
	}
}

func TestClassifyEventProcessorRolloutWiringCompletenessFullyWiredDegraded(t *testing.T) {
	completeness := classifyEventProcessorRolloutWiringCompleteness(eventProcessorRolloutRuntimeState{
		DatabaseReady:             true,
		EventStoreReady:           true,
		MetadataStoreReady:        true,
		KafkaReady:                true,
		EventStoreHealthStatus:    "degraded",
		EventStoreHealthMessage:   "mongo write latency elevated",
		MetadataStoreHealthStatus: "healthy",
		KafkaHealthStatus:         "healthy",
		KafkaActivityState:        "stalled",
		ActiveConsumers:           1,
		ConsumerLag:               15,
		ConsumerProgressState:     "lagging",
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
	if got := completeness.PostureHint; got != "hold the runtime-wired event-processor rollout and investigate degraded processing dependency health before treating it as ready" {
		t.Fatalf("expected degraded posture hint, got %q", got)
	}
	if !strings.Contains(completeness.Reason, "event_store_status: degraded") {
		t.Fatalf("expected degraded event store status in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "consumer_progress_posture: consumer-backlog") {
		t.Fatalf("expected consumer progress posture in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "consumer_lag_severity: backlog-low") {
		t.Fatalf("expected consumer lag severity in reason, got %q", completeness.Reason)
	}
	if !strings.Contains(completeness.Reason, "consumer_operator_hint: consumer backlog is present but small; continue observing drain progression") {
		t.Fatalf("expected consumer backlog hint in reason, got %q", completeness.Reason)
	}
}

func TestClassifyEventProcessorRolloutPostureHint(t *testing.T) {
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
			wantPosture: "finish wiring the missing event-processor runtime dependencies before treating this rollout as ready",
		},
		{
			name:        "runtime wired",
			mode:        "runtime-wired",
			advisory:    "runtime-wired",
			missing:     0,
			wantPosture: "observe the runtime-wired event-processor rollout while local processing dependencies remain healthy",
		},
		{
			name:        "runtime wired degraded",
			mode:        "runtime-wired",
			advisory:    "runtime-wired-degraded",
			missing:     0,
			wantPosture: "hold the runtime-wired event-processor rollout and investigate degraded processing dependency health before treating it as ready",
		},
		{
			name:        "runtime wired unhealthy",
			mode:        "runtime-wired",
			advisory:    "runtime-wired-unhealthy",
			missing:     0,
			wantPosture: "restore unhealthy event-processor processing dependencies before relying on this runtime-wired rollout",
		},
		{
			name:        "runtime wired health unknown",
			mode:        "runtime-wired",
			advisory:    "runtime-wired-health-unknown",
			missing:     0,
			wantPosture: "verify event-processor processing dependency health before treating this runtime-wired rollout as ready",
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
		if got := classifyEventProcessorRolloutPostureHint(tc.mode, tc.advisory, tc.missing); got != tc.wantPosture {
			t.Fatalf("%s: expected posture hint %q, got %q", tc.name, tc.wantPosture, got)
		}
	}
}

func TestEventProcessorRolloutReportHandlerRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.SetRolloutReportProducer(newEventProcessorRolloutReportProducer("event-processor-1", func() eventProcessorRolloutRuntimeState {
		return eventProcessorRolloutRuntimeState{
			DatabaseReady:   true,
			EventStoreReady: true,
			KafkaReady:      true,
		}
	}))
	handler.InitializedForTests()

	req := httptest.NewRequest("GET", "/health/rollout", nil)
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
		t.Fatal("expected rollout details to be available")
	}
	if got := payload.Details.Service; got != "event-processor" {
		t.Fatalf("expected service event-processor, got %q", got)
	}
	if got := payload.Details.Mode; got != "partially-wired" {
		t.Fatalf("expected mode partially-wired, got %q", got)
	}
	if got := payload.Details.Advisory.Status; got != "partial-runtime-wiring" {
		t.Fatalf("expected advisory status partial-runtime-wiring, got %q", got)
	}
	if !strings.Contains(payload.Details.Advisory.Reason, "rollout_posture_hint: finish wiring the missing event-processor runtime dependencies before treating this rollout as ready") {
		t.Fatalf("expected advisory reason to include rollout posture hint, got %q", payload.Details.Advisory.Reason)
	}
}
