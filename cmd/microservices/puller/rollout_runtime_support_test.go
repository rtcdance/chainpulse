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

type pullerTestDatabaseManager struct {
	postgresHealthy bool
}

var _ database.DatabaseManager = (*pullerTestDatabaseManager)(nil)

func (m *pullerTestDatabaseManager) Initialize(ctx context.Context) error { return nil }
func (m *pullerTestDatabaseManager) GetMongoClient(ctx context.Context) (any, error) {
	return nil, nil
}
func (m *pullerTestDatabaseManager) GetMongoDatabase(name string) *mongo.Database { return nil }
func (m *pullerTestDatabaseManager) GetPostgresDB(ctx context.Context) (any, error) {
	return nil, nil
}
func (m *pullerTestDatabaseManager) CheckMongoHealth(ctx context.Context) error { return nil }
func (m *pullerTestDatabaseManager) CheckPostgresHealth(ctx context.Context) error {
	if m.postgresHealthy {
		return nil
	}
	return context.DeadlineExceeded
}

func (m *pullerTestDatabaseManager) Health(ctx context.Context) any {
	if m.postgresHealthy {
		return map[string]any{
			"status":   "healthy",
			"postgres": true,
		}
	}
	return map[string]any{
		"status":   "unhealthy",
		"postgres": false,
		"reason":   "postgres unreachable",
	}
}
func (m *pullerTestDatabaseManager) Close(ctx context.Context) error { return nil }

type pullerTestKafkaHealth struct {
	status  string
	message string
}

func (k *pullerTestKafkaHealth) Health() *core.HealthStatus {
	return &core.HealthStatus{
		Status:  k.status,
		Message: k.message,
	}
}

func TestBuildPullerRuntimeRolloutState(t *testing.T) {
	state := buildPullerRuntimeRolloutState(
		context.Background(),
		&pullerTestDatabaseManager{postgresHealthy: true},
		&pullerTestKafkaHealth{status: "healthy"},
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     []string{"http://ethereum-rpc:8545"},
			PollInterval:       12,
			CheckpointInterval: 100,
		},
		nil,
		nil,
		nil,
	)

	if !state.DatabaseReady {
		t.Fatal("expected database ready")
	}
	if got := state.DatabaseHealthStatus; got != "healthy" {
		t.Fatalf("expected healthy database status, got %q", got)
	}
	if got := state.DatabaseHealthMessage; got != "postgres dependency is healthy" {
		t.Fatalf("expected database health message, got %q", got)
	}
	if !state.KafkaReady {
		t.Fatal("expected kafka ready")
	}
	if got := state.KafkaHealthStatus; got != "healthy" {
		t.Fatalf("expected healthy kafka status, got %q", got)
	}
	if !state.BlockchainRPCsConfigured {
		t.Fatal("expected blockchain RPCs configured")
	}
	if !state.PullerLoopConfigured {
		t.Fatal("expected puller loop configured")
	}
}

func TestBuildPullerRuntimeRolloutStateIncludesProgress(t *testing.T) {
	progress := &pullerLoopRuntimeProgress{}
	progress.recordPoll(time.Unix(1_712_345_678, 0))
	progress.recordObservedBlock(120)
	progress.recordProcessedBlock(118)

	state := buildPullerRuntimeRolloutStateAt(
		time.Unix(1_712_345_690, 0),
		context.Background(),
		&pullerTestDatabaseManager{postgresHealthy: true},
		&pullerTestKafkaHealth{status: "healthy"},
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     []string{"http://ethereum-rpc:8545"},
			PollInterval:       12,
			CheckpointInterval: 100,
		},
		nil,
		progress,
		nil,
	)

	if got := state.PollCount; got != 1 {
		t.Fatalf("expected poll count 1, got %d", got)
	}
	if got := state.LastPollUnix; got != 1_712_345_678 {
		t.Fatalf("expected last poll unix 1712345678, got %d", got)
	}
	if got := state.ObservedBlock; got != 120 {
		t.Fatalf("expected observed block 120, got %d", got)
	}
	if got := state.ProcessedBlock; got != 118 {
		t.Fatalf("expected processed block 118, got %d", got)
	}
	if got := state.BlockGap; got != 2 {
		t.Fatalf("expected block gap 2, got %d", got)
	}
	if got := state.CheckpointProgressState; got != "checkpoint-pending" {
		t.Fatalf("expected checkpoint progress state checkpoint-pending, got %q", got)
	}
	if got := state.BlocksUntilCheckpoint; got != 82 {
		t.Fatalf("expected blocks until checkpoint 82, got %d", got)
	}
	if got := state.PersistedCheckpointState; got != "persisted-checkpoint-missing" {
		t.Fatalf("expected persisted checkpoint state persisted-checkpoint-missing, got %q", got)
	}
	if got := state.ReorgCheckpointState; got != "reorg-clear" {
		t.Fatalf("expected reorg checkpoint state reorg-clear, got %q", got)
	}
	if got := state.CheckpointChainSummary; got != "" {
		t.Fatalf("expected empty checkpoint chain summary without persisted checkpoints, got %q", got)
	}
	if got := state.CheckpointChainPostureSummary; got != "" {
		t.Fatalf("expected empty checkpoint chain posture summary without persisted checkpoints, got %q", got)
	}
	if got := state.CheckpointCoverageHint; got != "" {
		t.Fatalf("expected empty checkpoint coverage hint without persisted checkpoints, got %q", got)
	}
	if got := state.CheckpointCoveragePosture; got != "" {
		t.Fatalf("expected empty checkpoint coverage posture without persisted checkpoints, got %q", got)
	}
	if got := state.CheckpointRecoveryHint; got != "no persisted checkpoint has been recorded yet; verify checkpoint creation before relying on recovery posture" {
		t.Fatalf("expected checkpoint recovery hint for missing checkpoint, got %q", got)
	}
	if got := state.PollActivityState; got != "active" {
		t.Fatalf("expected poll activity state active, got %q", got)
	}
}

func TestBuildPullerRuntimeRolloutStateIncludesPersistedCheckpoint(t *testing.T) {
	progress := &pullerLoopRuntimeProgress{}
	progress.recordPoll(time.Unix(1_712_345_678, 0))
	progress.recordObservedBlock(120)
	progress.recordProcessedBlock(118)
	checkpointSource := newPullerRuntimeCheckpointSource()
	if err := checkpointSource.SaveCheckpoint(context.Background(), "ethereum", 100); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	state := buildPullerRuntimeRolloutStateAt(
		time.Unix(1_712_345_690, 0),
		context.Background(),
		&pullerTestDatabaseManager{postgresHealthy: true},
		&pullerTestKafkaHealth{status: "healthy"},
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     []string{"http://ethereum-rpc:8545"},
			PollInterval:       12,
			CheckpointInterval: 100,
		},
		checkpointSource,
		progress,
		nil,
	)

	if got := state.PersistedCheckpointBlock; got != 100 {
		t.Fatalf("expected persisted checkpoint block 100, got %d", got)
	}
	if got := state.BlocksSinceCheckpoint; got != 18 {
		t.Fatalf("expected blocks since checkpoint 18, got %d", got)
	}
	if got := state.PersistedCheckpointState; got != "persisted-checkpoint-behind" {
		t.Fatalf("expected persisted checkpoint state persisted-checkpoint-behind, got %q", got)
	}
	if got := state.ReorgCheckpointState; got != "reorg-clear" {
		t.Fatalf("expected reorg checkpoint state reorg-clear, got %q", got)
	}
	if got := state.CheckpointChainSummary; got != "ethereum=checkpoint-recorded:fresh@100" {
		t.Fatalf("expected checkpoint chain summary ethereum=checkpoint-recorded:fresh@100, got %q", got)
	}
	if got := state.CheckpointChainPostureSummary; got != "ethereum=recorded-healthy" {
		t.Fatalf("expected checkpoint chain posture summary ethereum=recorded-healthy, got %q", got)
	}
	if got := state.CheckpointCoverageHint; got != "tracked=1,recorded=1,reorg_risk=0,reorg_reconciled=0" {
		t.Fatalf("expected checkpoint coverage hint tracked=1,recorded=1,reorg_risk=0,reorg_reconciled=0, got %q", got)
	}
	if got := state.CheckpointCoveragePosture; got != "coverage-healthy" {
		t.Fatalf("expected checkpoint coverage posture coverage-healthy, got %q", got)
	}
	if got := state.CheckpointRecoveryHint; got != "persisted checkpoint is behind live progress; continue observing catch-up and checkpoint advancement" {
		t.Fatalf("expected checkpoint recovery hint for catch-up, got %q", got)
	}
}

func TestBuildPullerRuntimeRolloutStateIncludesReconciledCheckpoint(t *testing.T) {
	progress := &pullerLoopRuntimeProgress{}
	progress.recordPoll(time.Unix(1_712_345_678, 0))
	progress.recordObservedBlock(220)
	progress.recordProcessedBlock(200)
	checkpointSource := newPullerRuntimeCheckpointSource()
	if err := checkpointSource.SaveCheckpoint(context.Background(), "ethereum", 100); err != nil {
		t.Fatalf("save initial checkpoint: %v", err)
	}
	if err := checkpointSource.ObserveChainProgress(context.Background(), "ethereum", 90); err != nil {
		t.Fatalf("observe regressed progress: %v", err)
	}
	if err := checkpointSource.SaveCheckpoint(context.Background(), "ethereum", 200); err != nil {
		t.Fatalf("save reconciled checkpoint: %v", err)
	}

	state := buildPullerRuntimeRolloutStateAt(
		time.Unix(1_712_345_690, 0),
		context.Background(),
		&pullerTestDatabaseManager{postgresHealthy: true},
		&pullerTestKafkaHealth{status: "healthy"},
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     []string{"http://ethereum-rpc:8545"},
			PollInterval:       12,
			CheckpointInterval: 100,
		},
		checkpointSource,
		progress,
		nil,
	)

	if got := state.ReorgCheckpointState; got != "reorg-reconciled" {
		t.Fatalf("expected reorg checkpoint state reorg-reconciled, got %q", got)
	}
	if got := state.ReorgCheckpointBlock; got != 200 {
		t.Fatalf("expected reorg checkpoint block 200, got %d", got)
	}
	if got := state.CheckpointChainSummary; got != "ethereum=reorg-reconciled:fresh@200" {
		t.Fatalf("expected checkpoint chain summary ethereum=reorg-reconciled:fresh@200, got %q", got)
	}
	if got := state.CheckpointChainPostureSummary; got != "ethereum=reconciled" {
		t.Fatalf("expected checkpoint chain posture summary ethereum=reconciled, got %q", got)
	}
	if got := state.CheckpointCoverageHint; got != "tracked=1,recorded=0,reorg_risk=0,reorg_reconciled=1" {
		t.Fatalf("expected checkpoint coverage hint tracked=1,recorded=0,reorg_risk=0,reorg_reconciled=1, got %q", got)
	}
	if got := state.CheckpointCoveragePosture; got != "coverage-reconciled" {
		t.Fatalf("expected checkpoint coverage posture coverage-reconciled, got %q", got)
	}
	if got := state.CheckpointRecoveryHint; got != "checkpoint recovery has reconciled recent risk; continue observing fresh checkpoint coverage" {
		t.Fatalf("expected checkpoint recovery hint for reconciled state, got %q", got)
	}
}

func TestBuildPullerPollProgressSnapshot(t *testing.T) {
	progress := &pullerLoopRuntimeProgress{}
	progress.recordPoll(time.Unix(1_712_345_678, 0))
	progress.recordObservedBlock(120)
	progress.recordProcessedBlock(118)

	snapshot := buildPullerPollProgressSnapshot(
		time.Unix(1_712_345_690, 0),
		12,
		100,
		pullerCheckpointSourceSnapshot{},
		progress,
	)

	if got := snapshot.PollCount; got != 1 {
		t.Fatalf("expected poll count 1, got %d", got)
	}
	if got := snapshot.LastPollUnix; got != 1_712_345_678 {
		t.Fatalf("expected last poll unix 1712345678, got %d", got)
	}
	if got := snapshot.ObservedBlock; got != 120 {
		t.Fatalf("expected observed block 120, got %d", got)
	}
	if got := snapshot.ProcessedBlock; got != 118 {
		t.Fatalf("expected processed block 118, got %d", got)
	}
	if got := snapshot.BlockGap; got != 2 {
		t.Fatalf("expected block gap 2, got %d", got)
	}
	if got := snapshot.CheckpointState; got != "checkpoint-pending" {
		t.Fatalf("expected checkpoint state checkpoint-pending, got %q", got)
	}
	if got := snapshot.BlocksUntilCheckpoint; got != 82 {
		t.Fatalf("expected blocks until checkpoint 82, got %d", got)
	}
	if got := snapshot.PersistedCheckpointState; got != "persisted-checkpoint-missing" {
		t.Fatalf("expected persisted checkpoint state persisted-checkpoint-missing, got %q", got)
	}
	if got := snapshot.ReorgCheckpointState; got != "reorg-clear" {
		t.Fatalf("expected reorg checkpoint state reorg-clear, got %q", got)
	}
	if got := snapshot.ActivityState; got != "active" {
		t.Fatalf("expected poll activity state active, got %q", got)
	}
}

func TestBuildPullerPollProgressSnapshotIncludesPersistedCheckpoint(t *testing.T) {
	progress := &pullerLoopRuntimeProgress{}
	progress.recordPoll(time.Unix(1_712_345_678, 0))
	progress.recordObservedBlock(120)
	progress.recordProcessedBlock(118)

	snapshot := buildPullerPollProgressSnapshot(
		time.Unix(1_712_345_690, 0),
		12,
		100,
		pullerCheckpointSourceSnapshot{HighestCheckpointBlock: 100},
		progress,
	)

	if got := snapshot.PersistedCheckpointBlock; got != 100 {
		t.Fatalf("expected persisted checkpoint block 100, got %d", got)
	}
	if got := snapshot.BlocksSinceCheckpoint; got != 18 {
		t.Fatalf("expected blocks since checkpoint 18, got %d", got)
	}
	if got := snapshot.PersistedCheckpointState; got != "persisted-checkpoint-behind" {
		t.Fatalf("expected persisted checkpoint state persisted-checkpoint-behind, got %q", got)
	}
	if got := snapshot.ReorgCheckpointState; got != "reorg-clear" {
		t.Fatalf("expected reorg checkpoint state reorg-clear, got %q", got)
	}
}

func TestBuildPullerPollProgressSnapshotWithoutProgress(t *testing.T) {
	snapshot := buildPullerPollProgressSnapshot(
		time.Unix(1_712_345_690, 0),
		12,
		100,
		pullerCheckpointSourceSnapshot{},
		nil,
	)

	if got := snapshot.PollCount; got != 0 {
		t.Fatalf("expected poll count 0, got %d", got)
	}
	if got := snapshot.LastPollUnix; got != 0 {
		t.Fatalf("expected last poll unix 0, got %d", got)
	}
	if got := snapshot.ActivityState; got != "no-polls-yet" {
		t.Fatalf("expected poll activity state no-polls-yet, got %q", got)
	}
	if got := snapshot.CheckpointState; got != "checkpoint-uninitialized" {
		t.Fatalf("expected checkpoint state checkpoint-uninitialized, got %q", got)
	}
	if got := snapshot.PersistedCheckpointState; got != "persisted-checkpoint-missing" {
		t.Fatalf("expected persisted checkpoint state persisted-checkpoint-missing, got %q", got)
	}
	if got := snapshot.ReorgCheckpointState; got != "reorg-clear" {
		t.Fatalf("expected reorg checkpoint state reorg-clear, got %q", got)
	}
}

func TestClassifyPullerCheckpointProgress(t *testing.T) {
	cases := []struct {
		name            string
		processedBlock  int64
		checkpointInt   int
		wantState       string
		wantBlocksUntil int64
	}{
		{"disabled", 118, 0, "", 0},
		{"uninitialized", 0, 100, "checkpoint-uninitialized", 0},
		{"due", 200, 100, "checkpoint-due", 0},
		{"pending", 118, 100, "checkpoint-pending", 82},
	}

	for _, tc := range cases {
		gotState, gotBlocksUntil := classifyPullerCheckpointProgress(tc.processedBlock, tc.checkpointInt)
		if gotState != tc.wantState || gotBlocksUntil != tc.wantBlocksUntil {
			t.Fatalf("%s: expected (%q,%d), got (%q,%d)", tc.name, tc.wantState, tc.wantBlocksUntil, gotState, gotBlocksUntil)
		}
	}
}

func TestClassifyPullerPersistedCheckpoint(t *testing.T) {
	cases := []struct {
		name            string
		processedBlock  int64
		snapshot        pullerCheckpointSourceSnapshot
		wantState       string
		wantBlocksSince int64
	}{
		{"missing", 118, pullerCheckpointSourceSnapshot{}, "persisted-checkpoint-missing", 0},
		{"present without processed", 0, pullerCheckpointSourceSnapshot{HighestCheckpointBlock: 100}, "persisted-checkpoint-present", 0},
		{"current", 100, pullerCheckpointSourceSnapshot{HighestCheckpointBlock: 100}, "persisted-checkpoint-current", 0},
		{"behind", 118, pullerCheckpointSourceSnapshot{HighestCheckpointBlock: 100}, "persisted-checkpoint-behind", 18},
	}

	for _, tc := range cases {
		gotState, gotBlocksSince := classifyPullerPersistedCheckpoint(tc.processedBlock, tc.snapshot)
		if gotState != tc.wantState || gotBlocksSince != tc.wantBlocksSince {
			t.Fatalf("%s: expected (%q,%d), got (%q,%d)", tc.name, tc.wantState, tc.wantBlocksSince, gotState, gotBlocksSince)
		}
	}
}

func TestClassifyPullerCheckpointReorgRisk(t *testing.T) {
	cases := []struct {
		name      string
		snapshot  pullerCheckpointSourceSnapshot
		wantState string
		wantBlock int64
	}{
		{"clear", pullerCheckpointSourceSnapshot{}, "reorg-clear", 0},
		{"risk", pullerCheckpointSourceSnapshot{LastReorgRiskBlock: 100}, "reorg-risk", 100},
		{"reconciled-only", pullerCheckpointSourceSnapshot{LastReconciledBlock: 200}, "reorg-reconciled", 200},
		{"reconciled-after-risk", pullerCheckpointSourceSnapshot{LastReorgRiskBlock: 100, LastReorgRiskUnix: 10, LastReconciledBlock: 200, LastReconciledUnix: 11}, "reorg-reconciled", 200},
	}

	for _, tc := range cases {
		gotState, gotBlock := classifyPullerCheckpointReorgRisk(tc.snapshot)
		if gotState != tc.wantState || gotBlock != tc.wantBlock {
			t.Fatalf("%s: expected (%q,%d), got (%q,%d)", tc.name, tc.wantState, tc.wantBlock, gotState, gotBlock)
		}
	}
}

func TestFormatPullerCheckpointChainSummaryAt(t *testing.T) {
	now := time.Unix(1_712_345_690, 0)
	snapshot := pullerCheckpointSourceSnapshot{
		ChainSummaries: []pullerCheckpointChainSummary{
			{
				ChainID:          "eth",
				CheckpointBlock:  100,
				CheckpointStatus: "checkpoint-recorded",
				LastUpdatedUnix:  1_712_345_678,
			},
			{
				ChainID:          "polygon",
				CheckpointBlock:  200,
				CheckpointStatus: "reorg-risk",
				LastUpdatedUnix:  1_712_345_500,
			},
		},
	}

	got := formatPullerCheckpointChainSummaryAt(now, 12, snapshot)
	want := "eth=checkpoint-recorded:fresh@100,polygon=reorg-risk:stale@200"
	if got != want {
		t.Fatalf("expected checkpoint chain summary %q, got %q", want, got)
	}
}

func TestFormatPullerCheckpointChainPostureSummaryAt(t *testing.T) {
	now := time.Unix(1_712_345_690, 0)
	snapshot := pullerCheckpointSourceSnapshot{
		ChainSummaries: []pullerCheckpointChainSummary{
			{
				ChainID:          "eth",
				CheckpointBlock:  100,
				CheckpointStatus: "checkpoint-recorded",
				LastUpdatedUnix:  1_712_345_678,
			},
			{
				ChainID:          "polygon",
				CheckpointBlock:  200,
				CheckpointStatus: "reorg-risk",
				LastUpdatedUnix:  1_712_345_500,
			},
		},
	}

	got := formatPullerCheckpointChainPostureSummaryAt(now, 12, snapshot)
	want := "eth=recorded-healthy,polygon=risk-stale"
	if got != want {
		t.Fatalf("expected checkpoint chain posture summary %q, got %q", want, got)
	}
}

func TestFormatPullerCheckpointCoverageSummary(t *testing.T) {
	snapshot := pullerCheckpointSourceSnapshot{
		TrackedChains: 3,
		ChainSummaries: []pullerCheckpointChainSummary{
			{ChainID: "eth", CheckpointStatus: "checkpoint-recorded"},
			{ChainID: "polygon", CheckpointStatus: "reorg-risk"},
			{ChainID: "arbitrum", CheckpointStatus: "reorg-reconciled"},
		},
	}

	got := formatPullerCheckpointCoverageSummary(snapshot)
	want := "tracked=3,recorded=1,reorg_risk=1,reorg_reconciled=1"
	if got != want {
		t.Fatalf("expected checkpoint coverage summary %q, got %q", want, got)
	}
}

func TestClassifyPullerPollActivityState(t *testing.T) {
	cases := []struct {
		name         string
		nowUnix      int64
		pollInterval int
		snapshot     pullerLoopRuntimeProgressSnapshot
		want         string
	}{
		{
			name:         "no polls yet",
			nowUnix:      1_712_345_690,
			pollInterval: 12,
			snapshot:     pullerLoopRuntimeProgressSnapshot{},
			want:         "no-polls-yet",
		},
		{
			name:         "active",
			nowUnix:      1_712_345_690,
			pollInterval: 12,
			snapshot: pullerLoopRuntimeProgressSnapshot{
				PollCount:    1,
				LastPollUnix: 1_712_345_678,
			},
			want: "active",
		},
		{
			name:         "stale",
			nowUnix:      1_712_345_800,
			pollInterval: 12,
			snapshot: pullerLoopRuntimeProgressSnapshot{
				PollCount:    1,
				LastPollUnix: 1_712_345_678,
			},
			want: "stale",
		},
	}

	for _, tc := range cases {
		got := classifyPullerPollActivityState(time.Unix(tc.nowUnix, 0), tc.pollInterval, tc.snapshot)
		if got != tc.want {
			t.Fatalf("%s: expected poll activity state %q, got %q", tc.name, tc.want, got)
		}
	}
}

func TestBuildPullerRuntimeRolloutHealthHandler(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	progress := &pullerLoopRuntimeProgress{}
	progress.recordPoll(time.Unix(1_712_345_678, 0))
	progress.recordObservedBlock(120)
	progress.recordProcessedBlock(118)
	checkpointSource := newPullerRuntimeCheckpointSource()
	if err := checkpointSource.SaveCheckpoint(context.Background(), "ethereum", 100); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	handler, err := buildPullerRuntimeRolloutHealthHandler(
		context.Background(),
		"puller-1",
		logger,
		metrics,
		&pullerTestDatabaseManager{postgresHealthy: true},
		&pullerTestKafkaHealth{status: "healthy"},
		pullerRolloutRuntimeConfig{
			BlockchainRPCs:     []string{"http://ethereum-rpc:8545"},
			PollInterval:       12,
			CheckpointInterval: 100,
		},
		checkpointSource,
		progress,
		nil,
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
		"puller",
		"puller-ownership-rollout-runtime",
		"microservice:puller-1",
	); err != nil {
		t.Fatalf("expected metadata parity validation: %v", err)
	}
	if err := api.ValidateMicroserviceRuntimeDerivedRolloutParity(payload.Details); err != nil {
		t.Fatalf("expected runtime-derived parity validation: %v", err)
	}
	if got := payload.Details.Service; got != "puller" {
		t.Fatalf("expected service puller, got %q", got)
	}
	if got := payload.Details.Mode; got != "runtime-wired" {
		t.Fatalf("expected mode runtime-wired, got %q", got)
	}
	if got := payload.Details.Advisory.Status; got != "runtime-wired" {
		t.Fatalf("expected advisory status runtime-wired, got %q", got)
	}
	if err := api.ValidateRolloutExecutionProgressReasonCoverage(
		payload.Details.Advisory.Reason,
		api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Poll: api.RolloutPollProgressSnapshot{
				PollCount:                1,
				LastPollUnix:             1712345678,
				ObservedBlock:            120,
				ProcessedBlock:           118,
				BlockGap:                 2,
				CheckpointState:          "checkpoint-pending",
				BlocksUntilCheckpoint:    82,
				PersistedCheckpointBlock: 100,
				BlocksSinceCheckpoint:    18,
				PersistedCheckpointState: "persisted-checkpoint-behind",
				ReorgCheckpointState:     "reorg-clear",
			},
		}),
	); err != nil {
		t.Fatalf("expected execution progress reason coverage validation: %v", err)
	}
	if err := api.ValidateRolloutExecutionProgressPostureReasonCoverage(
		payload.Details.Advisory.Reason,
		api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Poll: api.RolloutPollProgressSnapshot{
				PollCount:                1,
				LastPollUnix:             1712345678,
				ObservedBlock:            120,
				ProcessedBlock:           118,
				BlockGap:                 2,
				CheckpointState:          "checkpoint-pending",
				BlocksUntilCheckpoint:    82,
				PersistedCheckpointBlock: 100,
				BlocksSinceCheckpoint:    18,
				PersistedCheckpointState: "persisted-checkpoint-behind",
				ReorgCheckpointState:     "reorg-clear",
				ActivityState:            "active",
			},
		}),
	); err != nil {
		t.Fatalf("expected execution progress posture reason coverage validation: %v", err)
	}
	if got := payload.Details.Advisory.Reason; !containsSubstring(got, "checkpoint_chain_summary: ethereum=checkpoint-recorded:fresh@100") {
		t.Fatalf("expected advisory reason to include checkpoint chain summary, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; !containsSubstring(got, "checkpoint_chain_posture_summary: ethereum=recorded-healthy") {
		t.Fatalf("expected advisory reason to include checkpoint chain posture summary, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; !containsSubstring(got, "checkpoint_coverage: tracked=1,recorded=1,reorg_risk=0,reorg_reconciled=0") {
		t.Fatalf("expected advisory reason to include checkpoint coverage hint, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; !containsSubstring(got, "checkpoint_coverage_posture: coverage-healthy") {
		t.Fatalf("expected advisory reason to include checkpoint coverage posture, got %q", got)
	}
	if got := payload.Details.Advisory.Reason; !containsSubstring(got, "poll_operator_hint: persisted checkpoint is behind live progress; continue observing catch-up and checkpoint advancement") {
		t.Fatalf("expected advisory reason to include checkpoint recovery hint, got %q", got)
	}
}

func TestBuildPullerRuntimeComponentStatus(t *testing.T) {
	status := buildPullerRuntimeComponentStatus(pullerRolloutRuntimeState{
		DatabaseReady:             true,
		KafkaReady:                true,
		PullerLoopConfigured:      true,
		BlockchainRPCsConfigured:  true,
		DatabaseHealthStatus:      "healthy",
		DatabaseHealthMessage:     "postgres dependency is healthy",
		KafkaHealthStatus:         "healthy",
		KafkaHealthMessage:        "kafka dependency is healthy",
		PollActivityState:         "active",
		CheckpointCoveragePosture: "coverage-healthy",
		CheckpointRecoveryHint:    "persisted checkpoint is behind live progress; continue observing catch-up and checkpoint advancement",
		PersistedCheckpointState:  "persisted-checkpoint-behind",
		CheckpointProgressState:   "checkpoint-pending",
		PollCount:                 1,
		LastPollUnix:              1_712_345_678,
	}, time.Unix(1_712_345_690, 0))

	if status == nil {
		t.Fatal("expected component status")
	}
	if got := status.Status; got != "healthy" {
		t.Fatalf("expected healthy component status, got %q", got)
	}
	if got := status.Details["runtime_mode"]; got != "runtime-wired" {
		t.Fatalf("expected runtime mode runtime-wired, got %v", got)
	}
	if got := status.Details["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout gate decision allow, got %v", got)
	}
}

func TestBuildPullerRuntimeReadinessDetails(t *testing.T) {
	details := buildPullerRuntimeReadinessDetails(pullerRolloutRuntimeState{
		DatabaseReady:                 true,
		KafkaReady:                    true,
		PullerLoopConfigured:          true,
		BlockchainRPCsConfigured:      true,
		DatabaseHealthStatus:          "healthy",
		DatabaseHealthMessage:         "postgres dependency is healthy",
		KafkaHealthStatus:             "healthy",
		KafkaHealthMessage:            "kafka dependency is healthy",
		PollActivityState:             "active",
		PollCount:                     1,
		LastPollUnix:                  1_712_345_678,
		ObservedBlock:                 120,
		ProcessedBlock:                118,
		BlockGap:                      2,
		CheckpointProgressState:       "checkpoint-pending",
		BlocksUntilCheckpoint:         82,
		PersistedCheckpointState:      "persisted-checkpoint-behind",
		PersistedCheckpointBlock:      100,
		BlocksSinceCheckpoint:         18,
		ReorgCheckpointState:          "reorg-clear",
		CheckpointCoveragePosture:     "coverage-healthy",
		CheckpointCoverageHint:        "tracked=1,recorded=1,reorg_risk=0,reorg_reconciled=0",
		CheckpointChainSummary:        "ethereum=checkpoint-recorded:fresh@100",
		CheckpointChainPostureSummary: "ethereum=recorded-healthy",
		CheckpointRecoveryHint:        "persisted checkpoint is behind live progress; continue observing catch-up and checkpoint advancement",
	})

	if got := details["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout gate decision allow, got %v", got)
	}
	if got := details["observed_block"]; got != int64(120) {
		t.Fatalf("expected observed block 120, got %v", got)
	}
	if got := details["checkpoint_chain_summary"]; got != "ethereum=checkpoint-recorded:fresh@100" {
		t.Fatalf("expected checkpoint chain summary, got %v", got)
	}
	if got := details["poll_operator_hint"]; got != "persisted checkpoint is behind live progress; continue observing catch-up and checkpoint advancement" {
		t.Fatalf("expected poll operator hint, got %v", got)
	}
}

func containsSubstring(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
