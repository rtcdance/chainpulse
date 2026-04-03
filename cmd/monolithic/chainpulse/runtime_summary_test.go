package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
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
			State:                 "running",
			Initialized:           true,
			Started:               true,
			Chains:                []string{"ethereum", "polygon"},
			CheckpointingEnabled:  true,
			IdempotencyEnabled:    true,
			FailureRoutingEnabled: true,
			ReplayEnabled:         true,
			LastCheckpointChainID: "polygon",
			LastCheckpointCursor:  "22:0",
			LastCheckpointBlock:   22,
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

	gateway.SetRuntimeSummaryProvider(buildMonolithicRuntimeSummaryProvider(metrics, gateway, sharedRuntime, indexer))
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
	if got := payload["runtime_mode"]; got != "monolithic-runtime-ready" {
		t.Fatalf("expected runtime_mode monolithic-runtime-ready, got %v", got)
	}
	if got := payload["runtime_posture"]; got != "monolithic-runtime-ready" {
		t.Fatalf("expected runtime_posture monolithic-runtime-ready, got %v", got)
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

	gatewaySummary, ok := payload["gateway"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gateway object, got %T", payload["gateway"])
	}
	if got := gatewaySummary["runtime_routes_enabled"]; got != true {
		t.Fatalf("expected runtime_routes_enabled true, got %v", got)
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
}
