package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

func TestBuildEventProcessorRuntimeHTTPHandlerExposesRolloutRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.SetRolloutReportProducer(newEventProcessorRolloutReportProducer("event-processor-1", func() eventProcessorRolloutRuntimeState {
		return eventProcessorRolloutRuntimeState{
			DatabaseReady:      true,
			EventStoreReady:    true,
			MetadataStoreReady: true,
			KafkaReady:         true,
		}
	}))
	handler.InitializedForTests()

	mux := buildEventProcessorRuntimeHTTPHandler(handler, metrics, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health/rollout", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

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
	if got := payload.Details.Service; got != "event-processor" {
		t.Fatalf("expected service event-processor, got %q", got)
	}
}

func TestBuildEventProcessorRuntimeHTTPHandlerExposesReadyAndComponentsDetails(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(&eventProcessorTestDatabaseManager{
		mongoHealthy:    true,
		postgresHealthy: true,
	}, logger, metrics)
	handler.SetRuntimeComponentProvider(func(ctx context.Context) *api.ComponentStatus {
		return &api.ComponentStatus{
			Name:      "Indexing Runtime",
			Status:    "healthy",
			Timestamp: 1712345678,
			Details: map[string]interface{}{
				"runtime_mode":          "runtime-wired",
				"rollout_gate_decision": "allow",
			},
		}
	})
	handler.SetReadinessDetailsProvider(func(ctx context.Context) map[string]interface{} {
		return map[string]interface{}{
			"runtime_mode":          "runtime-wired",
			"rollout_gate_decision": "allow",
			"rollout_status":        "runtime-wired",
		}
	})
	if err := handler.Initialize(context.Background()); err != nil {
		t.Fatalf("initialize handler: %v", err)
	}

	mux := buildEventProcessorRuntimeHTTPHandler(handler, metrics, nil, nil)

	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyRec := httptest.NewRecorder()
	mux.ServeHTTP(readyRec, readyReq)

	if readyRec.Code != http.StatusOK {
		t.Fatalf("expected ready status 200, got %d body=%s", readyRec.Code, readyRec.Body.String())
	}

	var readyPayload api.ReadinessResponse
	if err := json.Unmarshal(readyRec.Body.Bytes(), &readyPayload); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	if got := readyPayload.Details["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout gate decision allow, got %v", got)
	}

	componentsReq := httptest.NewRequest(http.MethodGet, "/health/components", nil)
	componentsRec := httptest.NewRecorder()
	mux.ServeHTTP(componentsRec, componentsReq)

	if componentsRec.Code != http.StatusOK {
		t.Fatalf("expected components status 200, got %d body=%s", componentsRec.Code, componentsRec.Body.String())
	}

	var componentsPayload api.HealthCheckResponse
	if err := json.Unmarshal(componentsRec.Body.Bytes(), &componentsPayload); err != nil {
		t.Fatalf("decode components response: %v", err)
	}
	component, ok := componentsPayload.Components["indexing_runtime"]
	if !ok {
		t.Fatal("expected indexing_runtime component")
	}
	if got := component.Details["runtime_mode"]; got != "runtime-wired" {
		t.Fatalf("expected runtime mode runtime-wired, got %v", got)
	}
}

func TestBuildEventProcessorRuntimeHTTPHandlerExposesMetricsRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()

	metrics.RecordCounter("event_processor_test_counter", 3, nil)
	metrics.RecordGauge("event_processor_test_gauge", 7, map[string]string{"component": "runtime"})

	mux := buildEventProcessorRuntimeHTTPHandler(handler, metrics, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; version=0.0.4" {
		t.Fatalf("expected prometheus content type, got %q", got)
	}

	body := rec.Body.String()
	for _, expected := range []string{
		`chainpulse_event_processor_test_counter{chain_id="global"} 3`,
		`chainpulse_event_processor_test_gauge{chain_id="global",component="runtime"} 7`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", expected, body)
		}
	}
}

func TestBuildEventProcessorRuntimeHTTPHandlerExposesRuntimeSummaryRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()
	metrics.RecordCounter("summary_counter", 1, nil)
	metrics.RecordGauge("summary_gauge", 2, nil)

	summaryProvider := func(r *http.Request) *eventProcessorRuntimeSummaryResponse {
		return &eventProcessorRuntimeSummaryResponse{
			Service:        "event-processor",
			Timestamp:      1712345678,
			DeploymentMode: "microservice",
			RuntimeMode:    "runtime-wired",
			RuntimePosture: "runtime-wired",
			ComponentState: "healthy",
			Rollout: map[string]interface{}{
				"rollout_gate_decision":     "allow",
				"consumer_progress_posture": "consumer-advancing",
			},
			Processor: map[string]interface{}{
				"runtime_ready":      true,
				"health_status":      "healthy",
				"execution_boundary": "consume-process-seam",
			},
			Metrics: map[string]interface{}{
				"collector_state":   "available",
				"counter_count":     1,
				"gauge_count":       1,
				"histogram_count":   0,
				"execution_summary": "counters=1 gauges=1 histograms=0",
			},
			Security: map[string]interface{}{
				"auth_enabled":       false,
				"rate_limit_enabled": false,
				"security_posture":   "event-processor-security-unconfigured",
			},
		}
	}

	mux := buildEventProcessorRuntimeHTTPHandler(handler, metrics, summaryProvider, nil)

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected runtime summary status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload eventProcessorRuntimeSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime summary response: %v", err)
	}
	if payload.Service != "event-processor" {
		t.Fatalf("expected service event-processor, got %q", payload.Service)
	}
	if payload.DeploymentMode != "microservice" {
		t.Fatalf("expected deployment mode microservice, got %q", payload.DeploymentMode)
	}
	if payload.RuntimeMode != "runtime-wired" {
		t.Fatalf("expected runtime mode runtime-wired, got %q", payload.RuntimeMode)
	}
	if got := payload.Rollout["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout gate decision allow, got %v", got)
	}
	if got := payload.Processor["execution_boundary"]; got != "consume-process-seam" {
		t.Fatalf("expected consume-process-seam processor boundary, got %v", got)
	}
	if got := payload.Metrics["execution_summary"]; got != "counters=1 gauges=1 histograms=0" {
		t.Fatalf("expected execution summary, got %v", got)
	}
	if got := payload.Security["auth_enabled"]; got != false {
		t.Fatalf("expected auth disabled by default, got %v", got)
	}
	if got := payload.Security["rate_limit_enabled"]; got != false {
		t.Fatalf("expected rate limit disabled by default, got %v", got)
	}
	if got := payload.Security["security_posture"]; got != "event-processor-security-unconfigured" {
		t.Fatalf("expected security posture event-processor-security-unconfigured, got %v", got)
	}
}

func TestNewEventProcessorRuntimeHTTPServerUsesRuntimeHandler(t *testing.T) {
	server := newEventProcessorRuntimeHTTPServer(8092, nil, core.NewDefaultMetricsCollector(), nil, nil)
	if server == nil {
		t.Fatal("expected runtime HTTP server")
	}
	if got := server.Addr; got != ":8092" {
		t.Fatalf("expected server addr :8092, got %q", got)
	}
	if server.Handler == nil {
		t.Fatal("expected runtime HTTP handler")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown server: %v", err)
	}
}

func TestBuildEventProcessorRuntimeHTTPHandlerExposesControlRoutes(t *testing.T) {
	controller := newEventProcessorConsumeRuntime(
		core.NewDefaultLogger(core.LogLevelInfo),
		core.NewDefaultMetricsCollector(),
		nil,
		nil,
		[]string{"raw-events", "blockchain-events"},
	)

	mux := buildEventProcessorRuntimeHTTPHandler(nil, nil, nil, controller)

	getReq := httptest.NewRequest(http.MethodGet, "/runtime/control", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected control get status 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}

	var getPayload eventProcessorRuntimeControlResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getPayload); err != nil {
		t.Fatalf("decode control payload: %v", err)
	}
	if err := api.ValidateRuntimeControlEnvelopeWithTarget(api.RuntimeControlEnvelopeWithTarget{
		Service:   getPayload.Service,
		Timestamp: getPayload.Timestamp,
		Target:    getPayload.Target,
		Control:   getPayload.Control.runtimeControlCore(),
	}, "event-processor", api.RuntimeControlTargetConsumeLoopIntake); err != nil {
		t.Fatalf("expected shared control get validation: %v", err)
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/runtime/control/pause-intake", nil)
	pauseRec := httptest.NewRecorder()
	mux.ServeHTTP(pauseRec, pauseReq)
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("expected pause status 200, got %d body=%s", pauseRec.Code, pauseRec.Body.String())
	}

	var pausePayload eventProcessorRuntimeControlResponse
	if err := json.Unmarshal(pauseRec.Body.Bytes(), &pausePayload); err != nil {
		t.Fatalf("decode pause payload: %v", err)
	}
	if err := api.ValidateRuntimeControlEnvelopeWithTarget(api.RuntimeControlEnvelopeWithTarget{
		Service:   pausePayload.Service,
		Timestamp: pausePayload.Timestamp,
		Target:    pausePayload.Target,
		Control:   pausePayload.Control.runtimeControlCore(),
	}, "event-processor", api.RuntimeControlTargetConsumeLoopIntake); err != nil {
		t.Fatalf("expected shared pause validation: %v", err)
	}
	if got := pausePayload.Target; got != api.RuntimeControlTargetConsumeLoopIntake {
		t.Fatalf("expected target %s, got %q", api.RuntimeControlTargetConsumeLoopIntake, got)
	}
	if !pausePayload.Control.Paused {
		t.Fatal("expected paused intake control")
	}
	if got := pausePayload.Control.State; got != "paused" {
		t.Fatalf("expected paused state, got %q", got)
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/runtime/control/resume-intake", nil)
	resumeRec := httptest.NewRecorder()
	mux.ServeHTTP(resumeRec, resumeReq)
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("expected resume status 200, got %d body=%s", resumeRec.Code, resumeRec.Body.String())
	}

	var resumePayload eventProcessorRuntimeControlResponse
	if err := json.Unmarshal(resumeRec.Body.Bytes(), &resumePayload); err != nil {
		t.Fatalf("decode resume payload: %v", err)
	}
	if err := api.ValidateRuntimeControlEnvelopeWithTarget(api.RuntimeControlEnvelopeWithTarget{
		Service:   resumePayload.Service,
		Timestamp: resumePayload.Timestamp,
		Target:    resumePayload.Target,
		Control:   resumePayload.Control.runtimeControlCore(),
	}, "event-processor", api.RuntimeControlTargetConsumeLoopIntake); err != nil {
		t.Fatalf("expected shared resume validation: %v", err)
	}
}

func TestBuildEventProcessorRuntimeHTTPHandlerSecuritySurfaceProtectsControlRoutes(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()
	controller := newEventProcessorConsumeRuntime(
		logger,
		metrics,
		nil,
		nil,
		[]string{"raw-events"},
	)

	mux := buildEventProcessorRuntimeHTTPHandler(handler, metrics, nil, controller)
	authMiddleware, rateLimitMiddleware, err := buildEventProcessorSecurityControls(EventProcessorConfig{
		AuthEnabled:   true,
		AuthJWTSecret: "secret-123",
		AuthAPIKeys:   []string{"svc-key=client-1"},
	}, logger, metrics)
	if err != nil {
		t.Fatalf("build security controls: %v", err)
	}
	wrapped := wrapEventProcessorRuntimeSecurityHandler(mux, authMiddleware, rateLimitMiddleware)

	req := httptest.NewRequest(http.MethodGet, "/runtime/control", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without api key, got %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/runtime/control", nil)
	req.Header.Set("X-API-Key", "svc-key")
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected control status 200 with api key, got %d body=%s", rec.Code, rec.Body.String())
	}
}
