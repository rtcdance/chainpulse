package main

import (
	"net/http"
	"strconv"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"

	appindexing "chainpulse/pkg/application/indexing"
)

type monolithicRuntimeSummaryResponse struct {
	Service         string                 `json:"service"`
	Timestamp       int64                  `json:"timestamp"`
	DeploymentMode  string                 `json:"deployment_mode"`
	RuntimeMode     string                 `json:"runtime_mode"`
	RuntimePosture  string                 `json:"runtime_posture"`
	FaultPosture    string                 `json:"fault_posture"`
	ReliabilityHint string                 `json:"reliability_hint"`
	ComponentState  string                 `json:"component_state"`
	Deployment      map[string]interface{} `json:"deployment"`
	Rollout         map[string]interface{} `json:"rollout"`
	Indexing        map[string]interface{} `json:"indexing"`
	Query           map[string]interface{} `json:"query"`
	Puller          map[string]interface{} `json:"puller"`
	Gateway         map[string]interface{} `json:"gateway"`
	Metrics         map[string]interface{} `json:"metrics"`
}

type monolithicRuntimeSharedStatus interface {
	Status() appindexing.RuntimeStatus
}

type monolithicRuntimeIndexerStatus interface {
	GetStatus() map[string]map[string]interface{}
}

type monolithicRuntimeQuerySurface interface {
	summary() map[string]interface{}
}

type monolithicRuntimeReorgSurface interface {
	ReorgStatus() monolithicReorgSummary
}

type monolithicRuntimePullerSurface interface {
	PullerStatus() monolithicPullerSummary
}

type monolithicDeploymentModeSurface interface {
	deploymentSummary() map[string]interface{}
}

func buildMonolithicRuntimeSummaryProvider(
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
	sharedRuntime monolithicRuntimeSharedStatus,
	indexer monolithicRuntimeIndexerStatus,
	reorgRuntime monolithicRuntimeReorgSurface,
	pullerRuntime monolithicRuntimePullerSurface,
	querySurface monolithicRuntimeQuerySurface,
	deploymentMode monolithicDeploymentModeSurface,
) func(*http.Request) interface{} {
	return func(r *http.Request) interface{} {
		_ = r
		return buildMonolithicRuntimeSummaryResponse(metrics, gateway, sharedRuntime, indexer, reorgRuntime, pullerRuntime, querySurface, deploymentMode)
	}
}

func buildMonolithicRuntimeSummaryResponse(
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
	sharedRuntime monolithicRuntimeSharedStatus,
	indexer monolithicRuntimeIndexerStatus,
	reorgRuntime monolithicRuntimeReorgSurface,
	pullerRuntime monolithicRuntimePullerSurface,
	querySurface monolithicRuntimeQuerySurface,
	deploymentMode monolithicDeploymentModeSurface,
) *monolithicRuntimeSummaryResponse {
	//nolint:funlen // Runtime summary builds many fields from various sources.
	sharedStatus := appindexing.RuntimeStatus{}
	if sharedRuntime != nil {
		sharedStatus = sharedRuntime.Status()
	}

	ownership := ownershipSummary{}
	rollout := ownershipRolloutSummarySnapshot{}
	if indexer != nil {
		indexerStatus := indexer.GetStatus()
		ownership = aggregateIndexerOwnership(indexerStatus)
		rollout = buildOwnershipRolloutSummary(indexerStatus)
	}

	gatewayRuntimeRoutes := false
	gatewayDomainBridgeEnabled := false
	gatewayEventQueryEnabled := false
	gatewayEventSubscriptionEnabled := false
	gatewayHealthCheckEnabled := false
	gatewayMetricsRouteEnabled := false
	gatewaySummaryRouteEnabled := false
	gatewayControlRouteEnabled := false
	gatewayReplayRouteEnabled := false
	gatewayBridgeConfiguredCount := 0
	gatewayBridgeAttachedCount := 0
	gatewayBridgeAvailableCount := 0
	gatewayQueryBridgePosture := "query-bridge-unconfigured"
	gatewayQueryBridgeHint := "monolithic gateway query bridge is not configured"
	gatewaySurfaceMode := "gateway-surface-unconfigured"
	gatewaySurfacePosture := "gateway-surface-unconfigured"
	gatewaySurfaceHint := "monolithic gateway surface mode is not yet configured"
	gatewayRegisteredRouteCount := 0
	gatewayRuntimeRouteCount := 0
	gatewayRuntimeSurfaceCount := 0
	gatewayRuntimeSurfacePosture := "monolithic-runtime-surface-unconfigured"
	gatewayRuntimeSurfaceHint := "monolithic gateway runtime route inventory is not yet configured"
	gatewayMethodContractPosture := "monolithic-gateway-method-contract-unconfigured"
	gatewayMethodContractHint := "monolithic gateway route method contract is not yet enforced"
	gatewayPosture := "monolithic-gateway-unconfigured"
	gatewayHint := "monolithic gateway runtime routes are not yet configured"
	if gateway != nil {
		gatewayRuntimeRoutes = gateway.IsRuntimeRoutesEnabled()
		gatewayDomainBridgeEnabled = gateway.IsDomainBridgeEnabled()
		gatewayEventQueryEnabled = gateway.IsEventQueryHandlerEnabled()
		gatewayEventSubscriptionEnabled = gateway.IsEventSubscriptionHandlerEnabled()
		gatewayHealthCheckEnabled = gateway.IsHealthCheckHandlerEnabled()
		gatewayMetricsRouteEnabled = gateway.IsMetricsRouteEnabled()
		gateway.RefreshUpstreamQueryBridgeHealth()
		gatewayBridgeConfiguredCount, gatewayBridgeAttachedCount, gatewayBridgeAvailableCount = gateway.GetUpstreamQueryBridgeStatus()
		gatewayQueryBridgePosture = classifyMonolithicQueryBridgePosture(gatewayBridgeConfiguredCount, gatewayBridgeAttachedCount, gatewayBridgeAvailableCount)
		gatewayQueryBridgeHint = classifyMonolithicQueryBridgeHint(gatewayBridgeConfiguredCount, gatewayBridgeAttachedCount, gatewayBridgeAvailableCount)
		gatewaySurfaceMode = classifyMonolithicGatewaySurfaceMode(gatewayEventQueryEnabled, gatewayEventSubscriptionEnabled, gatewayBridgeConfiguredCount, gatewayBridgeAttachedCount)
		gatewaySurfacePosture = classifyMonolithicGatewaySurfacePosture(gatewayEventQueryEnabled, gatewayEventSubscriptionEnabled, gatewayBridgeConfiguredCount, gatewayBridgeAttachedCount)
		gatewaySurfaceHint = classifyMonolithicGatewaySurfaceHint(gatewayEventQueryEnabled, gatewayEventSubscriptionEnabled, gatewayBridgeConfiguredCount, gatewayBridgeAttachedCount)
		routeInventory := gateway.GetRuntimeRouteInventory()
		gatewayRegisteredRouteCount = routeInventory.RegisteredRouteCount
		gatewayRuntimeRouteCount = routeInventory.RuntimeRouteCount
		gatewayRuntimeSurfaceCount = routeInventory.RuntimeSurfaceCount
		gatewaySummaryRouteEnabled = routeInventory.SummaryRouteEnabled
		gatewayControlRouteEnabled = routeInventory.ControlRouteEnabled
		gatewayReplayRouteEnabled = routeInventory.ReplayRouteEnabled
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
		gatewayRuntimeSurfacePosture = classifyMonolithicRuntimeSurfacePosture(routeInventory)
		gatewayRuntimeSurfaceHint = classifyMonolithicRuntimeSurfaceHint(routeInventory)
		gatewayMethodContractPosture = classifyMonolithicGatewayMethodContractPosture(routeInventory)
		gatewayMethodContractHint = classifyMonolithicGatewayMethodContractHint(routeInventory)
	}

	querySummary := map[string]interface{}{
		"query_alignment_posture": "monolithic-query-unaligned",
		"query_reliability_hint":  "monolithic query surface is still using legacy shared runtime wiring",
	}
	if querySurface != nil {
		querySummary = querySurface.summary()
	}
	reorgSummary := monolithicReorgSummary{
		Posture: "monolithic-reorg-unwired",
		Hint:    "monolithic reorg rollback is not wired",
	}
	if reorgRuntime != nil {
		reorgSummary = reorgRuntime.ReorgStatus()
	}
	pullerSummary := monolithicPullerSummary{
		Posture:        "monolithic-puller-unconfigured",
		Hint:           "monolithic puller runtime is not configured",
		ControlTarget:  api.RuntimeControlTargetPollingLoop,
		ControlPosture: "monolithic-puller-control-unconfigured",
		ControlHint:    "monolithic puller control surface is not configured",
		Control: api.RuntimeControlCore{
			Paused: false,
			State:  "idle",
			Reason: "monolithic puller control surface is not configured",
		},
	}
	if pullerRuntime != nil {
		pullerSummary = pullerRuntime.PullerStatus()
	}
	deploymentSummary := map[string]interface{}{
		"deployment_mode":            deploymentModeMonolithic,
		"deployment_posture":         "deployment-mode-monolithic",
		"reliability_hint":           "monolithic cmd wiring is running in its expected deployment mode baseline",
		"adapter_profile":            "monolithic-runtime-profile",
		"adapter_selection_posture":  "adapter-profile-ready",
		"indexing_storage_adapter":   "monolithic-memory-indexing-storage",
		"query_runtime_adapter":      "indexing-backed-query-surface",
		"transport_adapter_boundary": "monolithic-in-process-runtime",
	}
	if deploymentMode != nil {
		deploymentSummary = deploymentMode.deploymentSummary()
	}
	transportBoundary := classifyMonolithicTransportBoundary(
		stringValue(deploymentSummary["transport_adapter_boundary"], "monolithic-in-process-runtime"),
		gatewaySurfaceMode,
		gatewayBridgeConfiguredCount,
		gatewayBridgeAttachedCount,
		gatewayBridgeAvailableCount,
	)
	deploymentSummary["transport_boundary_posture"] = transportBoundary.Posture
	deploymentSummary["transport_boundary_hint"] = transportBoundary.Hint
	recoveryPosture := classifyMonolithicRecoveryPosture(sharedStatus)
	runtimeMode, runtimePosture, faultPosture, reliabilityHint, componentState := classifyMonolithicRuntimeLifecycle(
		sharedStatus,
		gatewayRuntimeRoutes,
		ownership,
		pullerSummary.Posture,
		recoveryPosture,
		reorgSummary.Posture,
	)

	return &monolithicRuntimeSummaryResponse{
		Service:         "monolithic",
		Timestamp:       time.Now().Unix(),
		DeploymentMode:  stringValue(deploymentSummary["deployment_mode"], deploymentModeMonolithic),
		RuntimeMode:     runtimeMode,
		RuntimePosture:  runtimePosture,
		FaultPosture:    faultPosture,
		ReliabilityHint: reliabilityHint,
		ComponentState:  componentState,
		Deployment:      deploymentSummary,
		Rollout:         rollout.readinessDetails(),
		Indexing: map[string]interface{}{
			"shared_runtime_state":           sharedStatus.State,
			"shared_runtime_initialized":     sharedStatus.Initialized,
			"shared_runtime_started":         sharedStatus.Started,
			"shared_runtime_chains":          append([]string(nil), sharedStatus.Chains...),
			"shared_runtime_chain_count":     len(sharedStatus.Chains),
			"checkpointing_enabled":          sharedStatus.CheckpointingEnabled,
			"idempotency_enabled":            sharedStatus.IdempotencyEnabled,
			"failure_routing_enabled":        sharedStatus.FailureRoutingEnabled,
			"replay_enabled":                 sharedStatus.ReplayEnabled,
			"skipped_duplicates":             sharedStatus.SkippedDuplicates,
			"last_checkpoint_chain_id":       sharedStatus.LastCheckpointChainID,
			"last_checkpoint_cursor":         sharedStatus.LastCheckpointCursor,
			"last_checkpoint_block":          sharedStatus.LastCheckpointBlock,
			"ownership_mode":                 classifyOwnershipMode(ownership),
			"ownership_chains":               ownership.Chains,
			"shadow_owned_events":            ownership.ShadowOwnedEvents,
			"legacy_owned_events":            ownership.LegacyOwnedEvents,
			"indexing_runtime_posture":       classifyMonolithicIndexingPosture(sharedStatus, gatewayRuntimeRoutes, ownership),
			"indexing_reliability_hint":      classifyMonolithicIndexingHint(sharedStatus, gatewayRuntimeRoutes, ownership),
			"checkpoint_scope":               classifyMonolithicCheckpointScope(sharedStatus),
			"replay_boundary":                classifyMonolithicReplayBoundary(sharedStatus),
			"checkpoint_recovery_readiness":  classifyMonolithicCheckpointRecoveryReadiness(sharedStatus),
			"recovery_state":                 sharedStatus.RecoveryState,
			"recovery_run_total":             sharedStatus.RecoveryRuns,
			"recovery_failure_total":         sharedStatus.RecoveryFailures,
			"recovery_checkpoint_load_total": sharedStatus.RecoveryCheckpointLoads,
			"recovery_replayed_events":       sharedStatus.RecoveryReplayedEvents,
			"last_recovery_chain_id":         sharedStatus.LastRecoveryChainID,
			"last_recovery_cursor":           sharedStatus.LastRecoveryCursor,
			"last_recovery_block":            sharedStatus.LastRecoveryBlock,
			"last_recovery_replay_count":     sharedStatus.LastRecoveryReplayCount,
			"last_recovery_error":            sharedStatus.LastRecoveryError,
			"recovery_posture":               recoveryPosture,
			"recovery_reliability_hint":      classifyMonolithicRecoveryHint(sharedStatus),
			"reorg_wiring_enabled":           reorgSummary.Wired,
			"reorg_chain_count":              reorgSummary.ChainCount,
			"reorg_detected_total":           reorgSummary.DetectedTotal,
			"reorg_handled_total":            reorgSummary.HandledTotal,
			"reorg_last_detected_chain_id":   reorgSummary.LastDetectedChainID,
			"reorg_last_detected_block":      reorgSummary.LastDetectedBlock,
			"reorg_last_handled_chain_id":    reorgSummary.LastHandledChainID,
			"reorg_last_handled_block":       reorgSummary.LastHandledBlock,
			"reorg_last_error":               reorgSummary.LastError,
			"reorg_posture":                  reorgSummary.Posture,
			"reorg_reliability_hint":         reorgSummary.Hint,
		},
		Query: querySummary,
		Puller: map[string]interface{}{
			"puller_count":        pullerSummary.PullerCount,
			"active_puller_count": pullerSummary.ActivePullers,
			"backing_off_chains":  pullerSummary.BackingOffChains,
			"poll_request_count":  pullerSummary.RequestCount,
			"poll_error_count":    pullerSummary.ErrorCount,
			"loop_restart_total":  pullerSummary.LoopRestartTotal,
			"loop_failure_total":  pullerSummary.LoopFailureTotal,
			"last_backoff_ms":     pullerSummary.LastBackoffMS,
			"last_error_chain_id": pullerSummary.LastErrorChainID,
			"last_error":          pullerSummary.LastError,
			"puller_posture":      pullerSummary.Posture,
			"reliability_hint":    pullerSummary.Hint,
			"control_target":      pullerSummary.ControlTarget,
			"control_posture":     pullerSummary.ControlPosture,
			"control_hint":        pullerSummary.ControlHint,
			"control_state":       pullerSummary.Control.State,
			"control_paused":      pullerSummary.Control.Paused,
		},
		Gateway: map[string]interface{}{
			"domain_bridge_enabled":           gatewayDomainBridgeEnabled,
			"event_query_enabled":             gatewayEventQueryEnabled,
			"event_subscription_enabled":      gatewayEventSubscriptionEnabled,
			"health_check_enabled":            gatewayHealthCheckEnabled,
			"runtime_summary_enabled":         gatewaySummaryRouteEnabled,
			"metrics_route_enabled":           gatewayMetricsRouteEnabled,
			"runtime_control_enabled":         gatewayControlRouteEnabled,
			"runtime_replay_enabled":          gatewayReplayRouteEnabled,
			"runtime_routes_enabled":          gatewayRuntimeRoutes,
			"upstream_query_configured_count": gatewayBridgeConfiguredCount,
			"upstream_query_attached_count":   gatewayBridgeAttachedCount,
			"upstream_query_available_count":  gatewayBridgeAvailableCount,
			"query_bridge_posture":            gatewayQueryBridgePosture,
			"query_bridge_hint":               gatewayQueryBridgeHint,
			"gateway_surface_mode":            gatewaySurfaceMode,
			"gateway_surface_posture":         gatewaySurfacePosture,
			"gateway_surface_hint":            gatewaySurfaceHint,
			"registered_route_count":          gatewayRegisteredRouteCount,
			"runtime_route_count":             gatewayRuntimeRouteCount,
			"runtime_surface_count":           gatewayRuntimeSurfaceCount,
			"runtime_surface_posture":         gatewayRuntimeSurfacePosture,
			"runtime_surface_hint":            gatewayRuntimeSurfaceHint,
			"method_contract_posture":         gatewayMethodContractPosture,
			"method_contract_hint":            gatewayMethodContractHint,
			"gateway_posture":                 gatewayPosture,
			"reliability_hint":                gatewayHint,
		},
		Metrics: buildMonolithicMetricsSummary(metrics),
	}
}

func classifyMonolithicRuntimeLifecycle(
	sharedStatus appindexing.RuntimeStatus,
	runtimeRoutesEnabled bool,
	ownership ownershipSummary,
	pullerPosture string,
	recoveryPosture string,
	reorgPosture string,
) (string, string, string, string, string) {
	switch {
	case !sharedStatus.Initialized:
		return "monolithic-runtime-uninitialized", "monolithic-runtime-uninitialized", "runtime-uninitialized", "shared indexing runtime has not been initialized yet", "unavailable"
	case !sharedStatus.Started:
		return "monolithic-runtime-primed", "monolithic-runtime-primed", "runtime-primed", "shared indexing runtime is initialized but not started yet", "degraded"
	case hasMonolithicRuntimeFault(pullerPosture, recoveryPosture, reorgPosture):
		return "monolithic-runtime-degraded", "monolithic-runtime-degraded", "runtime-faulted", classifyMonolithicRuntimeFaultHint(pullerPosture, recoveryPosture, reorgPosture), "degraded"
	case hasMonolithicRuntimeWatch(pullerPosture, recoveryPosture, reorgPosture):
		return "monolithic-runtime-watch", "monolithic-runtime-watch", "runtime-watch", classifyMonolithicRuntimeWatchHint(pullerPosture, recoveryPosture, reorgPosture), "degraded"
	case runtimeRoutesEnabled && ownership.Chains > 0:
		return "monolithic-runtime-ready", "monolithic-runtime-ready", "runtime-stable", "monolithic runtime surfaces and resilience seams are healthy for the current runnable baseline", "healthy"
	case runtimeRoutesEnabled:
		return "monolithic-runtime-partial", "monolithic-runtime-partial", "runtime-partial", "gateway runtime routes are enabled but chain ownership is still partial", "degraded"
	default:
		return "monolithic-runtime-partial", "monolithic-runtime-partial", "runtime-partial", "monolithic runtime is only partially wired", "degraded"
	}
}

func hasMonolithicRuntimeFault(pullerPosture, recoveryPosture, reorgPosture string) bool {
	return pullerPosture == "monolithic-puller-degraded" ||
		recoveryPosture == "monolithic-recovery-degraded" ||
		reorgPosture == "monolithic-reorg-degraded"
}

func hasMonolithicRuntimeWatch(pullerPosture, recoveryPosture, reorgPosture string) bool {
	return pullerPosture == "monolithic-puller-recovering" ||
		pullerPosture == "monolithic-puller-partial" ||
		recoveryPosture == "monolithic-recovery-replayed" ||
		reorgPosture == "monolithic-reorg-active"
}

func classifyMonolithicRuntimeFaultHint(pullerPosture, recoveryPosture, reorgPosture string) string {
	switch {
	case recoveryPosture == "monolithic-recovery-degraded":
		return "the latest monolithic recovery probe failed; inspect recovery errors before treating the runtime as healthy"
	case reorgPosture == "monolithic-reorg-degraded":
		return "the monolithic reorg rollback seam reported a recent failure; verify rollback health before trusting indexing correctness"
	default:
		return "the monolithic pull runtime has active failures; inspect puller errors and recovery counters before treating the runtime as healthy"
	}
}

func classifyMonolithicRuntimeWatchHint(pullerPosture, recoveryPosture, reorgPosture string) string {
	switch {
	case pullerPosture == "monolithic-puller-recovering":
		return "the monolithic pull runtime is actively recovering failed poll loops with bounded backoff"
	case recoveryPosture == "monolithic-recovery-replayed":
		return "startup recovery replay was applied successfully; watch the runtime until fresh polling fully stabilizes"
	case reorgPosture == "monolithic-reorg-active":
		return "the reorg rollback seam has already handled a rollback; keep the runtime under watch while indexing continues"
	default:
		return "the monolithic runtime is partially healthy but still requires observation"
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

func classifyMonolithicGatewaySurfaceMode(eventQueryEnabled bool, eventSubscriptionEnabled bool, bridgeConfigured int, bridgeAttached int) string {
	switch {
	case eventQueryEnabled && eventSubscriptionEnabled:
		return "full-in-process"
	case !eventQueryEnabled && !eventSubscriptionEnabled && bridgeConfigured > 0 && bridgeAttached > 0:
		return "upstream-query-bridge"
	case !eventQueryEnabled && !eventSubscriptionEnabled:
		return "runtime-operator-only"
	default:
		return "partial-in-process"
	}
}

func classifyMonolithicGatewaySurfacePosture(eventQueryEnabled bool, eventSubscriptionEnabled bool, bridgeConfigured int, bridgeAttached int) string {
	switch {
	case eventQueryEnabled && eventSubscriptionEnabled:
		return "gateway-surface-full"
	case !eventQueryEnabled && !eventSubscriptionEnabled && bridgeConfigured > 0 && bridgeAttached > 0:
		return "gateway-surface-query-bridge"
	case !eventQueryEnabled && !eventSubscriptionEnabled:
		return "gateway-surface-runtime-only"
	default:
		return "gateway-surface-partial"
	}
}

func classifyMonolithicGatewaySurfaceHint(eventQueryEnabled bool, eventSubscriptionEnabled bool, bridgeConfigured int, bridgeAttached int) string {
	switch {
	case eventQueryEnabled && eventSubscriptionEnabled:
		return "monolithic gateway currently owns both query and subscription surfaces in-process"
	case !eventQueryEnabled && !eventSubscriptionEnabled && bridgeConfigured > 0 && bridgeAttached > 0:
		return "monolithic gateway is exposing a read-only upstream query bridge while keeping subscriptions and in-process business ownership intentionally withheld under microservice deployment intent"
	case !eventQueryEnabled && !eventSubscriptionEnabled:
		return "monolithic gateway currently exposes runtime and health surfaces only; query ownership is intentionally withheld under microservice deployment intent"
	default:
		return "monolithic gateway exposure is partially wired; verify which business routes are intentionally owned before treating the API surface as complete"
	}
}

func classifyMonolithicQueryBridgePosture(configured int, attached int, available int) string {
	switch {
	case configured == 0:
		return "query-bridge-unconfigured"
	case attached == 0:
		return "query-bridge-unattached"
	case available == 0:
		return "query-bridge-unavailable"
	case available < attached:
		return "query-bridge-degraded"
	default:
		return "query-bridge-ready"
	}
}

func classifyMonolithicQueryBridgeHint(configured int, attached int, available int) string {
	switch classifyMonolithicQueryBridgePosture(configured, attached, available) {
	case "query-bridge-ready":
		return "monolithic gateway can forward read-only query traffic to configured api-service upstreams"
	case "query-bridge-degraded":
		return "monolithic gateway has attached upstream query handlers, but only part of the bridge is currently healthy"
	case "query-bridge-unavailable":
		return "monolithic gateway has upstream query handlers attached, but none are currently healthy"
	case "query-bridge-unattached":
		return "monolithic gateway has query upstream endpoints configured, but the bridge has not attached them to query routes yet"
	default:
		return "monolithic gateway query bridge is not configured"
	}
}

func classifyMonolithicRuntimeSurfacePosture(inventory api.GatewayRuntimeRouteInventory) string {
	switch {
	case inventory.RuntimeSurfaceCount == 0:
		return "monolithic-runtime-surface-unconfigured"
	case inventory.HealthRoutesEnabled && inventory.SummaryRouteEnabled && inventory.MetricsRouteEnabled && inventory.ControlRouteEnabled:
		return "monolithic-runtime-surface-ready"
	case inventory.RuntimeSurfaceCount >= 2:
		return "monolithic-runtime-surface-partial"
	default:
		return "monolithic-runtime-surface-minimal"
	}
}

func classifyMonolithicRuntimeSurfaceHint(inventory api.GatewayRuntimeRouteInventory) string {
	switch {
	case inventory.RuntimeSurfaceCount == 0:
		return "monolithic gateway runtime route inventory is not yet configured"
	case inventory.HealthRoutesEnabled && inventory.SummaryRouteEnabled && inventory.MetricsRouteEnabled && inventory.ControlRouteEnabled:
		return "monolithic gateway runtime route inventory now covers health, summary, metrics, and control for the current M1c baseline"
	case inventory.RuntimeSurfaceCount >= 2:
		return "monolithic gateway runtime route inventory is partially wired; keep tightening runtime surfaces before treating observability as complete"
	default:
		return "monolithic gateway runtime route inventory is minimal; add more operator-facing runtime surfaces before treating it as ready"
	}
}

func classifyMonolithicGatewayMethodContractPosture(inventory api.GatewayRuntimeRouteInventory) string {
	switch {
	case inventory.RuntimeSurfaceCount == 0:
		return "monolithic-gateway-method-contract-unconfigured"
	case inventory.SummaryRouteEnabled || inventory.MetricsRouteEnabled || inventory.ControlRouteEnabled:
		return "monolithic-gateway-method-contract-hardened"
	default:
		return "monolithic-gateway-method-contract-minimal"
	}
}

func classifyMonolithicGatewayMethodContractHint(inventory api.GatewayRuntimeRouteInventory) string {
	switch {
	case inventory.RuntimeSurfaceCount == 0:
		return "monolithic gateway route method contract is not yet enforced"
	case inventory.SummaryRouteEnabled || inventory.MetricsRouteEnabled || inventory.ControlRouteEnabled:
		return "monolithic gateway now rejects wrong-method calls on the current runtime routes with an explicit method contract boundary"
	default:
		return "monolithic gateway only exposes a minimal route-method contract surface right now"
	}
}

func stringValue(value interface{}, fallback string) string {
	if typed, ok := value.(string); ok && typed != "" {
		return typed
	}
	return fallback
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

func classifyMonolithicRecoveryPosture(sharedStatus appindexing.RuntimeStatus) string {
	switch {
	case !sharedStatus.CheckpointingEnabled:
		return "monolithic-recovery-unconfigured"
	case sharedStatus.RecoveryState == "recovery-error":
		return "monolithic-recovery-degraded"
	case sharedStatus.RecoveryRuns == 0:
		return "monolithic-recovery-unobserved"
	case sharedStatus.RecoveryReplayedEvents > 0:
		return "monolithic-recovery-replayed"
	case sharedStatus.RecoveryCheckpointLoads > 0:
		return "monolithic-recovery-ready"
	default:
		return "monolithic-recovery-unobserved"
	}
}

func classifyMonolithicRecoveryHint(sharedStatus appindexing.RuntimeStatus) string {
	switch classifyMonolithicRecoveryPosture(sharedStatus) {
	case "monolithic-recovery-unconfigured":
		return "checkpoint and replay recovery ports are not configured for the monolithic indexing runtime"
	case "monolithic-recovery-degraded":
		return "the last monolithic recovery probe failed; inspect the last recovery error before trusting restart readiness"
	case "monolithic-recovery-replayed":
		return "the monolithic indexing runtime has successfully loaded a checkpoint and replayed recovery events during startup"
	case "monolithic-recovery-ready":
		return "the monolithic indexing runtime has successfully loaded startup checkpoints and is ready for additive replay recovery"
	default:
		return "the monolithic indexing runtime has not executed a recovery probe yet"
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
