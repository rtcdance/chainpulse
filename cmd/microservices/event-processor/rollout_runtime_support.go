package main

import (
	"context"
	"fmt"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/infrastructure/database"
	"chainpulse/pkg/plugins/api"
)

type eventProcessorComponentHealthProvider interface {
	Health(context.Context) *core.HealthStatus
}

type eventProcessorKafkaHealthProvider interface {
	Health() *core.HealthStatus
}

type eventProcessorKafkaConsumerGroupStatusProvider interface {
	GetConsumerGroupStatus() map[string]interface{}
}

type eventProcessorKafkaConsumerGroupMetricsProvider interface {
	GetConsumerGroupMetrics() map[string]int64
}

type eventProcessorProcessorHealthProvider interface {
	Health() *core.HealthStatus
	GetProcessedCount() int64
	GetFailedCount() int64
	GetDuplicateCount() int64
}

type eventProcessorProcessorSharedRuntimeShadowProvider interface {
	SharedRuntimeShadowSnapshot() eventProcessorSharedRuntimeShadowSnapshot
}

type eventProcessorConsumeRuntimeProvider interface {
	Snapshot() eventProcessorConsumeLoopSnapshot
}

func buildEventProcessorRuntimeRolloutHealthHandler(
	ctx context.Context,
	instanceID string,
	logger core.Logger,
	metrics core.MetricsCollector,
	dbManager database.DatabaseManager,
	eventStore eventProcessorComponentHealthProvider,
	metadataStore eventProcessorComponentHealthProvider,
	kafkaHealth eventProcessorKafkaHealthProvider,
	processorRuntime eventProcessorProcessorHealthProvider,
	consumeRuntime eventProcessorConsumeRuntimeProvider,
) (*api.HealthCheckHandler, error) {
	healthHandler := api.NewHealthCheckHandler(dbManager, nil, logger, metrics)
	if err := healthHandler.Initialize(ctx); err != nil {
		return nil, err
	}

	healthHandler.SetRuntimeComponentProvider(func(ctx context.Context) *api.ComponentStatus {
		state := buildEventProcessorRuntimeRolloutState(ctx, dbManager, eventStore, metadataStore, kafkaHealth, processorRuntime, consumeRuntime)
		return buildEventProcessorRuntimeComponentStatus(state, time.Now())
	})
	healthHandler.SetReadinessDetailsProvider(func(ctx context.Context) map[string]interface{} {
		state := buildEventProcessorRuntimeRolloutState(ctx, dbManager, eventStore, metadataStore, kafkaHealth, processorRuntime, consumeRuntime)
		return buildEventProcessorRuntimeReadinessDetails(state)
	})
	healthHandler.SetRolloutReportProducer(newEventProcessorRolloutReportProducer(instanceID, func() eventProcessorRolloutRuntimeState {
		return buildEventProcessorRuntimeRolloutState(context.Background(), dbManager, eventStore, metadataStore, kafkaHealth, processorRuntime, consumeRuntime)
	}))

	return healthHandler, nil
}

func buildEventProcessorRuntimeRolloutState(
	ctx context.Context,
	dbManager database.DatabaseManager,
	eventStore eventProcessorComponentHealthProvider,
	metadataStore eventProcessorComponentHealthProvider,
	kafkaHealth eventProcessorKafkaHealthProvider,
	processorRuntime eventProcessorProcessorHealthProvider,
	consumeRuntime eventProcessorConsumeRuntimeProvider,
) eventProcessorRolloutRuntimeState {
	state := eventProcessorRolloutRuntimeState{
		KafkaReady:            kafkaHealth != nil,
		EventStoreReady:       eventStore != nil,
		MetadataStoreReady:    metadataStore != nil,
		ProcessorRuntimeReady: processorRuntime != nil,
		ConsumeLoopOwned:      consumeRuntime != nil,
	}

	if dbManager != nil {
		state.DatabaseReady = dbManager.CheckMongoHealth(ctx) == nil && dbManager.CheckPostgresHealth(ctx) == nil
	}
	if eventStore != nil {
		state.EventStoreHealthStatus, state.EventStoreHealthMessage = eventProcessorHealthFields(eventStore.Health(ctx))
	}
	if metadataStore != nil {
		state.MetadataStoreHealthStatus, state.MetadataStoreHealthMessage = eventProcessorHealthFields(metadataStore.Health(ctx))
	}
	if kafkaHealth != nil {
		kafkaStatus := kafkaHealth.Health()
		state.KafkaHealthStatus, state.KafkaHealthMessage = eventProcessorHealthFields(kafkaStatus)
		state.KafkaMessageCount, state.KafkaErrorCount = eventProcessorKafkaActivityFields(kafkaStatus)
		state.KafkaActivityState = classifyEventProcessorKafkaActivityState(state.KafkaMessageCount, state.KafkaErrorCount)
		progress := buildEventProcessorKafkaConsumerProgressSnapshot(kafkaHealth, state.KafkaActivityState)
		state.ActiveConsumers = progress.ActiveConsumers
		state.ConsumerLag = progress.Lag
		state.ConsumerLagSeverity = classifyEventProcessorConsumerLagSeverity(progress.Lag)
		state.ConsumerOffset = progress.CurrentOffset
		state.ConsumerProgressState = progress.ProgressState
		state.ConsumerProgressPosture = api.BuildRolloutExecutionProgressPosture(api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
			Consumer: progress,
		})).Consumer
		state.ConsumerBacklogHint = classifyEventProcessorConsumerBacklogHint(
			state.ConsumerProgressPosture,
			state.ConsumerLagSeverity,
		)
	}
	if processorRuntime != nil {
		processorStatus := processorRuntime.Health()
		state.ProcessorHealthStatus, state.ProcessorHealthMessage = eventProcessorHealthFields(processorStatus)
		state.ProcessedEventCount = processorRuntime.GetProcessedCount()
		state.FailedEventCount = processorRuntime.GetFailedCount()
		state.DuplicateEventCount = processorRuntime.GetDuplicateCount()
		if sharedRuntimeProvider, ok := processorRuntime.(eventProcessorProcessorSharedRuntimeShadowProvider); ok {
			shared := sharedRuntimeProvider.SharedRuntimeShadowSnapshot()
			state.SharedRuntimeShadowEnabled = shared.Enabled
			state.SharedRuntimeChainCount = shared.RuntimeCount
			state.SharedRuntimeProcessed = shared.ProcessedEvents
			state.SharedRuntimeSkipped = shared.SkippedDuplicates
			state.SharedRuntimeFailures = shared.RoutedFailures
			state.SharedRuntimeLastChain = shared.LastCheckpointChain
			state.SharedRuntimeLastCursor = shared.LastCheckpointCursor
			state.SharedRuntimeLastBlock = shared.LastCheckpointBlock
			state.SharedRuntimeLastError = shared.LastError
		}
	}
	if consumeRuntime != nil {
		snapshot := consumeRuntime.Snapshot()
		state.ConfiguredConsumeTopics = snapshot.ConfiguredTopics
		state.ActiveConsumeTopics = snapshot.ActiveTopics
		state.ConsumeLoopLastError = snapshot.LastError
		switch {
		case snapshot.Paused:
			state.ConsumeLoopStatus = "paused"
		case snapshot.Running && snapshot.ActiveTopics > 0:
			state.ConsumeLoopStatus = "active"
		case snapshot.Running && snapshot.ConfiguredTopics > 0:
			state.ConsumeLoopStatus = "configured"
		case snapshot.ConfiguredTopics > 0:
			state.ConsumeLoopStatus = "idle"
		default:
			state.ConsumeLoopStatus = "unavailable"
		}
	}

	return state
}

func buildEventProcessorRuntimeComponentStatus(runtimeState eventProcessorRolloutRuntimeState, now time.Time) *api.ComponentStatus {
	completeness := classifyEventProcessorRolloutWiringCompleteness(runtimeState)

	status := "degraded"
	switch completeness.AdvisoryStatus {
	case "runtime-wired":
		status = "healthy"
	case "runtime-wired-unhealthy":
		status = "unhealthy"
	}

	return &api.ComponentStatus{
		Name:      "Indexing Runtime",
		Status:    status,
		Timestamp: now.Unix(),
		Details:   buildEventProcessorRuntimeReadinessDetails(runtimeState),
	}
}

func buildEventProcessorRuntimeReadinessDetails(runtimeState eventProcessorRolloutRuntimeState) map[string]interface{} {
	completeness := classifyEventProcessorRolloutWiringCompleteness(runtimeState)

	details := map[string]interface{}{
		"runtime_mode":              completeness.Mode,
		"rollout_ready":             completeness.AdvisoryReady,
		"rollout_status":            completeness.AdvisoryStatus,
		"rollout_reason":            completeness.Reason,
		"rollout_gate_decision":     "hold",
		"rollout_gate_reason":       completeness.Reason,
		"database_ready":            runtimeState.DatabaseReady,
		"event_store_ready":         runtimeState.EventStoreReady,
		"metadata_store_ready":      runtimeState.MetadataStoreReady,
		"kafka_ready":               runtimeState.KafkaReady,
		"processor_runtime_ready":   runtimeState.ProcessorRuntimeReady,
		"consume_loop_owned":        runtimeState.ConsumeLoopOwned,
		"consume_loop_status":       runtimeState.ConsumeLoopStatus,
		"kafka_activity_state":      runtimeState.KafkaActivityState,
		"consumer_progress_posture": runtimeState.ConsumerProgressPosture,
	}
	if completeness.AdvisoryReady {
		details["rollout_gate_decision"] = "allow"
	}
	if runtimeState.ConsumerLag > 0 {
		details["consumer_lag"] = runtimeState.ConsumerLag
	}
	if runtimeState.ConsumerLagSeverity != "" {
		details["consumer_lag_severity"] = runtimeState.ConsumerLagSeverity
	}
	if runtimeState.ConsumerBacklogHint != "" {
		details["consumer_backlog_hint"] = runtimeState.ConsumerBacklogHint
	}
	if runtimeState.ProcessorHealthStatus != "" {
		details["processor_health_status"] = runtimeState.ProcessorHealthStatus
	}
	if runtimeState.ProcessorHealthMessage != "" {
		details["processor_health_message"] = runtimeState.ProcessorHealthMessage
	}
	details["processor_processed_count"] = runtimeState.ProcessedEventCount
	details["processor_failed_count"] = runtimeState.FailedEventCount
	details["processor_duplicate_count"] = runtimeState.DuplicateEventCount
	details["shared_runtime_shadow_enabled"] = runtimeState.SharedRuntimeShadowEnabled
	details["configured_consume_topics"] = runtimeState.ConfiguredConsumeTopics
	details["active_consume_topics"] = runtimeState.ActiveConsumeTopics
	if runtimeState.SharedRuntimeChainCount > 0 {
		details["shared_runtime_chain_count"] = runtimeState.SharedRuntimeChainCount
	}
	if runtimeState.SharedRuntimeProcessed > 0 {
		details["shared_runtime_processed_count"] = runtimeState.SharedRuntimeProcessed
	}
	if runtimeState.SharedRuntimeSkipped > 0 {
		details["shared_runtime_skipped_duplicates"] = runtimeState.SharedRuntimeSkipped
	}
	if runtimeState.SharedRuntimeFailures > 0 {
		details["shared_runtime_routed_failures"] = runtimeState.SharedRuntimeFailures
	}
	if runtimeState.SharedRuntimeLastChain != "" {
		details["shared_runtime_last_checkpoint_chain"] = runtimeState.SharedRuntimeLastChain
		details["shared_runtime_last_checkpoint_cursor"] = runtimeState.SharedRuntimeLastCursor
		details["shared_runtime_last_checkpoint_block"] = runtimeState.SharedRuntimeLastBlock
	}
	if runtimeState.ConsumeLoopLastError != "" {
		details["consume_loop_last_error"] = runtimeState.ConsumeLoopLastError
	}
	if runtimeState.SharedRuntimeLastError != "" {
		details["shared_runtime_last_error"] = runtimeState.SharedRuntimeLastError
	}

	return details
}

func buildEventProcessorRuntimeSummary(
	runtimeState eventProcessorRolloutRuntimeState,
	metrics core.MetricsCollector,
	now time.Time,
	authEnabled bool,
	rateLimitEnabled bool,
) *eventProcessorRuntimeSummaryResponse {
	completeness := classifyEventProcessorRolloutWiringCompleteness(runtimeState)
	componentStatus := buildEventProcessorRuntimeComponentStatus(runtimeState, now)

	return &eventProcessorRuntimeSummaryResponse{
		Service:        "event-processor",
		Timestamp:      now.Unix(),
		DeploymentMode: "microservice",
		RuntimeMode:    completeness.Mode,
		RuntimePosture: completeness.AdvisoryStatus,
		ComponentState: componentStatus.Status,
		Rollout:        buildEventProcessorRuntimeReadinessDetails(runtimeState),
		Processor:      buildEventProcessorProcessorSummary(runtimeState),
		Security:       buildEventProcessorSecurityRuntimeSection(authEnabled, rateLimitEnabled),
		Metrics:        buildEventProcessorMetricsSummary(metrics),
	}
}

func buildEventProcessorSecurityRuntimeSection(authEnabled, rateLimitEnabled bool) map[string]interface{} {
	authPosture := classifyEventProcessorAuthPosture(authEnabled)
	rateLimitPosture := classifyEventProcessorRateLimitPosture(rateLimitEnabled)
	securityPosture := classifyEventProcessorSecurityPosture(authEnabled, rateLimitEnabled)

	return map[string]interface{}{
		"route_boundary":     "runtime-entrypoint",
		"auth_enabled":       authEnabled,
		"rate_limit_enabled": rateLimitEnabled,
		"auth_posture":       authPosture,
		"rate_limit_posture": rateLimitPosture,
		"security_posture":   securityPosture,
		"security_hint":      classifyEventProcessorSecurityHint(authEnabled, rateLimitEnabled),
		"runtime_boundary":   "optional-security-surface",
	}
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyEventProcessorAuthPosture(enabled bool) string {
	if !enabled {
		return "event-processor-auth-unconfigured"
	}
	return "event-processor-auth-ready"
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyEventProcessorRateLimitPosture(enabled bool) string {
	if !enabled {
		return "event-processor-rate-limit-unconfigured"
	}
	return "event-processor-rate-limit-ready"
}

func classifyEventProcessorSecurityPosture(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "event-processor-security-unconfigured"
	case authEnabled && rateLimitEnabled:
		return "event-processor-security-ready"
	default:
		return "event-processor-security-partial"
	}
}

func classifyEventProcessorSecurityHint(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "event-processor security controls are disabled by default; enable auth or rate limiting explicitly before exposing the runtime/control surface"
	case authEnabled && rateLimitEnabled:
		return "event-processor security controls are aligned for the current runtime/control baseline"
	case authEnabled || rateLimitEnabled:
		return "event-processor security controls are partially enabled; verify the remaining control surface before treating the runtime/control surface as hardened"
	default:
		return "event-processor security controls are partially configured; verify auth and rate limit wiring"
	}
}

func buildEventProcessorProcessorSummary(runtimeState eventProcessorRolloutRuntimeState) map[string]interface{} {
	summary := map[string]interface{}{
		"runtime_ready":                 runtimeState.ProcessorRuntimeReady,
		"health_status":                 runtimeState.ProcessorHealthStatus,
		"health_message":                runtimeState.ProcessorHealthMessage,
		"processed_count":               runtimeState.ProcessedEventCount,
		"failed_count":                  runtimeState.FailedEventCount,
		"duplicate_count":               runtimeState.DuplicateEventCount,
		"execution_boundary":            "lifecycle-only",
		"consume_loop_owned":            runtimeState.ConsumeLoopOwned,
		"consume_loop_status":           runtimeState.ConsumeLoopStatus,
		"shared_runtime_shadow_enabled": runtimeState.SharedRuntimeShadowEnabled,
	}
	if runtimeState.ConsumeLoopOwned {
		summary["execution_boundary"] = "consume-process-seam"
	}
	if runtimeState.SharedRuntimeShadowEnabled {
		summary["execution_boundary"] = "consume-process-shared-runtime-shadow"
	}
	if runtimeState.ProcessorHealthStatus == "" {
		summary["health_status"] = "unavailable"
	}
	if runtimeState.ProcessorHealthMessage == "" {
		summary["health_message"] = "processor runtime not wired"
	}
	if runtimeState.ConfiguredConsumeTopics > 0 {
		summary["configured_consume_topics"] = runtimeState.ConfiguredConsumeTopics
	}
	if runtimeState.ActiveConsumeTopics > 0 {
		summary["active_consume_topics"] = runtimeState.ActiveConsumeTopics
	}
	if runtimeState.ConsumeLoopLastError != "" {
		summary["consume_loop_last_error"] = runtimeState.ConsumeLoopLastError
	}
	if runtimeState.SharedRuntimeChainCount > 0 {
		summary["shared_runtime_chain_count"] = runtimeState.SharedRuntimeChainCount
	}
	if runtimeState.SharedRuntimeProcessed > 0 {
		summary["shared_runtime_processed_count"] = runtimeState.SharedRuntimeProcessed
	}
	if runtimeState.SharedRuntimeSkipped > 0 {
		summary["shared_runtime_skipped_duplicates"] = runtimeState.SharedRuntimeSkipped
	}
	if runtimeState.SharedRuntimeFailures > 0 {
		summary["shared_runtime_routed_failures"] = runtimeState.SharedRuntimeFailures
	}
	if runtimeState.SharedRuntimeLastChain != "" {
		summary["shared_runtime_last_checkpoint_chain"] = runtimeState.SharedRuntimeLastChain
		summary["shared_runtime_last_checkpoint_cursor"] = runtimeState.SharedRuntimeLastCursor
		summary["shared_runtime_last_checkpoint_block"] = runtimeState.SharedRuntimeLastBlock
	}
	if runtimeState.SharedRuntimeLastError != "" {
		summary["shared_runtime_last_error"] = runtimeState.SharedRuntimeLastError
	}
	return summary
}

func buildEventProcessorMetricsSummary(metrics core.MetricsCollector) map[string]interface{} {
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
	summary["counter_count"] = eventProcessorMetricsSectionCount(exported, "counters")
	summary["gauge_count"] = eventProcessorMetricsSectionCount(exported, "gauges")
	summary["histogram_count"] = eventProcessorMetricsSectionCount(exported, "histograms")
	summary["execution_summary"] = fmt.Sprintf(
		"counters=%d gauges=%d histograms=%d",
		summary["counter_count"],
		summary["gauge_count"],
		summary["histogram_count"],
	)

	return summary
}

func eventProcessorMetricsSectionCount(exported map[string]interface{}, section string) int {
	if exported == nil {
		return 0
	}
	values, ok := exported[section].(map[string]interface{})
	if !ok {
		return 0
	}
	return len(values)
}

func eventProcessorHealthFields(status *core.HealthStatus) (string, string) {
	if status == nil {
		return "", ""
	}
	return status.Status, status.Message
}

func eventProcessorKafkaActivityFields(status *core.HealthStatus) (int64, int64) {
	if status == nil || status.Details == nil {
		return 0, 0
	}

	return eventProcessorInt64Detail(status.Details, "message_count"),
		eventProcessorInt64Detail(status.Details, "error_count")
}

func classifyEventProcessorKafkaActivityState(messageCount, errorCount int64) string {
	if messageCount > 0 || errorCount > 0 {
		return "active"
	}
	return "stalled"
}

func eventProcessorInt64Detail(details map[string]interface{}, key string) int64 {
	if details == nil {
		return 0
	}

	switch value := details[key].(type) {
	case int:
		return int64(value)
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func eventProcessorInt64FromInterface(details map[string]interface{}, key string) int64 {
	return eventProcessorInt64Detail(details, key)
}

func eventProcessorInt64Metric(details map[string]int64, key string) int64 {
	if details == nil {
		return 0
	}
	return details[key]
}
