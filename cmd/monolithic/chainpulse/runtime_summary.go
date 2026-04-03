package main

import (
	"net/http"
	"strconv"
	"time"

	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

type monolithicRuntimeSummaryResponse struct {
	Service        string                 `json:"service"`
	Timestamp      int64                  `json:"timestamp"`
	RuntimeMode    string                 `json:"runtime_mode"`
	RuntimePosture string                 `json:"runtime_posture"`
	ComponentState string                 `json:"component_state"`
	Rollout        map[string]interface{} `json:"rollout"`
	Indexing       map[string]interface{} `json:"indexing"`
	Gateway        map[string]interface{} `json:"gateway"`
	Metrics        map[string]interface{} `json:"metrics"`
}

type monolithicRuntimeSharedStatus interface {
	Status() appindexing.RuntimeStatus
}

type monolithicRuntimeIndexerStatus interface {
	GetStatus() map[string]map[string]interface{}
}

func buildMonolithicRuntimeSummaryProvider(
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
	sharedRuntime monolithicRuntimeSharedStatus,
	indexer monolithicRuntimeIndexerStatus,
) func(*http.Request) interface{} {
	return func(r *http.Request) interface{} {
		_ = r
		return buildMonolithicRuntimeSummaryResponse(metrics, gateway, sharedRuntime, indexer)
	}
}

func buildMonolithicRuntimeSummaryResponse(
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
	sharedRuntime monolithicRuntimeSharedStatus,
	indexer monolithicRuntimeIndexerStatus,
) *monolithicRuntimeSummaryResponse {
	sharedStatus := appindexing.RuntimeStatus{}
	if sharedRuntime != nil {
		sharedStatus = sharedRuntime.Status()
	}

	indexerStatus := map[string]map[string]interface{}{}
	ownership := ownershipSummary{}
	rollout := ownershipRolloutSummarySnapshot{}
	if indexer != nil {
		indexerStatus = indexer.GetStatus()
		ownership = aggregateIndexerOwnership(indexerStatus)
		rollout = buildOwnershipRolloutSummary(indexerStatus)
	}

	gatewayRuntimeRoutes := false
	gatewayDomainBridgeEnabled := false
	gatewayEventQueryEnabled := false
	gatewayEventSubscriptionEnabled := false
	gatewayHealthCheckEnabled := false
	gatewayPosture := "monolithic-gateway-unconfigured"
	gatewayHint := "monolithic gateway runtime routes are not yet configured"
	if gateway != nil {
		gatewayRuntimeRoutes = gateway.IsRuntimeRoutesEnabled()
		gatewayDomainBridgeEnabled = gateway.IsDomainBridgeEnabled()
		gatewayEventQueryEnabled = gateway.IsEventQueryHandlerEnabled()
		gatewayEventSubscriptionEnabled = gateway.IsEventSubscriptionHandlerEnabled()
		gatewayHealthCheckEnabled = gateway.IsHealthCheckHandlerEnabled()
		gatewayPosture = classifyMonolithicGatewayPosture(
			gatewayRuntimeRoutes,
			gatewayDomainBridgeEnabled,
			gatewayEventQueryEnabled,
			gatewayEventSubscriptionEnabled,
			gatewayHealthCheckEnabled,
		)
		gatewayHint = classifyMonolithicGatewayHint(
			gatewayRuntimeRoutes,
			gatewayDomainBridgeEnabled,
			gatewayEventQueryEnabled,
			gatewayEventSubscriptionEnabled,
			gatewayHealthCheckEnabled,
		)
	}

	runtimeMode, runtimePosture, componentState := classifyMonolithicRuntimeLifecycle(sharedStatus, gatewayRuntimeRoutes, ownership)

	return &monolithicRuntimeSummaryResponse{
		Service:        "monolithic",
		Timestamp:      time.Now().Unix(),
		RuntimeMode:    runtimeMode,
		RuntimePosture: runtimePosture,
		ComponentState: componentState,
		Rollout:        rollout.readinessDetails(),
		Indexing: map[string]interface{}{
			"shared_runtime_state":          sharedStatus.State,
			"shared_runtime_initialized":    sharedStatus.Initialized,
			"shared_runtime_started":        sharedStatus.Started,
			"shared_runtime_chains":         append([]string(nil), sharedStatus.Chains...),
			"shared_runtime_chain_count":    len(sharedStatus.Chains),
			"checkpointing_enabled":         sharedStatus.CheckpointingEnabled,
			"idempotency_enabled":           sharedStatus.IdempotencyEnabled,
			"failure_routing_enabled":       sharedStatus.FailureRoutingEnabled,
			"replay_enabled":                sharedStatus.ReplayEnabled,
			"skipped_duplicates":            sharedStatus.SkippedDuplicates,
			"last_checkpoint_chain_id":      sharedStatus.LastCheckpointChainID,
			"last_checkpoint_cursor":        sharedStatus.LastCheckpointCursor,
			"last_checkpoint_block":         sharedStatus.LastCheckpointBlock,
			"ownership_mode":                classifyOwnershipMode(ownership),
			"ownership_chains":              ownership.Chains,
			"shadow_owned_events":           ownership.ShadowOwnedEvents,
			"legacy_owned_events":           ownership.LegacyOwnedEvents,
			"indexing_runtime_posture":      classifyMonolithicIndexingPosture(sharedStatus, gatewayRuntimeRoutes, ownership),
			"indexing_reliability_hint":     classifyMonolithicIndexingHint(sharedStatus, gatewayRuntimeRoutes, ownership),
			"checkpoint_scope":              classifyMonolithicCheckpointScope(sharedStatus),
			"replay_boundary":               classifyMonolithicReplayBoundary(sharedStatus),
			"checkpoint_recovery_readiness": classifyMonolithicCheckpointRecoveryReadiness(sharedStatus),
		},
		Gateway: map[string]interface{}{
			"domain_bridge_enabled":      gatewayDomainBridgeEnabled,
			"event_query_enabled":        gatewayEventQueryEnabled,
			"event_subscription_enabled": gatewayEventSubscriptionEnabled,
			"health_check_enabled":       gatewayHealthCheckEnabled,
			"runtime_routes_enabled":     gatewayRuntimeRoutes,
			"gateway_posture":            gatewayPosture,
			"reliability_hint":           gatewayHint,
		},
		Metrics: buildMonolithicMetricsSummary(metrics),
	}
}

func classifyMonolithicRuntimeLifecycle(
	sharedStatus appindexing.RuntimeStatus,
	runtimeRoutesEnabled bool,
	ownership ownershipSummary,
) (string, string, string) {
	switch {
	case !sharedStatus.Initialized:
		return "monolithic-runtime-uninitialized", "monolithic-runtime-uninitialized", "unavailable"
	case !sharedStatus.Started:
		return "monolithic-runtime-primed", "monolithic-runtime-primed", "degraded"
	case runtimeRoutesEnabled && ownership.Chains > 0:
		return "monolithic-runtime-ready", "monolithic-runtime-ready", "healthy"
	case runtimeRoutesEnabled:
		return "monolithic-runtime-partial", "monolithic-runtime-partial", "degraded"
	default:
		return "monolithic-runtime-partial", "monolithic-runtime-partial", "degraded"
	}
}

func classifyMonolithicGatewayPosture(
	runtimeRoutesEnabled bool,
	domainBridgeEnabled bool,
	eventQueryEnabled bool,
	eventSubscriptionEnabled bool,
	healthCheckEnabled bool,
) string {
	if !runtimeRoutesEnabled {
		return "monolithic-gateway-unconfigured"
	}
	if eventQueryEnabled && eventSubscriptionEnabled && healthCheckEnabled {
		return "monolithic-gateway-ready"
	}
	return "monolithic-gateway-partial"
}

func classifyMonolithicGatewayHint(
	runtimeRoutesEnabled bool,
	domainBridgeEnabled bool,
	eventQueryEnabled bool,
	eventSubscriptionEnabled bool,
	healthCheckEnabled bool,
) string {
	switch {
	case !runtimeRoutesEnabled:
		return "monolithic gateway runtime routes are not yet configured; initialize the gateway wiring before treating the monolith as runnable"
	case eventQueryEnabled && eventSubscriptionEnabled && healthCheckEnabled:
		return "monolithic gateway runtime routes are fully wired for the current runnable baseline"
	default:
		return "monolithic gateway runtime routes are partially wired; verify the shared gateway surface before treating the monolith as fully ready"
	}
}

func classifyMonolithicIndexingPosture(
	sharedStatus appindexing.RuntimeStatus,
	runtimeRoutesEnabled bool,
	ownership ownershipSummary,
) string {
	switch {
	case !sharedStatus.Initialized:
		return "monolithic-indexing-uninitialized"
	case !sharedStatus.Started:
		return "monolithic-indexing-primed"
	case runtimeRoutesEnabled && ownership.Chains > 0:
		return "monolithic-indexing-ready"
	default:
		return "monolithic-indexing-partial"
	}
}

func classifyMonolithicIndexingHint(
	sharedStatus appindexing.RuntimeStatus,
	runtimeRoutesEnabled bool,
	ownership ownershipSummary,
) string {
	switch classifyMonolithicIndexingPosture(sharedStatus, runtimeRoutesEnabled, ownership) {
	case "monolithic-indexing-ready":
		return "shared indexing runtime is started and the monolithic gateway is exposing the runnable surface"
	case "monolithic-indexing-primed":
		return "shared indexing runtime is initialized but not started yet"
	case "monolithic-indexing-uninitialized":
		return "shared indexing runtime has not been initialized yet"
	default:
		return "shared indexing runtime is partially wired; verify chain ownership and gateway runtime routes"
	}
}

func classifyMonolithicCheckpointScope(sharedStatus appindexing.RuntimeStatus) string {
	if sharedStatus.CheckpointingEnabled {
		return "monolithic-inmemory-checkpoint"
	}
	return "checkpoint-unconfigured"
}

func classifyMonolithicReplayBoundary(sharedStatus appindexing.RuntimeStatus) string {
	if sharedStatus.ReplayEnabled {
		return "monolithic-inmemory-failure-replay"
	}
	return "replay-unconfigured"
}

func classifyMonolithicCheckpointRecoveryReadiness(sharedStatus appindexing.RuntimeStatus) string {
	switch {
	case !sharedStatus.CheckpointingEnabled:
		return "checkpoint-recovery-unconfigured"
	case !sharedStatus.ReplayEnabled:
		return "checkpoint-only"
	case !sharedStatus.FailureRoutingEnabled:
		return "replay-partial"
	default:
		return "checkpoint-replay-ready"
	}
}

func buildMonolithicMetricsSummary(metrics core.MetricsCollector) map[string]interface{} {
	summary := map[string]interface{}{
		"collector_state":   "unavailable",
		"counter_count":     0,
		"gauge_count":       0,
		"histogram_count":   0,
		"execution_summary": "collector unavailable",
	}
	if metrics == nil {
		return summary
	}

	exported := metrics.GetMetrics()
	counters, _ := exported["counters"].(map[string]interface{})
	gauges, _ := exported["gauges"].(map[string]interface{})
	histograms, _ := exported["histograms"].(map[string]interface{})

	summary["collector_state"] = "available"
	summary["counter_count"] = len(counters)
	summary["gauge_count"] = len(gauges)
	summary["histogram_count"] = len(histograms)
	summary["execution_summary"] = "counters=" + strconv.Itoa(len(counters)) +
		" gauges=" + strconv.Itoa(len(gauges)) +
		" histograms=" + strconv.Itoa(len(histograms))
	return summary
}
