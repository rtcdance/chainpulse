package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"

	appindexing "chainpulse/pkg/application/indexing"
)

type monolithicRuntimeSummarySharedRuntimeStub struct {
	status appindexing.RuntimeStatus
}

func (s monolithicRuntimeSummarySharedRuntimeStub) Status() appindexing.RuntimeStatus {
	return s.status
}

type monolithicRuntimeSummaryIndexerStub struct {
	status map[string]map[string]interface{}
}

func (s monolithicRuntimeSummaryIndexerStub) GetStatus() map[string]map[string]interface{} {
	return s.status
}

type monolithicRuntimeSummaryQuerySurfaceStub struct {
	data map[string]interface{}
}

func (s monolithicRuntimeSummaryQuerySurfaceStub) summary() map[string]interface{} {
	return s.data
}

type monolithicRuntimeSummaryReorgStub struct {
	status monolithicReorgSummary
}

func (s monolithicRuntimeSummaryReorgStub) ReorgStatus() monolithicReorgSummary {
	return s.status
}

type monolithicRuntimeSummaryPullerStub struct {
	status monolithicPullerSummary
}

func (s monolithicRuntimeSummaryPullerStub) PullerStatus() monolithicPullerSummary {
	return s.status
}

type monolithicRuntimeSummaryDeploymentStub struct {
	data map[string]interface{}
}

func (s monolithicRuntimeSummaryDeploymentStub) deploymentSummary() map[string]interface{} {
	return s.data
}

func TestMonolithicRuntimeSummaryRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetEventQueryHandler(api.NewEventQueryHandler(nil, logger, metrics))
	gateway.SetEventSubscriptionHandler(api.NewEventSubscriptionHandler(nil, logger, metrics))
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	gateway.SetHealthCheckHandler(healthHandler)

	sharedRuntime := monolithicRuntimeSummarySharedRuntimeStub{
		status: appindexing.RuntimeStatus{
			State:                   "running",
			Initialized:             true,
			Started:                 true,
			Chains:                  []string{"ethereum", "polygon"},
			CheckpointingEnabled:    true,
			IdempotencyEnabled:      true,
			FailureRoutingEnabled:   true,
			ReplayEnabled:           true,
			RecoveryState:           "checkpoint-loaded",
			RecoveryRuns:            2,
			RecoveryCheckpointLoads: 2,
			RecoveryReplayedEvents:  0,
			LastRecoveryChainID:     "polygon",
			LastRecoveryCursor:      "21:0",
			LastRecoveryBlock:       21,
			LastRecoveryReplayCount: 0,
			LastCheckpointChainID:   "polygon",
			LastCheckpointCursor:    "22:0",
			LastCheckpointBlock:     22,
		},
	}
	indexer := monolithicRuntimeSummaryIndexerStub{
		status: map[string]map[string]interface{}{
			"ethereum": {
				"shadow_owned_events": int64(4),
				"legacy_owned_events": int64(1),
			},
			"polygon": {
				"shadow_owned_events": int64(2),
				"legacy_owned_events": int64(3),
			},
		},
	}

	querySurface := monolithicRuntimeSummaryQuerySurfaceStub{
		data: map[string]interface{}{
			"query_alignment_posture": "monolithic-query-indexing-aligned",
			"query_reliability_hint":  "query reads are aligned to indexing-backed storage",
		},
	}
	reorgRuntime := monolithicRuntimeSummaryReorgStub{
		status: monolithicReorgSummary{
			Wired:               true,
			ChainCount:          2,
			DetectedTotal:       0,
			HandledTotal:        0,
			LastDetectedChainID: "",
			LastDetectedBlock:   0,
			LastHandledChainID:  "",
			LastHandledBlock:    0,
			Posture:             "monolithic-reorg-armed",
			Hint:                "monolithic reorg rollback is wired with in-memory block snapshots",
		},
	}
	pullerRuntime := monolithicRuntimeSummaryPullerStub{
		status: monolithicPullerSummary{
			PullerCount:      2,
			ActivePullers:    2,
			BackingOffChains: 0,
			RequestCount:     9,
			ErrorCount:       0,
			LoopRestartTotal: 0,
			LoopFailureTotal: 0,
			LastBackoffMS:    0,
			Posture:          "monolithic-puller-healthy",
			Hint:             "monolithic puller runtime is polling all configured chains without recorded errors",
			ControlTarget:    api.RuntimeControlTargetPollingLoop,
			ControlPosture:   "monolithic-puller-read-only-control",
			ControlHint:      "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
			Control: api.RuntimeControlCore{
				Paused: false,
				State:  "running",
				Reason: "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
			},
		},
	}
	deploymentMode := monolithicRuntimeSummaryDeploymentStub{
		data: map[string]interface{}{
			"deployment_mode":            deploymentModeMonolithic,
			"deployment_posture":         "deployment-mode-monolithic",
			"adapter_profile":            "monolithic-runtime-profile",
			"adapter_selection_posture":  "adapter-profile-ready",
			"indexing_storage_adapter":   "monolithic-memory-indexing-storage",
			"query_runtime_adapter":      "indexing-backed-query-surface",
			"transport_adapter_boundary": "monolithic-in-process-runtime",
			"reliability_hint":           "monolithic cmd wiring is running in its expected deployment mode baseline",
		},
	}

	gateway.SetRuntimeSummaryProvider(buildMonolithicRuntimeSummaryProvider(metrics, gateway, sharedRuntime, indexer, reorgRuntime, pullerRuntime, querySurface, deploymentMode))
	gateway.SetRuntimeReplayProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}
	if !gateway.IsRuntimeRoutesEnabled() {
		t.Fatal("expected runtime routes to be enabled")
	}

	handler := gateway.HTTPHandler()
	if handler == nil {
		t.Fatal("expected gateway HTTP handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got := payload["service"]; got != "monolithic" {
		t.Fatalf("expected service monolithic, got %v", got)
	}
	if got := payload["deployment_mode"]; got != deploymentModeMonolithic {
		t.Fatalf("expected deployment_mode monolithic, got %v", got)
	}
	if got := payload["runtime_mode"]; got != "monolithic-runtime-ready" {
		t.Fatalf("expected runtime_mode monolithic-runtime-ready, got %v", got)
	}
	if got := payload["runtime_posture"]; got != "monolithic-runtime-ready" {
		t.Fatalf("expected runtime_posture monolithic-runtime-ready, got %v", got)
	}
	if got := payload["fault_posture"]; got != "runtime-stable" {
		t.Fatalf("expected fault_posture runtime-stable, got %v", got)
	}
	if got := payload["component_state"]; got != "healthy" {
		t.Fatalf("expected component_state healthy, got %v", got)
	}

	indexing, ok := payload["indexing"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected indexing object, got %T", payload["indexing"])
	}
	if got := indexing["shared_runtime_started"]; got != true {
		t.Fatalf("expected shared_runtime_started true, got %v", got)
	}
	if got := indexing["shared_runtime_chain_count"]; got != float64(2) {
		t.Fatalf("expected shared_runtime_chain_count 2, got %v", got)
	}
	if got := indexing["ownership_mode"]; got != "shadow" {
		t.Fatalf("expected ownership_mode shadow, got %v", got)
	}
	if got := indexing["checkpointing_enabled"]; got != true {
		t.Fatalf("expected checkpointing_enabled true, got %v", got)
	}
	if got := indexing["replay_enabled"]; got != true {
		t.Fatalf("expected replay_enabled true, got %v", got)
	}
	if got := indexing["checkpoint_recovery_readiness"]; got != "checkpoint-replay-ready" {
		t.Fatalf("expected checkpoint_recovery_readiness checkpoint-replay-ready, got %v", got)
	}
	if got := indexing["recovery_state"]; got != "checkpoint-loaded" {
		t.Fatalf("expected recovery_state checkpoint-loaded, got %v", got)
	}
	if got := indexing["recovery_run_total"]; got != float64(2) {
		t.Fatalf("expected recovery_run_total 2, got %v", got)
	}
	if got := indexing["recovery_replayed_events"]; got != float64(0) {
		t.Fatalf("expected recovery_replayed_events 0, got %v", got)
	}
	if got := indexing["recovery_posture"]; got != "monolithic-recovery-ready" {
		t.Fatalf("expected recovery_posture monolithic-recovery-ready, got %v", got)
	}
	if got := indexing["reorg_wiring_enabled"]; got != true {
		t.Fatalf("expected reorg_wiring_enabled true, got %v", got)
	}
	if got := indexing["reorg_detected_total"]; got != float64(0) {
		t.Fatalf("expected reorg_detected_total 0, got %v", got)
	}
	if got := indexing["reorg_posture"]; got != "monolithic-reorg-armed" {
		t.Fatalf("expected reorg_posture monolithic-reorg-armed, got %v", got)
	}

	querySummary, ok := payload["query"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected query object, got %T", payload["query"])
	}
	if got := querySummary["query_alignment_posture"]; got != "monolithic-query-indexing-aligned" {
		t.Fatalf("expected query_alignment_posture monolithic-query-indexing-aligned, got %v", got)
	}

	pullerSummary, ok := payload["puller"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected puller object, got %T", payload["puller"])
	}
	if got := pullerSummary["puller_posture"]; got != "monolithic-puller-healthy" {
		t.Fatalf("expected puller_posture monolithic-puller-healthy, got %v", got)
	}
	if got := pullerSummary["loop_restart_total"]; got != float64(0) {
		t.Fatalf("expected loop_restart_total 0, got %v", got)
	}
	if got := pullerSummary["backing_off_chains"]; got != float64(0) {
		t.Fatalf("expected backing_off_chains 0, got %v", got)
	}
	if got := pullerSummary["control_target"]; got != api.RuntimeControlTargetPollingLoop {
		t.Fatalf("expected control_target polling-loop, got %v", got)
	}

	gatewaySummary, ok := payload["gateway"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gateway object, got %T", payload["gateway"])
	}
	if got := gatewaySummary["metrics_route_enabled"]; got != false {
		t.Fatalf("expected metrics_route_enabled false, got %v", got)
	}
	if got := gatewaySummary["runtime_summary_enabled"]; got != true {
		t.Fatalf("expected runtime_summary_enabled true, got %v", got)
	}
	if got := gatewaySummary["runtime_control_enabled"]; got != false {
		t.Fatalf("expected runtime_control_enabled false, got %v", got)
	}
	if got := gatewaySummary["runtime_replay_enabled"]; got != true {
		t.Fatalf("expected runtime_replay_enabled true, got %v", got)
	}
	if got := gatewaySummary["runtime_routes_enabled"]; got != true {
		t.Fatalf("expected runtime_routes_enabled true, got %v", got)
	}
	if got := gatewaySummary["registered_route_count"]; got != float64(18) {
		t.Fatalf("expected registered_route_count 18, got %v", got)
	}
	if got := gatewaySummary["runtime_route_count"]; got != float64(7) {
		t.Fatalf("expected runtime_route_count 7, got %v", got)
	}
	if got := gatewaySummary["runtime_surface_count"]; got != float64(3) {
		t.Fatalf("expected runtime_surface_count 3, got %v", got)
	}
	if got := gatewaySummary["runtime_surface_posture"]; got != "monolithic-runtime-surface-partial" {
		t.Fatalf("expected runtime_surface_posture monolithic-runtime-surface-partial, got %v", got)
	}
	if got := gatewaySummary["method_contract_posture"]; got != "monolithic-gateway-method-contract-hardened" {
		t.Fatalf("expected method_contract_posture monolithic-gateway-method-contract-hardened, got %v", got)
	}
	if got := gatewaySummary["gateway_posture"]; got != "monolithic-gateway-ready" {
		t.Fatalf("expected gateway_posture monolithic-gateway-ready, got %v", got)
	}

	metricsSummary, ok := payload["metrics"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metrics object, got %T", payload["metrics"])
	}
	if got := metricsSummary["collector_state"]; got != "available" {
		t.Fatalf("expected collector_state available, got %v", got)
	}

	deploymentSummary, ok := payload["deployment"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected deployment object, got %T", payload["deployment"])
	}
	if got := deploymentSummary["deployment_posture"]; got != "deployment-mode-monolithic" {
		t.Fatalf("expected deployment_posture deployment-mode-monolithic, got %v", got)
	}
	if got := deploymentSummary["adapter_profile"]; got != "monolithic-runtime-profile" {
		t.Fatalf("expected adapter_profile monolithic-runtime-profile, got %v", got)
	}
	if got := deploymentSummary["query_runtime_adapter"]; got != "indexing-backed-query-surface" {
		t.Fatalf("expected query_runtime_adapter indexing-backed-query-surface, got %v", got)
	}
}

func TestMonolithicRuntimeSummaryRouteReflectsRecoveringRuntime(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetEventQueryHandler(api.NewEventQueryHandler(nil, logger, metrics))
	gateway.SetEventSubscriptionHandler(api.NewEventSubscriptionHandler(nil, logger, metrics))
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	gateway.SetHealthCheckHandler(healthHandler)

	sharedRuntime := monolithicRuntimeSummarySharedRuntimeStub{
		status: appindexing.RuntimeStatus{
			State:                   "running",
			Initialized:             true,
			Started:                 true,
			Chains:                  []string{"ethereum"},
			CheckpointingEnabled:    true,
			IdempotencyEnabled:      true,
			FailureRoutingEnabled:   true,
			ReplayEnabled:           true,
			RecoveryState:           "replay-applied",
			RecoveryRuns:            1,
			RecoveryCheckpointLoads: 1,
			RecoveryReplayedEvents:  2,
			LastRecoveryChainID:     "ethereum",
			LastRecoveryCursor:      "12:0",
			LastRecoveryBlock:       12,
			LastRecoveryReplayCount: 2,
		},
	}
	indexer := monolithicRuntimeSummaryIndexerStub{
		status: map[string]map[string]interface{}{
			"ethereum": {"shadow_owned_events": int64(4)},
		},
	}
	querySurface := monolithicRuntimeSummaryQuerySurfaceStub{
		data: map[string]interface{}{
			"query_alignment_posture": "monolithic-query-indexing-aligned",
			"query_reliability_hint":  "query reads are aligned to indexing-backed storage",
		},
	}
	reorgRuntime := monolithicRuntimeSummaryReorgStub{
		status: monolithicReorgSummary{
			Wired:      true,
			ChainCount: 1,
			Posture:    "monolithic-reorg-armed",
			Hint:       "monolithic reorg rollback is wired with in-memory block snapshots",
		},
	}
	pullerRuntime := monolithicRuntimeSummaryPullerStub{
		status: monolithicPullerSummary{
			PullerCount:      1,
			ActivePullers:    1,
			BackingOffChains: 1,
			RequestCount:     4,
			ErrorCount:       1,
			LoopRestartTotal: 1,
			LoopFailureTotal: 1,
			LastBackoffMS:    250,
			Posture:          "monolithic-puller-recovering",
			Hint:             "monolithic puller runtime is restarting failed poll loops with bounded backoff",
			ControlTarget:    api.RuntimeControlTargetPollingLoop,
			ControlPosture:   "monolithic-puller-read-only-control",
			ControlHint:      "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
			Control: api.RuntimeControlCore{
				Paused: false,
				State:  "running",
				Reason: "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
			},
		},
	}
	deploymentMode := monolithicRuntimeSummaryDeploymentStub{
		data: map[string]interface{}{
			"deployment_mode":            deploymentModeMonolithic,
			"deployment_posture":         "deployment-mode-monolithic",
			"adapter_profile":            "monolithic-runtime-profile",
			"adapter_selection_posture":  "adapter-profile-ready",
			"indexing_storage_adapter":   "monolithic-memory-indexing-storage",
			"query_runtime_adapter":      "indexing-backed-query-surface",
			"transport_adapter_boundary": "monolithic-in-process-runtime",
			"reliability_hint":           "monolithic cmd wiring is running in its expected deployment mode baseline",
		},
	}

	gateway.SetRuntimeSummaryProvider(buildMonolithicRuntimeSummaryProvider(metrics, gateway, sharedRuntime, indexer, reorgRuntime, pullerRuntime, querySurface, deploymentMode))
	gateway.SetRuntimeReplayProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rr := httptest.NewRecorder()
	gateway.HTTPHandler().ServeHTTP(rr, req)

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got := payload["runtime_mode"]; got != "monolithic-runtime-watch" {
		t.Fatalf("expected runtime_mode monolithic-runtime-watch, got %v", got)
	}
	if got := payload["runtime_posture"]; got != "monolithic-runtime-watch" {
		t.Fatalf("expected runtime_posture monolithic-runtime-watch, got %v", got)
	}
	if got := payload["fault_posture"]; got != "runtime-watch" {
		t.Fatalf("expected fault_posture runtime-watch, got %v", got)
	}
	if got := payload["component_state"]; got != "degraded" {
		t.Fatalf("expected component_state degraded, got %v", got)
	}
}

func TestMonolithicRuntimeSummaryRouteReflectsReadyRuntimeSurfaceInventory(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	gateway.SetHealthCheckHandler(healthHandler)

	sharedRuntime := monolithicRuntimeSummarySharedRuntimeStub{
		status: appindexing.RuntimeStatus{
			State:       "running",
			Initialized: true,
			Started:     true,
			Chains:      []string{"ethereum"},
		},
	}
	indexer := monolithicRuntimeSummaryIndexerStub{
		status: map[string]map[string]interface{}{
			"ethereum": {"shadow_owned_events": int64(1)},
		},
	}
	querySurface := monolithicRuntimeSummaryQuerySurfaceStub{
		data: map[string]interface{}{
			"query_alignment_posture": "monolithic-query-managed-runtime",
			"query_runtime_adapter":   "managed-db-runtime-wiring",
			"query_selection_posture": "query-surface-managed-runtime",
			"query_reliability_hint":  "microservice deployment intent currently keeps query reads on the managed-db/shared runtime path",
		},
	}
	reorgRuntime := monolithicRuntimeSummaryReorgStub{
		status: monolithicReorgSummary{
			Wired:      true,
			ChainCount: 1,
			Posture:    "monolithic-reorg-armed",
			Hint:       "monolithic reorg rollback is wired with in-memory block snapshots",
		},
	}
	pullerRuntime := monolithicRuntimeSummaryPullerStub{
		status: monolithicPullerSummary{
			PullerCount:    1,
			ActivePullers:  1,
			Posture:        "monolithic-puller-healthy",
			Hint:           "monolithic puller runtime is polling all configured chains without recorded errors",
			ControlTarget:  api.RuntimeControlTargetPollingLoop,
			ControlPosture: "monolithic-puller-read-only-control",
			ControlHint:    "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
			Control: api.RuntimeControlCore{
				Paused: false,
				State:  "running",
				Reason: "monolithic puller runtime currently exposes read-only control status; use process lifecycle for stop/start",
			},
		},
	}
	deploymentMode := monolithicRuntimeSummaryDeploymentStub{
		data: map[string]interface{}{
			"deployment_mode":            deploymentModeMicroservice,
			"deployment_posture":         "deployment-mode-microservice-intent",
			"adapter_profile":            "microservice-target-profile",
			"adapter_selection_posture":  "adapter-profile-partial",
			"indexing_storage_adapter":   "compatibility-mock-indexing-storage",
			"query_runtime_adapter":      "managed-db-runtime-wiring",
			"transport_adapter_boundary": "runtime-operator-only-gateway-intent",
			"reliability_hint":           "microservice deployment intent is selected, but monolithic cmd wiring still uses partial compatibility adapters until later M2 slices complete",
		},
	}

	gateway.SetRuntimeSummaryProvider(buildMonolithicRuntimeSummaryProvider(metrics, gateway, sharedRuntime, indexer, reorgRuntime, pullerRuntime, querySurface, deploymentMode))
	gateway.SetRuntimeMetricsProvider(buildMonolithicMetricsProvider(metrics, nil))
	gateway.SetRuntimeControlProvider(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	gateway.SetRuntimeReplayProvider(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rr := httptest.NewRecorder()
	gateway.HTTPHandler().ServeHTTP(rr, req)

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	gatewaySummary, ok := payload["gateway"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gateway object, got %T", payload["gateway"])
	}
	if got := gatewaySummary["runtime_summary_enabled"]; got != true {
		t.Fatalf("expected runtime_summary_enabled true, got %v", got)
	}
	if got := gatewaySummary["metrics_route_enabled"]; got != true {
		t.Fatalf("expected metrics_route_enabled true, got %v", got)
	}
	if got := gatewaySummary["runtime_control_enabled"]; got != true {
		t.Fatalf("expected runtime_control_enabled true, got %v", got)
	}
	if got := gatewaySummary["runtime_replay_enabled"]; got != true {
		t.Fatalf("expected runtime_replay_enabled true, got %v", got)
	}
	if got := gatewaySummary["event_query_enabled"]; got != false {
		t.Fatalf("expected event_query_enabled false, got %v", got)
	}
	if got := gatewaySummary["event_subscription_enabled"]; got != false {
		t.Fatalf("expected event_subscription_enabled false, got %v", got)
	}
	if got := gatewaySummary["gateway_surface_mode"]; got != "runtime-operator-only" {
		t.Fatalf("expected gateway_surface_mode runtime-operator-only, got %v", got)
	}
	if got := gatewaySummary["gateway_surface_posture"]; got != "gateway-surface-runtime-only" {
		t.Fatalf("expected gateway_surface_posture gateway-surface-runtime-only, got %v", got)
	}
	if got := gatewaySummary["registered_route_count"]; got != float64(10) {
		t.Fatalf("expected registered_route_count 10, got %v", got)
	}
	if got := gatewaySummary["runtime_route_count"]; got != float64(9) {
		t.Fatalf("expected runtime_route_count 9, got %v", got)
	}
	if got := gatewaySummary["runtime_surface_count"]; got != float64(5) {
		t.Fatalf("expected runtime_surface_count 5, got %v", got)
	}
	if got := gatewaySummary["runtime_surface_posture"]; got != "monolithic-runtime-surface-ready" {
		t.Fatalf("expected runtime_surface_posture monolithic-runtime-surface-ready, got %v", got)
	}
	if got := gatewaySummary["method_contract_posture"]; got != "monolithic-gateway-method-contract-hardened" {
		t.Fatalf("expected method_contract_posture monolithic-gateway-method-contract-hardened, got %v", got)
	}
	if got := payload["deployment_mode"]; got != deploymentModeMicroservice {
		t.Fatalf("expected deployment_mode microservice, got %v", got)
	}
	deploymentSummary, ok := payload["deployment"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected deployment object, got %T", payload["deployment"])
	}
	if got := deploymentSummary["adapter_profile"]; got != "microservice-target-profile" {
		t.Fatalf("expected adapter_profile microservice-target-profile, got %v", got)
	}
	if got := deploymentSummary["adapter_selection_posture"]; got != "adapter-profile-partial" {
		t.Fatalf("expected adapter_selection_posture adapter-profile-partial, got %v", got)
	}
	if got := deploymentSummary["indexing_storage_adapter"]; got != "compatibility-mock-indexing-storage" {
		t.Fatalf("expected indexing_storage_adapter compatibility-mock-indexing-storage, got %v", got)
	}
	if got := deploymentSummary["transport_adapter_boundary"]; got != "runtime-operator-only-gateway-intent" {
		t.Fatalf("expected transport_adapter_boundary runtime-operator-only-gateway-intent, got %v", got)
	}
	if got := deploymentSummary["transport_boundary_posture"]; got != "transport-boundary-runtime-operator-only" {
		t.Fatalf("expected transport_boundary_posture transport-boundary-runtime-operator-only, got %v", got)
	}
}

func TestBuildMonolithicRuntimeSummaryResponseMicroserviceBridgeTransportPosture(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	gateway.SetHealthCheckHandler(healthHandler)
	gateway.SetUpstreamQueryEndpoints([]string{"http://upstream-1:8081", "http://upstream-2:8081"})
	gateway.SetRuntimeSummaryProvider(func(r *http.Request) interface{} { return map[string]interface{}{"ok": true} })
	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	deploymentMode := monolithicRuntimeSummaryDeploymentStub{
		data: map[string]interface{}{
			"deployment_mode":            deploymentModeMicroservice,
			"deployment_posture":         "deployment-mode-microservice-intent",
			"adapter_profile":            "microservice-target-profile",
			"adapter_selection_posture":  "adapter-profile-bridged",
			"indexing_storage_adapter":   "compatibility-mock-indexing-storage",
			"query_runtime_adapter":      "managed-db-runtime-wiring",
			"transport_adapter_boundary": "upstream-query-bridge-gateway-intent",
			"reliability_hint":           "microservice deployment intent is selected and the monolithic gateway now exposes a read-only upstream query bridge while compatibility adapters continue to cover the remaining in-process seams",
		},
	}

	payload := buildMonolithicRuntimeSummaryResponse(
		metrics,
		gateway,
		nil,
		nil,
		nil,
		nil,
		nil,
		deploymentMode,
	)

	deploymentSummary := payload.Deployment
	if got := deploymentSummary["transport_boundary_posture"]; got != "transport-boundary-bridge-unavailable" {
		t.Fatalf("expected transport_boundary_posture transport-boundary-bridge-unavailable, got %v", got)
	}
	if got := deploymentSummary["transport_adapter_boundary"]; got != "upstream-query-bridge-gateway-intent" {
		t.Fatalf("expected upstream-query-bridge-gateway-intent, got %v", got)
	}
}
