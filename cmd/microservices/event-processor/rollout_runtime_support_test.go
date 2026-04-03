package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
	"go.mongodb.org/mongo-driver/mongo"
)

type eventProcessorTestDatabaseManager struct {
	mongoHealthy    bool
	postgresHealthy bool
}

var _ database.DatabaseManager = (*eventProcessorTestDatabaseManager)(nil)

func (m *eventProcessorTestDatabaseManager) Initialize(ctx context.Context) error { return nil }
func (m *eventProcessorTestDatabaseManager) GetMongoClient(ctx context.Context) (interface{}, error) {
	return nil, nil
}
func (m *eventProcessorTestDatabaseManager) GetMongoDatabase(name string) *mongo.Database { return nil }
func (m *eventProcessorTestDatabaseManager) GetPostgresDB(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (m *eventProcessorTestDatabaseManager) CheckMongoHealth(ctx context.Context) error {
	if m.mongoHealthy {
		return nil
	}
	return context.DeadlineExceeded
}

func (m *eventProcessorTestDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	if m.postgresHealthy {
		return nil
	}
	return context.DeadlineExceeded
}

func (m *eventProcessorTestDatabaseManager) Health(ctx context.Context) interface{} {
	return map[string]interface{}{
		"status":   "healthy",
		"mongodb":  m.mongoHealthy,
		"postgres": m.postgresHealthy,
	}
}
func (m *eventProcessorTestDatabaseManager) Close(ctx context.Context) error { return nil }

type eventProcessorTestComponentHealth struct {
	status  string
	message string
}

func (c *eventProcessorTestComponentHealth) Health(ctx context.Context) *core.HealthStatus {
	return &core.HealthStatus{
		Status:  c.status,
		Message: c.message,
	}
}

type eventProcessorTestKafkaHealth struct {
	status             string
	message            string
	details            map[string]interface{}
	consumerGroupState map[string]interface{}
	consumerMetrics    map[string]int64
}

func (k *eventProcessorTestKafkaHealth) Health() *core.HealthStatus {
	return &core.HealthStatus{
		Status:  k.status,
		Message: k.message,
		Details: k.details,
	}
}

func (k *eventProcessorTestKafkaHealth) GetConsumerGroupStatus() map[string]interface{} {
	return k.consumerGroupState
}

func (k *eventProcessorTestKafkaHealth) GetConsumerGroupMetrics() map[string]int64 {
	return k.consumerMetrics
}

type eventProcessorTestProcessorRuntime struct {
	status         string
	message        string
	processedCount int64
	failedCount    int64
	duplicateCount int64
	sharedShadow   eventProcessorSharedRuntimeShadowSnapshot
}

func (p *eventProcessorTestProcessorRuntime) Health() *core.HealthStatus {
	return &core.HealthStatus{
		Status:  p.status,
		Message: p.message,
	}
}

func (p *eventProcessorTestProcessorRuntime) GetProcessedCount() int64 {
	return p.processedCount
}

func (p *eventProcessorTestProcessorRuntime) GetFailedCount() int64 {
	return p.failedCount
}

func (p *eventProcessorTestProcessorRuntime) GetDuplicateCount() int64 {
	return p.duplicateCount
}

func (p *eventProcessorTestProcessorRuntime) SharedRuntimeShadowSnapshot() eventProcessorSharedRuntimeShadowSnapshot {
	return p.sharedShadow
}

type eventProcessorTestConsumeRuntime struct {
	configuredTopics int
	activeTopics     int
	running          bool
	lastError        string
}

func (c *eventProcessorTestConsumeRuntime) Snapshot() eventProcessorConsumeLoopSnapshot {
	return eventProcessorConsumeLoopSnapshot{
		ConfiguredTopics: c.configuredTopics,
		ActiveTopics:     c.activeTopics,
		Running:          c.running,
		LastError:        c.lastError,
	}
}

func TestBuildEventProcessorRuntimeRolloutState(t *testing.T) {
	state := buildEventProcessorRuntimeRolloutState(
		context.Background(),
		&eventProcessorTestDatabaseManager{mongoHealthy: true, postgresHealthy: true},
		&eventProcessorTestComponentHealth{status: "healthy", message: "event store ready"},
		&eventProcessorTestComponentHealth{status: "healthy", message: "metadata store ready"},
		&eventProcessorTestKafkaHealth{
			status:  "healthy",
			message: "kafka ready",
			details: map[string]interface{}{
				"message_count":      int64(12),
				"error_count":        int64(1),
				"max_tracked_offset": int64(144),
			},
			consumerGroupState: map[string]interface{}{
				"active_consumers": int64(2),
			},
			consumerMetrics: map[string]int64{
				"lag": 0,
			},
		},
		&eventProcessorTestProcessorRuntime{
			status:         "healthy",
			message:        "processor runtime healthy",
			processedCount: 8,
			failedCount:    1,
			duplicateCount: 2,
			sharedShadow: eventProcessorSharedRuntimeShadowSnapshot{
				Enabled:              true,
				RuntimeCount:         2,
				ProcessedEvents:      8,
				SkippedDuplicates:    2,
				RoutedFailures:       1,
				LastCheckpointChain:  "polygon",
				LastCheckpointCursor: "22:0",
				LastCheckpointBlock:  22,
			},
		},
		&eventProcessorTestConsumeRuntime{
			configuredTopics: 2,
			activeTopics:     1,
			running:          true,
		},
	)

	if !state.DatabaseReady {
		t.Fatal("expected database ready")
	}
	if !state.EventStoreReady {
		t.Fatal("expected event store ready")
	}
	if !state.MetadataStoreReady {
		t.Fatal("expected metadata store ready")
	}
	if !state.KafkaReady {
		t.Fatal("expected kafka ready")
	}
	if !state.ProcessorRuntimeReady {
		t.Fatal("expected processor runtime ready")
	}
	if !state.ConsumeLoopOwned {
		t.Fatal("expected consume loop owned")
	}
	if got := state.EventStoreHealthStatus; got != "healthy" {
		t.Fatalf("expected healthy event store status, got %q", got)
	}
	if got := state.MetadataStoreHealthStatus; got != "healthy" {
		t.Fatalf("expected healthy metadata store status, got %q", got)
	}
	if got := state.KafkaHealthStatus; got != "healthy" {
		t.Fatalf("expected healthy kafka status, got %q", got)
	}
	if got := state.KafkaMessageCount; got != 12 {
		t.Fatalf("expected kafka message count 12, got %d", got)
	}
	if got := state.KafkaErrorCount; got != 1 {
		t.Fatalf("expected kafka error count 1, got %d", got)
	}
	if got := state.KafkaActivityState; got != "active" {
		t.Fatalf("expected kafka activity state active, got %q", got)
	}
	if got := state.ActiveConsumers; got != 2 {
		t.Fatalf("expected active consumers 2, got %d", got)
	}
	if got := state.ConsumerLag; got != 0 {
		t.Fatalf("expected consumer lag 0, got %d", got)
	}
	if got := state.ConsumerLagSeverity; got != "" {
		t.Fatalf("expected empty consumer lag severity, got %q", got)
	}
	if got := state.ConsumerOffset; got != 144 {
		t.Fatalf("expected consumer offset 144, got %d", got)
	}
	if got := state.ConsumerProgressState; got != "active" {
		t.Fatalf("expected consumer progress state active, got %q", got)
	}
	if got := state.ConsumerProgressPosture; got != "consumer-advancing" {
		t.Fatalf("expected consumer progress posture consumer-advancing, got %q", got)
	}
	if !state.SharedRuntimeShadowEnabled {
		t.Fatal("expected shared runtime shadow enabled")
	}
	if got := state.SharedRuntimeChainCount; got != 2 {
		t.Fatalf("expected shared runtime chain count 2, got %d", got)
	}
	if got := state.SharedRuntimeLastChain; got != "polygon" {
		t.Fatalf("expected shared runtime last checkpoint chain polygon, got %q", got)
	}
	if got := state.ConsumerBacklogHint; got != "consumer progress is advancing; continue observing steady backlog drain" {
		t.Fatalf("expected consumer backlog hint for advancing progress, got %q", got)
	}
	if got := state.ProcessorHealthStatus; got != "healthy" {
		t.Fatalf("expected healthy processor status, got %q", got)
	}
	if got := state.ProcessedEventCount; got != 8 {
		t.Fatalf("expected processed count 8, got %d", got)
	}
	if got := state.FailedEventCount; got != 1 {
		t.Fatalf("expected failed count 1, got %d", got)
	}
	if got := state.DuplicateEventCount; got != 2 {
		t.Fatalf("expected duplicate count 2, got %d", got)
	}
	if got := state.ConsumeLoopStatus; got != "active" {
		t.Fatalf("expected consume loop status active, got %q", got)
	}
	if got := state.ConfiguredConsumeTopics; got != 2 {
		t.Fatalf("expected configured consume topics 2, got %d", got)
	}
	if got := state.ActiveConsumeTopics; got != 1 {
		t.Fatalf("expected active consume topics 1, got %d", got)
	}
}

func TestBuildEventProcessorRuntimeComponentStatus(t *testing.T) {
	component := buildEventProcessorRuntimeComponentStatus(eventProcessorRolloutRuntimeState{
		DatabaseReady:             true,
		EventStoreReady:           true,
		MetadataStoreReady:        true,
		KafkaReady:                true,
		EventStoreHealthStatus:    "healthy",
		MetadataStoreHealthStatus: "healthy",
		KafkaHealthStatus:         "healthy",
		KafkaActivityState:        "active",
		ConsumerProgressPosture:   "consumer-advancing",
		ProcessorRuntimeReady:     true,
		ProcessorHealthStatus:     "healthy",
		ConsumeLoopOwned:          true,
		ConsumeLoopStatus:         "active",
	}, time.Unix(1712345678, 0))

	if component == nil {
		t.Fatal("expected component status")
	}
	if got := component.Status; got != "healthy" {
		t.Fatalf("expected healthy component status, got %q", got)
	}
	if got := component.Details["runtime_mode"]; got != "runtime-wired" {
		t.Fatalf("expected runtime mode runtime-wired, got %v", got)
	}
	if got := component.Details["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout gate decision allow, got %v", got)
	}
}

func TestBuildEventProcessorRuntimeReadinessDetails(t *testing.T) {
	details := buildEventProcessorRuntimeReadinessDetails(eventProcessorRolloutRuntimeState{
		DatabaseReady:             true,
		EventStoreReady:           true,
		MetadataStoreReady:        true,
		KafkaReady:                true,
		EventStoreHealthStatus:    "healthy",
		MetadataStoreHealthStatus: "healthy",
		KafkaHealthStatus:         "healthy",
		KafkaActivityState:        "active",
		ConsumerLag:               15,
		ConsumerLagSeverity:       "backlog-medium",
		ConsumerProgressPosture:   "consumer-backlog",
		ConsumerBacklogHint:       "consumer backlog is building; monitor drain rate and investigate if it persists",
		ProcessorRuntimeReady:     true,
		ProcessorHealthStatus:     "healthy",
		ProcessorHealthMessage:    "processor runtime healthy",
		ProcessedEventCount:       9,
		FailedEventCount:          2,
		DuplicateEventCount:       1,
		ConsumeLoopOwned:          true,
		ConsumeLoopStatus:         "active",
		ConfiguredConsumeTopics:   2,
		ActiveConsumeTopics:       1,
	})

	if got := details["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout gate decision allow, got %v", got)
	}
	if got := details["consumer_lag"]; got != int64(15) {
		t.Fatalf("expected consumer lag 15, got %v", got)
	}
	if got := details["consumer_backlog_hint"]; got != "consumer backlog is building; monitor drain rate and investigate if it persists" {
		t.Fatalf("unexpected consumer backlog hint %v", got)
	}
	if got := details["processor_health_status"]; got != "healthy" {
		t.Fatalf("expected healthy processor status, got %v", got)
	}
	if got := details["processor_processed_count"]; got != int64(9) {
		t.Fatalf("expected processor processed count 9, got %v", got)
	}
	if got := details["consume_loop_status"]; got != "active" {
		t.Fatalf("expected consume loop status active, got %v", got)
	}
	if got := details["configured_consume_topics"]; got != 2 {
		t.Fatalf("expected configured consume topics 2, got %v", got)
	}
}

func TestClassifyEventProcessorKafkaActivityState(t *testing.T) {
	cases := []struct {
		name         string
		messageCount int64
		errorCount   int64
		want         string
	}{
		{
			name:         "active with messages",
			messageCount: 12,
			errorCount:   0,
			want:         "active",
		},
		{
			name:         "active with errors",
			messageCount: 0,
			errorCount:   1,
			want:         "active",
		},
		{
			name:         "stalled",
			messageCount: 0,
			errorCount:   0,
			want:         "stalled",
		},
	}

	for _, tc := range cases {
		got := classifyEventProcessorKafkaActivityState(tc.messageCount, tc.errorCount)
		if got != tc.want {
			t.Fatalf("%s: expected kafka activity state %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestClassifyEventProcessorConsumerProgressState(t *testing.T) {
	cases := []struct {
		name            string
		activeConsumers int64
		lag             int64
		activityState   string
		want            string
	}{
		{
			name:            "idle without consumers",
			activeConsumers: 0,
			lag:             0,
			activityState:   "stalled",
			want:            "idle",
		},
		{
			name:            "lagging with backlog",
			activeConsumers: 2,
			lag:             15,
			activityState:   "active",
			want:            "lagging",
		},
		{
			name:            "active with consumers and activity",
			activeConsumers: 2,
			lag:             0,
			activityState:   "active",
			want:            "active",
		},
		{
			name:            "monitoring",
			activeConsumers: 2,
			lag:             0,
			activityState:   "stalled",
			want:            "monitoring",
		},
	}

	for _, tc := range cases {
		got := classifyEventProcessorConsumerProgressState(tc.activeConsumers, tc.lag, tc.activityState)
		if got != tc.want {
			t.Fatalf("%s: expected consumer progress state %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestBuildEventProcessorConsumerProgressPosture(t *testing.T) {
	cases := []struct {
		name     string
		progress string
		lag      int64
		offset   int64
		want     string
	}{
		{name: "idle", progress: "idle", want: "consumer-idle"},
		{name: "lagging", progress: "lagging", lag: 15, offset: 120, want: "consumer-backlog"},
		{name: "active with offset", progress: "active", offset: 144, want: "consumer-advancing"},
		{name: "active without offset", progress: "active", want: "consumer-active"},
		{name: "monitoring", progress: "monitoring", want: "consumer-monitoring"},
	}

	for _, tc := range cases {
		got := api.BuildRolloutExecutionProgressPosture(api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Consumer: api.RolloutConsumerProgressSnapshot{
				ActiveConsumers: 1,
				Lag:             tc.lag,
				CurrentOffset:   tc.offset,
				ProgressState:   tc.progress,
			},
		})).Consumer
		if got != tc.want {
			t.Fatalf("%s: expected consumer progress posture %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestClassifyEventProcessorConsumerLagSeverity(t *testing.T) {
	cases := []struct {
		name string
		lag  int64
		want string
	}{
		{name: "none", lag: 0, want: ""},
		{name: "low", lag: 15, want: "backlog-low"},
		{name: "medium", lag: 20, want: "backlog-medium"},
		{name: "high", lag: 100, want: "backlog-high"},
	}

	for _, tc := range cases {
		if got := classifyEventProcessorConsumerLagSeverity(tc.lag); got != tc.want {
			t.Fatalf("%s: expected lag severity %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestClassifyEventProcessorConsumerBacklogHint(t *testing.T) {
	cases := []struct {
		name     string
		posture  string
		severity string
		want     string
	}{
		{name: "high backlog", posture: "consumer-backlog", severity: "backlog-high", want: "consumer backlog is high; prioritize drain and investigate processor throughput"},
		{name: "medium backlog", posture: "consumer-backlog", severity: "backlog-medium", want: "consumer backlog is building; monitor drain rate and investigate if it persists"},
		{name: "low backlog", posture: "consumer-backlog", severity: "backlog-low", want: "consumer backlog is present but small; continue observing drain progression"},
		{name: "advancing", posture: "consumer-advancing", severity: "", want: "consumer progress is advancing; continue observing steady backlog drain"},
		{name: "idle", posture: "consumer-idle", severity: "", want: "consumer group appears idle; verify whether no work is expected"},
	}

	for _, tc := range cases {
		if got := classifyEventProcessorConsumerBacklogHint(tc.posture, tc.severity); got != tc.want {
			t.Fatalf("%s: expected backlog hint %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestBuildEventProcessorKafkaConsumerProgressSnapshot(t *testing.T) {
	snapshot := buildEventProcessorKafkaConsumerProgressSnapshot(
		&eventProcessorTestKafkaHealth{
			consumerGroupState: map[string]interface{}{
				"active_consumers": int64(2),
			},
			consumerMetrics: map[string]int64{
				"lag": 15,
			},
		},
		"stalled",
	)

	if got := snapshot.ActiveConsumers; got != 2 {
		t.Fatalf("expected active consumers 2, got %d", got)
	}
	if got := snapshot.Lag; got != 15 {
		t.Fatalf("expected lag 15, got %d", got)
	}
	if got := snapshot.ProgressState; got != "lagging" {
		t.Fatalf("expected progress state lagging, got %q", got)
	}
}

func TestBuildEventProcessorKafkaConsumerProgressSnapshotFromHealthDetails(t *testing.T) {
	snapshot := buildEventProcessorKafkaConsumerProgressSnapshot(
		&eventProcessorTestKafkaHealth{
			details: map[string]interface{}{
				"active_consumers":   int64(3),
				"consumer_group_lag": int64(8),
				"max_tracked_offset": int64(144),
			},
		},
		"active",
	)

	if got := snapshot.ActiveConsumers; got != 3 {
		t.Fatalf("expected active consumers 3, got %d", got)
	}
	if got := snapshot.Lag; got != 8 {
		t.Fatalf("expected lag 8, got %d", got)
	}
	if got := snapshot.CurrentOffset; got != 144 {
		t.Fatalf("expected current offset 144, got %d", got)
	}
	if got := snapshot.ProgressState; got != "lagging" {
		t.Fatalf("expected progress state lagging, got %q", got)
	}
}

func TestBuildEventProcessorRuntimeRolloutHealthHandler(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler, err := buildEventProcessorRuntimeRolloutHealthHandler(
		context.Background(),
		"event-processor-1",
		logger,
		metrics,
		&eventProcessorTestDatabaseManager{mongoHealthy: true, postgresHealthy: true},
		&eventProcessorTestComponentHealth{status: "healthy", message: "event store ready"},
		&eventProcessorTestComponentHealth{status: "healthy", message: "metadata store ready"},
		&eventProcessorTestKafkaHealth{
			status:  "healthy",
			message: "kafka ready",
			details: map[string]interface{}{
				"message_count":      int64(12),
				"error_count":        int64(1),
				"max_tracked_offset": int64(144),
			},
			consumerGroupState: map[string]interface{}{
				"active_consumers": int64(2),
			},
			consumerMetrics: map[string]int64{
				"lag": 0,
			},
		},
		&eventProcessorTestProcessorRuntime{
			status:         "healthy",
			message:        "processor runtime healthy",
			processedCount: 5,
			failedCount:    1,
			duplicateCount: 1,
		},
		&eventProcessorTestConsumeRuntime{
			configuredTopics: 2,
			activeTopics:     2,
			running:          true,
		},
	)
	if err != nil {
		t.Fatalf("build runtime rollout health handler: %v", err)
	}
	if handler == nil {
		t.Fatal("expected handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()
	handler.HandleRollout(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload api.RolloutReportResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if !payload.Available || payload.Details == nil {
		t.Fatal("expected rollout details")
	}
	if err := api.ValidateMicroserviceRolloutMetadataParity(
		payload.Details,
		"event-processor",
		"event-processor-ownership-rollout-runtime",
		"microservice:event-processor-1",
	); err != nil {
		t.Fatalf("expected metadata parity validation: %v", err)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(payload.Details); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
	if got := payload.Details.Service; got != "event-processor" {
		t.Fatalf("expected service event-processor, got %q", got)
	}
	if got := payload.Details.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := payload.Details.Advisory.Status; got != "runtime-wired" {
		t.Fatalf("expected advisory status runtime-wired, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; got == "" || !eventProcessorContainsAll(got, []string{"kafka_message_count: 12", "kafka_error_count: 1", "kafka_activity_state: active"}) {
		t.Fatalf("expected advisory reason to include kafka activity details, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; !strings.Contains(got, "consumer_progress_posture: consumer-advancing") {
		t.Fatalf("expected advisory reason to include consumer progress posture, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; !strings.Contains(got, "consumer_operator_hint: consumer progress is advancing; continue observing steady backlog drain") {
		t.Fatalf("expected advisory reason to include consumer backlog hint, got %q", got)
	}
	if err := api.ValidateRolloutExecutionProgressReasonCoverage(
		payload.Details.Advisory.Reason,
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
}

func eventProcessorContainsAll(s string, patterns []string) bool {
	for _, pattern := range patterns {
		if !strings.Contains(s, pattern) {
			return false
		}
	}
	return true
}
