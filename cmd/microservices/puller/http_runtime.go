package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

type pullerRuntimeSummaryResponse struct {
	Service        string                 `json:"service"`
	Timestamp      int64                  `json:"timestamp"`
	DeploymentMode string                 `json:"deployment_mode"`
	RuntimeMode    string                 `json:"runtime_mode"`
	RuntimePosture string                 `json:"runtime_posture"`
	ComponentState string                 `json:"component_state"`
	Rollout        map[string]interface{} `json:"rollout"`
	Security       map[string]interface{} `json:"security"`
	Metrics        map[string]interface{} `json:"metrics"`
}

func buildPullerRuntimeHTTPHandler(
	healthHandler *api.HealthCheckHandler,
	metrics core.MetricsCollector,
	summaryProvider func(*http.Request) *pullerRuntimeSummaryResponse,
	controller *pullerLoopController,
) http.Handler {
	mux := http.NewServeMux()
	if metrics != nil {
		mux.HandleFunc("/metrics", buildPullerMetricsHandler(metrics))
	}
	if summaryProvider != nil {
		mux.HandleFunc("/runtime/summary", buildPullerRuntimeSummaryHandler(summaryProvider))
	}
	if controller != nil {
		mux.HandleFunc("/runtime/control", buildPullerRuntimeControlGetHandler(controller))
		mux.HandleFunc("/runtime/control/pause", buildPullerRuntimeControlPauseHandler(controller))
		mux.HandleFunc("/runtime/control/resume", buildPullerRuntimeControlResumeHandler(controller))
	}

	if healthHandler != nil {
		mux.HandleFunc("/health", healthHandler.HandleHealth)
		mux.HandleFunc("/health/ready", healthHandler.HandleReady)
		mux.HandleFunc("/health/live", healthHandler.HandleLive)
		mux.HandleFunc("/health/components", healthHandler.HandleComponents)
		mux.HandleFunc("/health/rollout", healthHandler.HandleRollout)
	}

	return mux
}

//nolint:wsl // Security middleware stacking is intentionally explicit here.
//nolint:wsl,nlreturn // Security middleware stacking is intentionally explicit here.
func wrapPullerRuntimeSecurityHandler(
	handler http.Handler,
	authMiddleware *api.AuthMiddleware,
	rateLimitMiddleware *api.RateLimitMiddleware,
) http.Handler {
	wrapped := handler
	if rateLimitMiddleware != nil && rateLimitMiddleware.Limiter() != nil {
		wrapped = rateLimitMiddleware.Middleware(rateLimitMiddleware.Limiter())(wrapped)
	}
	if authMiddleware != nil {
		wrapped = authMiddleware.Handler(wrapped)
	}

	return wrapped
}

func buildPullerRuntimeSummaryHandler(
	summaryProvider func(*http.Request) *pullerRuntimeSummaryResponse,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if summaryProvider == nil {
			http.Error(w, `{"error":"runtime summary unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		summary := summaryProvider(r)
		if summary == nil {
			http.Error(w, `{"error":"runtime summary unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			http.Error(w, `{"error":"failed to encode runtime summary"}`, http.StatusInternalServerError)
		}
	}
}

func buildPullerMetricsHandler(metrics core.MetricsCollector) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if metrics == nil {
			http.Error(w, `{"error":"metrics collector unavailable"}`, http.StatusServiceUnavailable)
			return
		}

		if r.URL.Query().Get("format") == "json" || r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")

			if err := json.NewEncoder(w).Encode(metrics.GetMetrics()); err != nil {
				http.Error(w, `{"error":"failed to encode metrics"}`, http.StatusInternalServerError)
			}

			return
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(core.ExportMetricsPrometheus(metrics)))
	}
}

func newPullerRuntimeHTTPServer(
	port int,
	healthHandler *api.HealthCheckHandler,
	metrics core.MetricsCollector,
	summaryProvider func(*http.Request) *pullerRuntimeSummaryResponse,
	controller *pullerLoopController,
) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           buildPullerRuntimeHTTPHandler(healthHandler, metrics, summaryProvider, controller),
		ReadHeaderTimeout: 10 * time.Second,
	}
}
