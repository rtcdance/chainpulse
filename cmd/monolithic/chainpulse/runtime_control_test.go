package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/indexing"

	appindexingadapter "chainpulse/pkg/adapters/indexing"
)

func TestMonolithicRuntimeControlRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	db := appindexingadapter.NewMonolithicMemoryDatabase(logger)
	if err := db.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize db: %v", err)
	}
	if err := db.Start(); err != nil {
		t.Fatalf("start db: %v", err)
	}

	indexer := indexing.NewMultiChainIndexer(logger, nil)
	runtime, err := newMonolithicPullerRuntime(context.Background(), core.Config{}, "http://localhost:8545", []string{"ethereum"}, logger, metrics, db, indexer)
	if err != nil {
		t.Fatalf("new monolithic runtime: %v", err)
	}

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetEventQueryHandler(api.NewEventQueryHandler(nil, logger, metrics))
	gateway.SetEventSubscriptionHandler(api.NewEventSubscriptionHandler(nil, logger, metrics))
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	gateway.SetHealthCheckHandler(healthHandler)
	gateway.SetRuntimeSummaryProvider(func(r *http.Request) interface{} {
		return map[string]interface{}{"service": "monolithic"}
	})
	gateway.SetRuntimeControlProvider(runtime.HandleRuntimeControl)

	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	handler := gateway.HTTPHandler()
	if handler == nil {
		t.Fatal("expected gateway HTTP handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/runtime/control", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload api.RuntimeControlEnvelopeWithTarget
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal runtime control: %v", err)
	}

	if err := api.ValidateRuntimeControlEnvelopeWithTarget(payload, "monolithic", api.RuntimeControlTargetPollingLoop); err != nil {
		t.Fatalf("validate runtime control envelope: %v", err)
	}
	if payload.Control.State != "idle" {
		t.Fatalf("expected idle control state before runtime start, got %q", payload.Control.State)
	}
}
