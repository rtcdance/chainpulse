package main

import (
	"net/http"
	"strconv"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

type apiGatewayRuntimeSummaryResponse struct {
	Service        string                 `json:"service"`
	Timestamp      int64                  `json:"timestamp"`
	RuntimeMode    string                 `json:"runtime_mode"`
	RuntimePosture string                 `json:"runtime_posture"`
	ComponentState string                 `json:"component_state"`
	Rollout        map[string]interface{} `json:"rollout"`
	Gateway        map[string]interface{} `json:"gateway"`
	Metrics        map[string]interface{} `json:"metrics"`
}

func buildAPIGatewayRuntimeSummaryProvider(
	instanceID string,
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
) func(*http.Request) interface{} {
	return func(r *http.Request) interface{} {
		_ = r
		return buildAPIGatewayRuntimeSummaryResponse(instanceID, metrics, gateway)
	}
}

func buildAPIGatewayRuntimeSummaryResponse(
	instanceID string,
	metrics core.MetricsCollector,
	gateway *api.APIGatewayPlugin,
) *apiGatewayRuntimeSummaryResponse {
	runtimeState := apiGatewayRolloutRuntimeState{}
	if gateway != nil {
		runtimeState.DomainBridgeEnabled = gateway.IsDomainBridgeEnabled()
		runtimeState.EventQueryEnabled = gateway.IsEventQueryHandlerEnabled()
		runtimeState.EventSubscriptionEnabled = gateway.IsEventSubscriptionHandlerEnabled()
		runtimeState.HealthCheckEnabled = gateway.IsHealthCheckHandlerEnabled()
		runtimeState.RuntimeRoutesEnabled = gateway.IsRuntimeRoutesEnabled()
	}

	completeness := classifyAPIGatewayRolloutWiringCompleteness(runtimeState, nil)
	runtimeMode := "unavailable"
	runtimePosture := "runtime-unavailable"
	componentState := "unavailable"
	if runtimeState.RuntimeRoutesEnabled ||
		runtimeState.EventQueryEnabled ||
		runtimeState.EventSubscriptionEnabled ||
		runtimeState.HealthCheckEnabled ||
		runtimeState.DomainBridgeEnabled {
		runtimeMode = completeness.Mode
		runtimePosture = completeness.AdvisoryStatus
		componentState = completeness.AdvisoryStatus
	}

	upstreamConfigured := 0
	upstreamAttached := 0
	upstreamAvailable := 0
	upstreamBridgePosture := "query-bridge-unconfigured"
	upstreamBridgeHint := "no upstream query bridge is configured; gateway query routes are not yet backed by api-service"
	upstreamHealthState := "query-upstream-unconfigured"
	authEnabled := false
	rateLimitEnabled := false
	authPosture := "gateway-auth-unconfigured"
	rateLimitPosture := "gateway-rate-limit-unconfigured"
	securityPosture := "gateway-security-unconfigured"
	securityHint := "gateway security controls are disabled; query access is currently open in the runnable baseline"
	if gateway != nil {
		gateway.RefreshUpstreamQueryBridgeHealth()
		upstreamConfigured, upstreamAttached, upstreamAvailable = gateway.GetUpstreamQueryBridgeStatus()
		upstreamBridgePosture = classifyAPIGatewayUpstreamQueryBridgePosture(upstreamConfigured, upstreamAttached, upstreamAvailable)
		upstreamBridgeHint = classifyAPIGatewayUpstreamQueryBridgeHint(upstreamConfigured, upstreamAttached, upstreamAvailable)
		upstreamHealthState = classifyAPIGatewayUpstreamQueryHealthState(upstreamConfigured, upstreamAvailable)
		authEnabled = gateway.IsAuthMiddlewareEnabled()
		rateLimitEnabled = gateway.IsRateLimitMiddlewareEnabled()
		authPosture = classifyAPIGatewayAuthPosture(authEnabled)
		rateLimitPosture = classifyAPIGatewayRateLimitPosture(rateLimitEnabled)
		securityPosture = classifyAPIGatewaySecurityPosture(authEnabled, rateLimitEnabled)
		securityHint = classifyAPIGatewaySecurityHint(authEnabled, rateLimitEnabled, upstreamBridgePosture)
	}

	return &apiGatewayRuntimeSummaryResponse{
		Service:        "api-gateway",
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
		Gateway: map[string]interface{}{
			"route_boundary":                  "gateway-entrypoint",
			"gateway_posture":                 completeness.AdvisoryStatus,
			"domain_bridge_enabled":           runtimeState.DomainBridgeEnabled,
			"event_query_enabled":             runtimeState.EventQueryEnabled,
			"event_subscription_enabled":      runtimeState.EventSubscriptionEnabled,
			"health_check_enabled":            runtimeState.HealthCheckEnabled,
			"runtime_routes_enabled":          runtimeState.RuntimeRoutesEnabled,
			"upstream_query_configured_count": upstreamConfigured,
			"upstream_query_attached_count":   upstreamAttached,
			"upstream_query_available_count":  upstreamAvailable,
			"upstream_query_health_state":     upstreamHealthState,
			"query_bridge_posture":            upstreamBridgePosture,
			"reliability_hint":                classifyAPIGatewayRuntimeSummaryHint(completeness, upstreamBridgePosture),
			"query_bridge_hint":               upstreamBridgeHint,
			"auth_enabled":                    authEnabled,
			"rate_limit_enabled":              rateLimitEnabled,
			"auth_posture":                    authPosture,
			"rate_limit_posture":              rateLimitPosture,
			"security_posture":                securityPosture,
			"security_hint":                   securityHint,
		},
		Metrics: buildAPIGatewayMetricsSummary(metrics),
	}
}

func classifyAPIGatewayRuntimeSummaryHint(completeness apiGatewayRolloutWiringCompleteness, bridgePosture string) string {
	switch completeness.AdvisoryStatus {
	case "runtime-wired":
		if bridgePosture == "query-bridge-ready" {
			return "gateway runtime wiring is locally ready and query upstreams are attached; continue observing upstream availability"
		}
		return "gateway runtime wiring is locally ready, but query upstream bridge still needs attention"
	case "partial-runtime-wiring":
		return "gateway runtime wiring is partial; finish missing runtime route signals before treating the entrypoint as fully ready"
	default:
		return "gateway runtime wiring is unavailable; investigate runtime route composition before relying on the entrypoint"
	}
}

func classifyAPIGatewayUpstreamQueryBridgePosture(configured, attached, available int) string {
	switch {
	case configured == 0:
		return "query-bridge-unconfigured"
	case attached == 0:
		return "query-bridge-detached"
	case available == 0:
		return "query-bridge-unavailable"
	case attached < configured:
		return "query-bridge-partial"
	default:
		return "query-bridge-ready"
	}
}

func classifyAPIGatewayUpstreamQueryBridgeHint(configured, attached, available int) string {
	switch classifyAPIGatewayUpstreamQueryBridgePosture(configured, attached, available) {
	case "query-bridge-unconfigured":
		return "no upstream query bridge is configured; gateway query routes are not yet backed by api-service"
	case "query-bridge-detached":
		return "query upstreams are configured but not attached to runtime routes; verify gateway initialization"
	case "query-bridge-unavailable":
		return "query upstream handlers are attached but currently unavailable; verify api-service health and bridge wiring"
	case "query-bridge-partial":
		return "only part of the configured upstream query bridge is attached or available; verify gateway-to-api-service alignment"
	default:
		return "query upstream bridge is attached and available for the current gateway runtime"
	}
}

func classifyAPIGatewayUpstreamQueryHealthState(configured, available int) string {
	switch {
	case configured == 0:
		return "query-upstream-unconfigured"
	case available == configured:
		return "query-upstream-healthy"
	case available > 0:
		return "query-upstream-degraded"
	default:
		return "query-upstream-unhealthy"
	}
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyAPIGatewayAuthPosture(enabled bool) string {
	if !enabled {
		return "gateway-auth-unconfigured"
	}
	return "gateway-auth-ready"
}

//nolint:wsl,nlreturn // Compact posture helpers keep the summary readable.
func classifyAPIGatewayRateLimitPosture(enabled bool) string {
	if !enabled {
		return "gateway-rate-limit-unconfigured"
	}
	return "gateway-rate-limit-ready"
}

func classifyAPIGatewaySecurityPosture(authEnabled, rateLimitEnabled bool) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "gateway-security-unconfigured"
	case authEnabled && rateLimitEnabled:
		return "gateway-security-ready"
	default:
		return "gateway-security-partial"
	}
}

func classifyAPIGatewaySecurityHint(authEnabled, rateLimitEnabled bool, bridgePosture string) string {
	switch {
	case !authEnabled && !rateLimitEnabled:
		return "gateway security controls are disabled by default; enable auth or rate limiting explicitly before exposing the entrypoint"
	case authEnabled && rateLimitEnabled && bridgePosture == "query-bridge-ready":
		return "gateway security controls and query bridge are aligned for the current runnable baseline"
	case authEnabled || rateLimitEnabled:
		return "gateway security controls are partially enabled; verify the remaining control surface before treating the entrypoint as hardened"
	default:
		return "gateway security controls are partially configured; verify auth and rate limit wiring"
	}
}

func buildAPIGatewayMetricsSummary(metrics core.MetricsCollector) map[string]interface{} {
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
