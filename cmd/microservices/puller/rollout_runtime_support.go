package main

import (
	"context"
	"fmt"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
)

type pullerKafkaHealthProvider interface {
	Health() *core.HealthStatus
}

type pullerRolloutRuntimeConfig struct {
	BlockchainRPCs     []string
	PollInterval       int
	CheckpointInterval int
}

func buildPullerRuntimeRolloutHealthHandler(
	ctx context.Context,
	instanceID string,
	logger core.Logger,
	metrics core.MetricsCollector,
	dbManager database.DatabaseManager,
	kafkaHealth pullerKafkaHealthProvider,
	config pullerRolloutRuntimeConfig,
	checkpointSource pullerCheckpointSource,
	progress *pullerLoopRuntimeProgress,
	execution pullerExecutionRuntimeStatusProvider,
) (*api.HealthCheckHandler, error) {
	healthHandler := api.NewHealthCheckHandler(dbManager, nil, logger, metrics)
	if err := healthHandler.Initialize(ctx); err != nil {
		return nil, err
	}

	healthHandler.SetRuntimeComponentProvider(func(ctx context.Context) *api.ComponentStatus {
		runtimeState := buildPullerRuntimeRolloutState(context.Background(), dbManager, kafkaHealth, config, checkpointSource, progress, execution)
		return buildPullerRuntimeComponentStatus(runtimeState, time.Now())
	})
	healthHandler.SetReadinessDetailsProvider(func(ctx context.Context) map[string]interface{} {
		runtimeState := buildPullerRuntimeRolloutState(context.Background(), dbManager, kafkaHealth, config, checkpointSource, progress, execution)
		return buildPullerRuntimeReadinessDetails(runtimeState)
	})
	healthHandler.SetRolloutReportProducer(newPullerRolloutReportProducer(instanceID, func() pullerRolloutRuntimeState {
		return buildPullerRuntimeRolloutState(context.Background(), dbManager, kafkaHealth, config, checkpointSource, progress, execution)
	}))

	return healthHandler, nil
}

func buildPullerRuntimeRolloutState(
	ctx context.Context,
	dbManager database.DatabaseManager,
	kafkaHealth pullerKafkaHealthProvider,
	config pullerRolloutRuntimeConfig,
	checkpointSource pullerCheckpointSource,
	progress *pullerLoopRuntimeProgress,
	execution pullerExecutionRuntimeStatusProvider,
) pullerRolloutRuntimeState {
	return buildPullerRuntimeRolloutStateAt(time.Now(), ctx, dbManager, kafkaHealth, config, checkpointSource, progress, execution)
}

func buildPullerRuntimeRolloutStateAt(
	now time.Time,
	ctx context.Context,
	dbManager database.DatabaseManager,
	kafkaHealth pullerKafkaHealthProvider,
	config pullerRolloutRuntimeConfig,
	checkpointSource pullerCheckpointSource,
	progress *pullerLoopRuntimeProgress,
	execution pullerExecutionRuntimeStatusProvider,
) pullerRolloutRuntimeState {
	state := pullerRolloutRuntimeState{
		BlockchainRPCsConfigured: len(config.BlockchainRPCs) > 0,
		PullerLoopConfigured:     config.PollInterval > 0,
		KafkaReady:               kafkaHealth != nil,
	}

	if dbManager != nil {
		state.DatabaseReady = dbManager.CheckPostgresHealth(ctx) == nil
		state.DatabaseHealthStatus, state.DatabaseHealthMessage = pullerDatabaseHealthFields(ctx, dbManager)
	}
	if kafkaHealth != nil {
		state.KafkaHealthStatus, state.KafkaHealthMessage = pullerHealthFields(kafkaHealth.Health())
	}
	checkpointSnapshot := pullerCheckpointSourceSnapshot{}
	if checkpointSource != nil {
		checkpointSnapshot = checkpointSource.Snapshot(ctx)
	}
	progressSnapshot := buildPullerPollProgressSnapshot(now, config.PollInterval, config.CheckpointInterval, checkpointSnapshot, progress)
	state.PollCount = progressSnapshot.PollCount
	state.LastPollUnix = progressSnapshot.LastPollUnix
	state.ObservedBlock = progressSnapshot.ObservedBlock
	state.ProcessedBlock = progressSnapshot.ProcessedBlock
	state.BlockGap = progressSnapshot.BlockGap
	state.CheckpointProgressState = progressSnapshot.CheckpointState
	state.BlocksUntilCheckpoint = progressSnapshot.BlocksUntilCheckpoint
	state.PersistedCheckpointBlock = progressSnapshot.PersistedCheckpointBlock
	state.BlocksSinceCheckpoint = progressSnapshot.BlocksSinceCheckpoint
	state.PersistedCheckpointState = progressSnapshot.PersistedCheckpointState
	state.ReorgCheckpointState = progressSnapshot.ReorgCheckpointState
	state.ReorgCheckpointBlock = progressSnapshot.ReorgCheckpointBlock
	state.CheckpointChainSummary = formatPullerCheckpointChainSummaryAt(now, config.PollInterval, checkpointSnapshot)
	state.CheckpointChainPostureSummary = formatPullerCheckpointChainPostureSummaryAt(now, config.PollInterval, checkpointSnapshot)
	state.CheckpointCoverageHint = formatPullerCheckpointCoverageSummary(checkpointSnapshot)
	state.CheckpointCoveragePosture = classifyPullerCheckpointCoveragePosture(checkpointSnapshot)
	state.PollActivityState = progressSnapshot.ActivityState
	state.CheckpointRecoveryHint = classifyPullerCheckpointRecoveryHint(
		state.ReorgCheckpointState,
		state.PersistedCheckpointState,
		state.CheckpointCoveragePosture,
		api.BuildRolloutExecutionProgressPosture(api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Poll: progressSnapshot,
		})).Poll,
	)
	if execution != nil {
		snapshot := execution.ExecutionSnapshot()
		state.ExecutionRuntimeEnabled = snapshot.Enabled
		state.ConfiguredPullerCount = snapshot.ConfiguredPullers
		state.AttachedPullerCount = snapshot.AttachedPullers
		state.PublishedEvents = snapshot.PublishedEvents
		state.PublishedMessages = snapshot.PublishedMessages
		state.SharedRuntimeCount = snapshot.RuntimeCount
		state.SharedRuntimeProcessed = snapshot.ProcessedEvents
		state.SharedRuntimeSkipped = snapshot.SkippedDuplicates
		state.SharedRuntimeFailures = snapshot.RoutedFailures
		state.SharedRuntimeLastChain = snapshot.LastCheckpointChain
		state.SharedRuntimeLastCursor = snapshot.LastCheckpointCursor
		state.SharedRuntimeLastBlock = int64(snapshot.LastCheckpointBlock)
		state.SharedRuntimeLastError = snapshot.LastError
	}

	return state
}

func classifyPullerPollActivityState(now time.Time, pollIntervalSeconds int, snapshot pullerLoopRuntimeProgressSnapshot) string {
	if pollIntervalSeconds <= 0 {
		return ""
	}
	if snapshot.PollCount == 0 || snapshot.LastPollUnix <= 0 {
		return "no-polls-yet"
	}

	threshold := int64(pollIntervalSeconds * 2)
	if threshold < 1 {
		threshold = 1
	}
	if now.Unix()-snapshot.LastPollUnix <= threshold {
		return "active"
	}
	return "stale"
}

func pullerHealthFields(status *core.HealthStatus) (string, string) {
	if status == nil {
		return "", ""
	}
	return status.Status, status.Message
}

func pullerDatabaseHealthFields(ctx context.Context, dbManager database.DatabaseManager) (string, string) {
	if dbManager == nil {
		return "", ""
	}

	health := dbManager.Health(ctx)
	healthMap, ok := health.(map[string]interface{})
	if !ok {
		return "", ""
	}

	status, _ := healthMap["status"].(string)
	if reason, ok := healthMap["reason"].(string); ok && reason != "" {
		return status, reason
	}

	postgresHealthy, hasPostgres := healthMap["postgres"].(bool)
	switch {
	case status == "healthy" && hasPostgres && postgresHealthy:
		return status, "postgres dependency is healthy"
	case status == "healthy" && hasPostgres && !postgresHealthy:
		return "degraded", "postgres dependency is unavailable while database manager reports partial health"
	case status != "":
		return status, fmt.Sprintf("database manager reported status %s", status)
	default:
		return "", ""
	}
}

func buildPullerRuntimeComponentStatus(runtimeState pullerRolloutRuntimeState, now time.Time) *api.ComponentStatus {
	completeness := classifyPullerRolloutWiringCompleteness(runtimeState)
	status := "degraded"
	switch completeness.AdvisoryStatus {
	case "runtime-wired":
		status = "healthy"
	case "runtime-wired-unhealthy":
		status = "unhealthy"
	}

	return &api.ComponentStatus{
		Name:      "Polling Runtime",
		Status:    status,
		Timestamp: now.Unix(),
		Details:   buildPullerRuntimeReadinessDetails(runtimeState),
	}
}

func buildPullerRuntimeReadinessDetails(runtimeState pullerRolloutRuntimeState) map[string]interface{} {
	completeness := classifyPullerRolloutWiringCompleteness(runtimeState)
	details := map[string]interface{}{
		"runtime_mode":               completeness.Mode,
		"rollout_ready":              completeness.AdvisoryReady,
		"rollout_status":             completeness.AdvisoryStatus,
		"rollout_reason":             completeness.Reason,
		"rollout_gate_decision":      "hold",
		"rollout_gate_reason":        completeness.Reason,
		"database_ready":             runtimeState.DatabaseReady,
		"kafka_ready":                runtimeState.KafkaReady,
		"puller_loop_configured":     runtimeState.PullerLoopConfigured,
		"blockchain_rpcs_configured": runtimeState.BlockchainRPCsConfigured,
		"poll_activity_state":        runtimeState.PollActivityState,
		"execution_runtime_enabled":  runtimeState.ExecutionRuntimeEnabled,
		"configured_puller_count":    runtimeState.ConfiguredPullerCount,
		"attached_puller_count":      runtimeState.AttachedPullerCount,
	}
	if completeness.AdvisoryReady {
		details["rollout_gate_decision"] = "allow"
	}
	if runtimeState.PollCount > 0 {
		details["poll_count"] = runtimeState.PollCount
	}
	if runtimeState.LastPollUnix > 0 {
		details["last_poll_unix"] = runtimeState.LastPollUnix
	}
	if runtimeState.ObservedBlock > 0 {
		details["observed_block"] = runtimeState.ObservedBlock
	}
	if runtimeState.ProcessedBlock > 0 {
		details["processed_block"] = runtimeState.ProcessedBlock
	}
	if runtimeState.BlockGap > 0 {
		details["block_gap"] = runtimeState.BlockGap
	}
	if runtimeState.CheckpointProgressState != "" {
		details["checkpoint_progress_state"] = runtimeState.CheckpointProgressState
	}
	if runtimeState.BlocksUntilCheckpoint > 0 {
		details["blocks_until_checkpoint"] = runtimeState.BlocksUntilCheckpoint
	}
	if runtimeState.PersistedCheckpointState != "" {
		details["persisted_checkpoint_state"] = runtimeState.PersistedCheckpointState
	}
	if runtimeState.PersistedCheckpointBlock > 0 {
		details["persisted_checkpoint_block"] = runtimeState.PersistedCheckpointBlock
	}
	if runtimeState.BlocksSinceCheckpoint > 0 {
		details["blocks_since_checkpoint"] = runtimeState.BlocksSinceCheckpoint
	}
	if runtimeState.ReorgCheckpointState != "" {
		details["reorg_checkpoint_state"] = runtimeState.ReorgCheckpointState
	}
	if runtimeState.ReorgCheckpointBlock > 0 {
		details["reorg_checkpoint_block"] = runtimeState.ReorgCheckpointBlock
	}
	if runtimeState.CheckpointCoveragePosture != "" {
		details["checkpoint_coverage_posture"] = runtimeState.CheckpointCoveragePosture
	}
	if runtimeState.CheckpointCoverageHint != "" {
		details["checkpoint_coverage"] = runtimeState.CheckpointCoverageHint
	}
	if runtimeState.CheckpointChainSummary != "" {
		details["checkpoint_chain_summary"] = runtimeState.CheckpointChainSummary
	}
	if runtimeState.CheckpointChainPostureSummary != "" {
		details["checkpoint_chain_posture_summary"] = runtimeState.CheckpointChainPostureSummary
	}
	if runtimeState.CheckpointRecoveryHint != "" {
		details["poll_operator_hint"] = runtimeState.CheckpointRecoveryHint
	}
	if runtimeState.PublishedEvents > 0 {
		details["published_events"] = runtimeState.PublishedEvents
	}
	if runtimeState.PublishedMessages > 0 {
		details["published_messages"] = runtimeState.PublishedMessages
	}
	if runtimeState.SharedRuntimeCount > 0 {
		details["shared_runtime_count"] = runtimeState.SharedRuntimeCount
	}
	if runtimeState.SharedRuntimeProcessed > 0 {
		details["shared_runtime_processed_events"] = runtimeState.SharedRuntimeProcessed
	}
	if runtimeState.SharedRuntimeSkipped > 0 {
		details["shared_runtime_skipped_duplicates"] = runtimeState.SharedRuntimeSkipped
	}
	if runtimeState.SharedRuntimeFailures > 0 {
		details["shared_runtime_routed_failures"] = runtimeState.SharedRuntimeFailures
	}
	if runtimeState.SharedRuntimeLastChain != "" {
		details["shared_runtime_last_checkpoint_chain"] = runtimeState.SharedRuntimeLastChain
	}
	if runtimeState.SharedRuntimeLastCursor != "" {
		details["shared_runtime_last_checkpoint_cursor"] = runtimeState.SharedRuntimeLastCursor
	}
	if runtimeState.SharedRuntimeLastBlock > 0 {
		details["shared_runtime_last_checkpoint_block"] = runtimeState.SharedRuntimeLastBlock
	}
	if runtimeState.SharedRuntimeLastError != "" {
		details["shared_runtime_last_error"] = runtimeState.SharedRuntimeLastError
	}

	return details
}

func buildPullerRuntimeSummary(
	runtimeState pullerRolloutRuntimeState,
	metrics core.MetricsCollector,
	now time.Time,
	authEnabled bool,
	rateLimitEnabled bool,
) *pullerRuntimeSummaryResponse {
	completeness := classifyPullerRolloutWiringCompleteness(runtimeState)
	componentStatus := buildPullerRuntimeComponentStatus(runtimeState, now)

	return &pullerRuntimeSummaryResponse{
		Service:        "puller",
		Timestamp:      now.Unix(),
		DeploymentMode: "microservice",
		RuntimeMode:    completeness.Mode,
		RuntimePosture: completeness.AdvisoryStatus,
		ComponentState: componentStatus.Status,
		Rollout:        buildPullerRuntimeReadinessDetails(runtimeState),
		Security:       buildPullerSecurityRuntimeSection(authEnabled, rateLimitEnabled),
		Metrics:        buildPullerMetricsSummary(metrics),
	}
}

func buildPullerSecurityRuntimeSection(authEnabled, rateLimitEnabled bool) map[string]interface{} {
	authPosture := classifyPullerAuthPosture(authEnabled)
	rateLimitPosture := classifyPullerRateLimitPosture(rateLimitEnabled)
	securityPosture := classifyPullerSecurityPosture(authEnabled, rateLimitEnabled)

	return map[string]interface{}{
		"route_boundary":     "runtime-entrypoint",
		"auth_enabled":       authEnabled,
		"rate_limit_enabled": rateLimitEnabled,
		"auth_posture":       authPosture,
		"rate_limit_posture": rateLimitPosture,
		"security_posture":   securityPosture,
		"security_hint":      classifyPullerSecurityHint(authEnabled, rateLimitEnabled),
		"runtime_boundary":   "optional-security-surface",
	}
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyPullerAuthPosture(enabled bool) string {
	if !enabled {
		return "puller-auth-unconfigured"
	}

	return "puller-auth-ready"
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyPullerRateLimitPosture(enabled bool) string {
	if !enabled {
		return "puller-rate-limit-unconfigured"
	}

	return "puller-rate-limit-ready"
}

func classifyPullerSecurityPosture(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "puller-security-unconfigured"
	case authEnabled && rateLimitEnabled:
		return "puller-security-ready"
	default:
		return "puller-security-partial"
	}
}

func classifyPullerSecurityHint(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "puller security controls are disabled by default; enable auth or rate limiting explicitly before exposing the runtime/control surface"
	case authEnabled && rateLimitEnabled:
		return "puller security controls are aligned for the current runtime/control baseline"
	case authEnabled || rateLimitEnabled:
		return "puller security controls are partially enabled; verify the remaining control surface before treating the runtime/control surface as hardened"
	default:
		return "puller security controls are partially configured; verify auth and rate limit wiring"
	}
}

func buildPullerMetricsSummary(metrics core.MetricsCollector) map[string]interface{} {
	summary := map[string]interface{}{
		"counter_count":   0,
		"gauge_count":     0,
		"histogram_count": 0,
	}
	if metrics == nil {
		summary["collector_state"] = "unavailable"
		return summary
	}

	exported := metrics.GetMetrics()
	summary["collector_state"] = "available"
	summary["counter_count"] = pullerMetricsSectionCount(exported, "counters")
	summary["gauge_count"] = pullerMetricsSectionCount(exported, "gauges")
	summary["histogram_count"] = pullerMetricsSectionCount(exported, "histograms")
	summary["execution_summary"] = fmt.Sprintf(
		"counters=%d gauges=%d histograms=%d",
		summary["counter_count"],
		summary["gauge_count"],
		summary["histogram_count"],
	)

	return summary
}

func pullerMetricsSectionCount(exported map[string]interface{}, section string) int {
	if exported == nil {
		return 0
	}
	values, ok := exported[section].(map[string]interface{})
	if !ok {
		return 0
	}
	return len(values)
}
