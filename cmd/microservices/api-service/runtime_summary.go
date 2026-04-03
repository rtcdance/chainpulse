package main

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
	"chainpulse/pkg/services/query"
)

type apiServiceRuntimeSummaryResponse struct {
	Service        string                 `json:"service"`
	Timestamp      int64                  `json:"timestamp"`
	RuntimeMode    string                 `json:"runtime_mode"`
	RuntimePosture string                 `json:"runtime_posture"`
	ComponentState string                 `json:"component_state"`
	Rollout        map[string]interface{} `json:"rollout"`
	Query          map[string]interface{} `json:"query"`
	Security       map[string]interface{} `json:"security"`
	Metrics        map[string]interface{} `json:"metrics"`
}

type apiServiceQueryRuntimeSummarizer interface {
	RuntimeSummary(ctx context.Context) *query.RuntimeSummary
}

func buildAPIServiceRuntimeSummaryProvider(
	instanceID string,
	metrics core.MetricsCollector,
	service *api.APIGatewayPlugin,
	queryService query.QueryService,
) func(*http.Request) interface{} {
	return func(r *http.Request) interface{} {
		_ = r
		return buildAPIServiceRuntimeSummaryResponse(instanceID, metrics, service, queryService)
	}
}

func buildAPIServiceRuntimeSummaryResponse(
	instanceID string,
	metrics core.MetricsCollector,
	service *api.APIGatewayPlugin,
	queryService query.QueryService,
) *apiServiceRuntimeSummaryResponse {
	runtimeState := apiServiceRolloutRuntimeState{}
	if service != nil {
		runtimeState.DomainBridgeEnabled = service.IsDomainBridgeEnabled()
		runtimeState.EventQueryEnabled = service.IsEventQueryHandlerEnabled()
		runtimeState.EventSubscriptionEnabled = service.IsEventSubscriptionHandlerEnabled()
		runtimeState.HealthCheckRoutesEnabled = service.IsHealthCheckHandlerEnabled()
		runtimeState.RuntimeRoutesEnabled = service.IsRuntimeRoutesEnabled()
	}

	queryHealthStatus := "unknown"
	queryHealthMessage := "query service unavailable"
	if queryService != nil {
		if health := queryService.Health(context.Background()); health != nil {
			if health.Status != "" {
				queryHealthStatus = health.Status
			}
			if health.Message != "" {
				queryHealthMessage = health.Message
			}
		}
	}
	runtimeState.QueryServiceStatus = queryHealthStatus
	runtimeState.QueryServiceMessage = queryHealthMessage

	completeness := classifyAPIServiceRolloutWiringCompleteness(runtimeState, nil)
	runtimeMode := "unavailable"
	runtimePosture := "runtime-unavailable"
	componentState := "unavailable"
	if runtimeState.RuntimeRoutesEnabled ||
		runtimeState.EventQueryEnabled ||
		runtimeState.EventSubscriptionEnabled ||
		runtimeState.HealthCheckRoutesEnabled ||
		runtimeState.DomainBridgeEnabled {
		runtimeMode = completeness.Mode
		runtimePosture = completeness.AdvisoryStatus
		componentState = queryHealthStatus
	}

	return &apiServiceRuntimeSummaryResponse{
		Service:        "api-service",
		Timestamp:      time.Now().Unix(),
		RuntimeMode:    runtimeMode,
		RuntimePosture: runtimePosture,
		ComponentState: componentState,
		Rollout: map[string]interface{}{
			"instance_id":          instanceID,
			"advisory_status":      completeness.AdvisoryStatus,
			"advisory_ready":       completeness.AdvisoryReady,
			"enabled_signals":      completeness.EnabledSignals,
			"missing_signals":      completeness.MissingSignals,
			"rollout_posture_hint": completeness.PostureHint,
		},
		Query:    buildAPIServiceQueryRuntimeSection(queryService, queryHealthStatus, queryHealthMessage),
		Security: buildAPIServiceSecurityRuntimeSection(service),
		Metrics:  buildAPIServiceMetricsSummary(metrics),
	}
}

func buildAPIServiceQueryRuntimeSection(
	queryService query.QueryService,
	queryHealthStatus, queryHealthMessage string,
) map[string]interface{} {
	section := map[string]interface{}{
		"status":           queryHealthStatus,
		"message":          queryHealthMessage,
		"health_hint":      classifyAPIServiceQueryHealthHint(queryHealthStatus),
		"runtime_boundary": "query-service-backed",
	}

	summarizer, ok := queryService.(apiServiceQueryRuntimeSummarizer)
	if !ok || summarizer == nil {
		return section
	}

	summary := summarizer.RuntimeSummary(context.Background())
	if summary == nil {
		return section
	}

	section["query_posture"] = summary.QueryPosture
	section["cache_posture"] = summary.CachePosture
	section["circuit_breaker_posture"] = summary.CircuitBreakerPosture
	section["consistency_posture"] = summary.ConsistencyPosture
	section["reliability_hint"] = summary.ReliabilityHint
	return section
}

func buildAPIServiceMetricsSummary(metrics core.MetricsCollector) map[string]interface{} {
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

//nolint:wsl,nlreturn // Security/runtime summary is intentionally compact.
func buildAPIServiceSecurityRuntimeSection(service *api.APIGatewayPlugin) map[string]interface{} {
	authEnabled := false
	rateLimitEnabled := false
	if service != nil {
		authEnabled = service.IsAuthMiddlewareEnabled()
		rateLimitEnabled = service.IsRateLimitMiddlewareEnabled()
	}

	authPosture := classifyAPIServiceAuthPosture(authEnabled)
	rateLimitPosture := classifyAPIServiceRateLimitPosture(rateLimitEnabled)
	securityPosture := classifyAPIServiceSecurityPosture(authEnabled, rateLimitEnabled)

	return map[string]interface{}{
		"route_boundary":     "service-entrypoint",
		"auth_enabled":       authEnabled,
		"rate_limit_enabled": rateLimitEnabled,
		"auth_posture":       authPosture,
		"rate_limit_posture": rateLimitPosture,
		"security_posture":   securityPosture,
		"security_hint":      classifyAPIServiceSecurityHint(authEnabled, rateLimitEnabled),
		"runtime_boundary":   "optional-security-surface",
	}
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyAPIServiceAuthPosture(enabled bool) string {
	if !enabled {
		return "api-service-auth-unconfigured"
	}
	return "api-service-auth-ready"
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyAPIServiceRateLimitPosture(enabled bool) string {
	if !enabled {
		return "api-service-rate-limit-unconfigured"
	}
	return "api-service-rate-limit-ready"
}

func classifyAPIServiceSecurityPosture(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "api-service-security-unconfigured"
	case authEnabled && rateLimitEnabled:
		return "api-service-security-ready"
	default:
		return "api-service-security-partial"
	}
}

func classifyAPIServiceSecurityHint(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "api-service security controls are disabled by default; enable auth or rate limiting explicitly before exposing the entrypoint"
	case authEnabled && rateLimitEnabled:
		return "api-service security controls are aligned for the current runnable baseline"
	case authEnabled || rateLimitEnabled:
		return "api-service security controls are partially enabled; verify the remaining control surface before treating the entrypoint as hardened"
	default:
		return "api-service security controls are partially configured; verify auth and rate limit wiring"
	}
}
