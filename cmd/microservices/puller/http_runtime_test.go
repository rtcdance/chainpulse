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

func TestBuildPullerRuntimeHTTPHandlerExposesRolloutRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.SetRolloutReportProducer(newPullerRolloutReportProducer("puller-1", func() pullerRolloutRuntimeState {
		return pullerRolloutRuntimeState{
			DatabaseReady:            true,
			KafkaReady:               true,
			PullerLoopConfigured:     true,
			BlockchainRPCsConfigured: true,
		}
	}))
	handler.InitializedForTests()

	mux := buildPullerRuntimeHTTPHandler(handler, metrics, nil, nil)

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
	if got := payload.Details.Service; got != "puller" {
		t.Fatalf("expected service puller, got %q", got)
	}
}

func TestBuildPullerRuntimeHTTPHandlerExposesReadyAndComponentsDetails(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()

	handler := api.NewHealthCheckHandler(&pullerTestDatabaseManager{postgresHealthy: true}, logger, metrics)
	handler.SetRuntimeComponentProvider(func(ctx context.Context) *api.ComponentStatus {
		return &api.ComponentStatus{
			Name:      "Polling Runtime",
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

	mux := buildPullerRuntimeHTTPHandler(handler, metrics, nil, nil)

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

func TestBuildPullerRuntimeHTTPHandlerExposesMetricsRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()

	metrics.RecordCounter("puller_test_counter", 5, nil)
	metrics.RecordGauge("puller_test_gauge", 11, map[string]string{"component": "runtime"})

	mux := buildPullerRuntimeHTTPHandler(handler, metrics, nil, nil)

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
		`chainpulse_puller_test_counter{chain_id="global"} 5`,
		`chainpulse_puller_test_gauge{chain_id="global",component="runtime"} 11`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected metrics output to contain %q, got:\n%s", expected, body)
		}
	}
}

func TestBuildPullerRuntimeHTTPHandlerExposesRuntimeSummaryRoute(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()
	metrics.RecordCounter("summary_counter", 1, nil)
	metrics.RecordGauge("summary_gauge", 2, nil)

	summaryProvider := func(r *http.Request) *pullerRuntimeSummaryResponse {
		return &pullerRuntimeSummaryResponse{
			Service:        "puller",
			Timestamp:      1712345678,
			DeploymentMode: "microservice",
			RuntimeMode:    "runtime-wired",
			RuntimePosture: "runtime-wired",
			ComponentState: "healthy",
			Rollout: map[string]interface{}{
				"rollout_gate_decision": "allow",
				"poll_activity_state":   "active",
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
				"security_posture":   "puller-security-unconfigured",
			},
		}
	}

	mux := buildPullerRuntimeHTTPHandler(handler, metrics, summaryProvider, nil)

	req := httptest.NewRequest(http.MethodGet, "/runtime/summary", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected runtime summary status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	var payload pullerRuntimeSummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode runtime summary response: %v", err)
	}
	if payload.Service != "puller" {
		t.Fatalf("expected service puller, got %q", payload.Service)
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
	if got := payload.Metrics["execution_summary"]; got != "counters=1 gauges=1 histograms=0" {
		t.Fatalf("expected execution summary, got %v", got)
	}
	if got := payload.Security["auth_enabled"]; got != false {
		t.Fatalf("expected auth disabled by default, got %v", got)
	}
	if got := payload.Security["rate_limit_enabled"]; got != false {
		t.Fatalf("expected rate limit disabled by default, got %v", got)
	}
	if got := payload.Security["security_posture"]; got != "puller-security-unconfigured" {
		t.Fatalf("expected security posture puller-security-unconfigured, got %v", got)
	}
}

func TestBuildPullerRuntimeHTTPHandlerExposesRuntimeControlRoutes(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()
	controller := newPullerLoopController()

	mux := buildPullerRuntimeHTTPHandler(handler, metrics, nil, controller)

	getReq := httptest.NewRequest(http.MethodGet, "/runtime/control", nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected runtime control status 200, got %d body=%s", getRec.Code, getRec.Body.String())
	}

	var getPayload pullerRuntimeControlResponse
	if err := json.Unmarshal(getRec.Body.Bytes(), &getPayload); err != nil {
		t.Fatalf("decode runtime control response: %v", err)
	}
	if err := api.ValidateRuntimeControlEnvelopeWithTarget(api.RuntimeControlEnvelopeWithTarget{
		Service:   getPayload.Service,
		Timestamp: getPayload.Timestamp,
		Target:    getPayload.Target,
		Control:   getPayload.Control.runtimeControlCore(),
	}, "puller", api.RuntimeControlTargetPollingLoop); err != nil {
		t.Fatalf("expected shared runtime control validation: %v", err)
	}
	if getPayload.Control.State != "running" {
		t.Fatalf("expected initial control state running, got %q", getPayload.Control.State)
	}

	pauseReq := httptest.NewRequest(http.MethodPost, "/runtime/control/pause", nil)
	pauseRec := httptest.NewRecorder()
	mux.ServeHTTP(pauseRec, pauseReq)
	if pauseRec.Code != http.StatusOK {
		t.Fatalf("expected pause status 200, got %d body=%s", pauseRec.Code, pauseRec.Body.String())
	}

	var pausePayload pullerRuntimeControlResponse
	if err := json.Unmarshal(pauseRec.Body.Bytes(), &pausePayload); err != nil {
		t.Fatalf("decode pause response: %v", err)
	}
	if err := api.ValidateRuntimeControlEnvelopeWithTarget(api.RuntimeControlEnvelopeWithTarget{
		Service:   pausePayload.Service,
		Timestamp: pausePayload.Timestamp,
		Target:    pausePayload.Target,
		Control:   pausePayload.Control.runtimeControlCore(),
	}, "puller", api.RuntimeControlTargetPollingLoop); err != nil {
		t.Fatalf("expected shared pause validation: %v", err)
	}
	if pausePayload.Control.State != "paused" {
		t.Fatalf("expected paused control state, got %q", pausePayload.Control.State)
	}

	resumeReq := httptest.NewRequest(http.MethodPost, "/runtime/control/resume", nil)
	resumeRec := httptest.NewRecorder()
	mux.ServeHTTP(resumeRec, resumeReq)
	if resumeRec.Code != http.StatusOK {
		t.Fatalf("expected resume status 200, got %d body=%s", resumeRec.Code, resumeRec.Body.String())
	}

	var resumePayload pullerRuntimeControlResponse
	if err := json.Unmarshal(resumeRec.Body.Bytes(), &resumePayload); err != nil {
		t.Fatalf("decode resume response: %v", err)
	}
	if err := api.ValidateRuntimeControlEnvelopeWithTarget(api.RuntimeControlEnvelopeWithTarget{
		Service:   resumePayload.Service,
		Timestamp: resumePayload.Timestamp,
		Target:    resumePayload.Target,
		Control:   resumePayload.Control.runtimeControlCore(),
	}, "puller", api.RuntimeControlTargetPollingLoop); err != nil {
		t.Fatalf("expected shared resume validation: %v", err)
	}
	if resumePayload.Control.State != "running" {
		t.Fatalf("expected resumed control state running, got %q", resumePayload.Control.State)
	}
}

func TestBuildPullerRuntimeHTTPHandlerSecuritySurfaceProtectsControlRoutes(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelInfo)
	metrics := core.NewDefaultMetricsCollector()
	handler := api.NewHealthCheckHandler(nil, logger, metrics)
	handler.InitializedForTests()
	controller := newPullerLoopController()

	mux := buildPullerRuntimeHTTPHandler(handler, metrics, nil, controller)
	authMiddleware, rateLimitMiddleware, err := buildPullerSecurityControls(PullerConfig{
		AuthEnabled:   true,
		AuthJWTSecret: "secret-123",
		AuthAPIKeys:   []string{"svc-key=client-1"},
	}, logger, metrics)
	if err != nil {
		t.Fatalf("build security controls: %v", err)
	}
	wrapped := wrapPullerRuntimeSecurityHandler(mux, authMiddleware, rateLimitMiddleware)

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

func TestNewPullerRuntimeHTTPServerUsesRuntimeHandler(t *testing.T) {
	server := newPullerRuntimeHTTPServer(8093, nil, core.NewDefaultMetricsCollector(), nil, nil)
	if server == nil {
		t.Fatal("expected runtime HTTP server")
	}
	if got := server.Addr; got != ":8093" {
		t.Fatalf("expected server addr :8093, got %q", got)
	}
	if server.Handler == nil {
		t.Fatal("expected runtime HTTP handler")
	}
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown server: %v", err)
	}
}
