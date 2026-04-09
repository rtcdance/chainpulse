package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"chainpulse/pkg/application/bootstrap"
	appindexing "chainpulse/pkg/application/indexing"
	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

type replayRouteRuntimeStub struct {
	replayed int
	err      error
	status   appindexing.RuntimeStatus
	last     struct {
		chainID string
		from    appindexing.Checkpoint
		to      appindexing.Checkpoint
		limit   int
	}
}

func (s *replayRouteRuntimeStub) ReplayChainRange(
	ctx context.Context,
	chainID string,
	from, to appindexing.Checkpoint,
	limit int,
) (int, error) {
	_ = ctx
	s.last.chainID = chainID
	s.last.from = from
	s.last.to = to
	s.last.limit = limit
	if s.err != nil {
		return 0, s.err
	}
	return s.replayed, nil
}

func (s *replayRouteRuntimeStub) Status() appindexing.RuntimeStatus {
	return s.status
}

type replayRouteFlakySink struct {
	failuresRemaining int
}

func (s *replayRouteFlakySink) Persist(ctx context.Context, events []appindexing.EventEnvelope) error {
	_ = ctx
	_ = events
	if s.failuresRemaining > 0 {
		s.failuresRemaining--
		return fmt.Errorf("persist failed")
	}
	return nil
}

func TestHandleMonolithicDLQReplaySuccess(t *testing.T) {
	runtime := &replayRouteRuntimeStub{
		replayed: 3,
		status:   appindexing.RuntimeStatus{State: "running"},
	}

	req := httptest.NewRequest(http.MethodPost, "/runtime/indexing/dlq/replay", bytes.NewBufferString(`{
		"chain_id":"ethereum",
		"from":{"block_number":10,"cursor":"10:0"},
		"to":{"block_number":12,"cursor":"12:0"},
		"limit":25
	}`))
	rec := httptest.NewRecorder()

	handleMonolithicDLQReplay(rec, req, runtime)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if runtime.last.chainID != "ethereum" || runtime.last.from.BlockNumber != 10 || runtime.last.to.BlockNumber != 12 || runtime.last.limit != 25 {
		t.Fatalf("unexpected replay request forwarded to runtime: %+v", runtime.last)
	}

	var payload dlqReplayResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal replay response: %v", err)
	}
	if payload.Replayed != 3 || payload.ChainID != "ethereum" || payload.RuntimeState != "running" {
		t.Fatalf("unexpected replay payload: %+v", payload)
	}
}

func TestHandleMonolithicDLQReplayRejectsInvalidRange(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/runtime/indexing/dlq/replay", bytes.NewBufferString(`{
		"chain_id":"ethereum",
		"from":{"block_number":12,"cursor":"12:0"},
		"to":{"block_number":10,"cursor":"10:0"}
	}`))
	rec := httptest.NewRecorder()

	handleMonolithicDLQReplay(rec, req, &replayRouteRuntimeStub{})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMonolithicDLQReplayRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	runtime, err := bootstrap.BuildInMemoryIndexingRuntime(logger, &replayRouteFlakySink{failuresRemaining: 1}, []string{"ethereum"})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}
	if err := runtime.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize runtime: %v", err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start runtime: %v", err)
	}

	err = runtime.ProcessBatch(context.Background(), "ethereum", []appindexing.EventEnvelope{
		{EventKey: "evt-1", ChainID: "ethereum", BlockNumber: 10, CheckpointCursor: "10:0"},
	})
	if err == nil {
		t.Fatal("expected failed process batch to seed replay journal")
	}

	gateway := api.NewAPIGatewayPlugin(logger, metrics)
	gateway.SetEventQueryHandler(api.NewEventQueryHandler(nil, logger, metrics))
	gateway.SetEventSubscriptionHandler(api.NewEventSubscriptionHandler(nil, logger, metrics))
	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	healthHandler.InitializedForTests()
	gateway.SetHealthCheckHandler(healthHandler)
	gateway.SetRuntimeReplayProvider(newMonolithicDLQReplayHandler(runtime))

	if err := gateway.Initialize(core.Config{}); err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/runtime/indexing/dlq/replay", bytes.NewBufferString(`{
		"chain_id":"ethereum",
		"from":{"block_number":10,"cursor":"10:0"},
		"to":{"block_number":10,"cursor":"10:9"},
		"limit":10
	}`))
	rec := httptest.NewRecorder()
	gateway.HTTPHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	replay, err := runtime.LoadReplayBatch(context.Background(), "ethereum", appindexing.Checkpoint{ChainID: "ethereum"})
	if err != nil {
		t.Fatalf("load replay batch: %v", err)
	}
	if len(replay) != 0 {
		t.Fatalf("expected replay journal to be acknowledged, got %+v", replay)
	}
}
