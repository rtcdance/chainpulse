package main

import (
	"context"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

const (
	apiServiceRolloutSkeletonReason       = "api-service rollout producer skeleton is not yet wired to ownership runtime state"
	apiServiceRolloutRuntimeWiringReason  = "api-service rollout producer reflects local runtime wiring state; ownership runtime state is not yet wired"
	apiServiceOwnershipParityReviewFields = "runtime_routes_enabled,event_query_enabled,domain_bridge_enabled"
)

type apiServiceRolloutRuntimeState struct {
	DomainBridgeEnabled      bool
	EventQueryEnabled        bool
	EventSubscriptionEnabled bool
	HealthCheckRoutesEnabled bool
	QueryServiceMessage      string
	QueryServiceStatus       string
	RuntimeRoutesEnabled     bool
}

type apiServiceRolloutWiringCompleteness struct {
	Mode           string
	AdvisoryStatus string
	AdvisoryReady  bool
	EnabledSignals []string
	MissingSignals []string
	MonolithParity api.RouteOwnershipParitySourceSnapshot
	OwnershipState api.RouteOwnershipParityState
	OwnershipHint  string
	PostureHint    string
	Reason         string
}

func buildAPIServiceRolloutSummary() api.RolloutReportSummary {
	return api.RolloutReportSummary{
		ShadowOwnedEvents: 0,
		LegacyOwnedEvents: 0,
		OwnershipChains:   0,
	}
}

func newAPIServiceRolloutReportProducer(instanceID string, stateProvider func() apiServiceRolloutRuntimeState) api.RolloutReportProducer {
	return newAPIServiceRolloutReportProducerWithOwnershipSource(instanceID, stateProvider, nil)
}

//nolint:unused
func newAPIServiceRolloutReportProducerWithReadinessDetails(
	instanceID string,
	stateProvider func() apiServiceRolloutRuntimeState,
	readinessDetailsProvider func() map[string]any,
) api.RolloutReportProducer {
	return newAPIServiceRolloutReportProducerWithOwnershipSource(
		instanceID,
		stateProvider,
		buildAPIServiceOwnershipParitySourceFromReadinessDetails(readinessDetailsProvider),
	)
}

func newAPIServiceRolloutReportProducerWithOwnershipSource(
	instanceID string,
	stateProvider func() apiServiceRolloutRuntimeState,
	ownershipSource api.RouteOwnershipParitySource,
) api.RolloutReportProducer {
	return api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx

		details := api.NewRolloutReportDetailsFromMetadata(api.NewOwnershipRolloutReportMetadata(
			"api-service",
			"api-service-ownership-rollout-runtime",
			"microservice",
			"microservice",
			time.Now().Unix(),
		))

		if instanceID != "" {
			details.ReportSource = "microservice:" + instanceID
		}

		runtimeState := apiServiceRolloutRuntimeState{}
		if stateProvider != nil {
			runtimeState = stateProvider()
		}

		if runtimeState.RuntimeRoutesEnabled ||
			runtimeState.EventQueryEnabled ||
			runtimeState.EventSubscriptionEnabled ||
			runtimeState.HealthCheckRoutesEnabled ||
			runtimeState.DomainBridgeEnabled {
			applyAPIServiceRuntimeDerivedRollout(details, runtimeState, ownershipSource)
			return details
		}

		applyAPIServiceSkeletonRollout(details)
		return details
	})
}

func applyAPIServiceSkeletonRollout(details *api.RolloutReportDetails) {
	if details == nil {
		return
	}

	applyAPIServiceRolloutReportSections(details, buildAPIServiceSkeletonSections())
}

func applyAPIServiceRuntimeDerivedRollout(details *api.RolloutReportDetails, runtimeState apiServiceRolloutRuntimeState, ownershipSource api.RouteOwnershipParitySource) {
	if details == nil {
		return
	}

	applyAPIServiceRolloutReportSections(details, buildAPIServiceRuntimeDerivedSections(runtimeState, ownershipSource))
}

//lint:ignore U1000 These helpers stay in place for the runtime-derived rollout variants.
func buildAPIServiceSkeletonSurfaceSection() api.RolloutReportSurfaceSection {
	return api.BuildRolloutReportSurfaceSection(buildAPIServiceSkeletonSurfaceInput())
}

func buildAPIServiceSkeletonSurfaceInput() api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		buildAPIServiceSkeletonSurfaceCoreInput(),
		buildAPIServiceSkeletonSurfaceCutoverInput(),
	)
}

func buildAPIServiceSkeletonSurfaceCoreInput() api.RolloutReportSurfaceCoreInput {
	return api.RolloutReportSurfaceCoreInput{
		Summary: buildAPIServiceRolloutSummary(),
		Mode:    "unavailable",
		Advisory: api.RolloutReportAdvisory{
			Decision: "unknown",
			Status:   "unavailable",
			Ready:    false,
			Reason:   apiServiceRolloutSkeletonReason,
		},
		Policy: api.RolloutReportPolicy{
			Mode:         "report-only",
			Action:       "report-unknown",
			Reason:       apiServiceRolloutSkeletonReason,
			Acknowledged: false,
			AckState:     "pending",
		},
		Progression: api.RolloutReportStateReason{
			State:  "unknown",
			Reason: apiServiceRolloutSkeletonReason,
		},
	}
}

func buildAPIServiceSkeletonSurfaceCutoverInput() api.RolloutReportSurfaceCutoverInput {
	return api.RolloutReportSurfaceCutoverInput{
		CutoverDryRun: api.RolloutReportAction{
			Action: "would-unknown",
			Reason: apiServiceRolloutSkeletonReason,
		},
		CutoverCandidate: api.RolloutReportCandidate{
			Eligible: false,
			Reason:   apiServiceRolloutSkeletonReason,
		},
	}
}

//lint:ignore U1000 These helpers stay in place for the runtime-derived rollout variants.
func buildAPIServiceRuntimeDerivedSurfaceSection(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportSurfaceSection {
	return api.BuildRolloutReportSurfaceSection(buildAPIServiceRuntimeDerivedSurfaceInput(completeness))
}

func buildAPIServiceRuntimeDerivedSurfaceInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		buildAPIServiceRuntimeDerivedSurfaceCoreInput(completeness),
		buildAPIServiceRuntimeDerivedSurfaceCutoverInput(completeness),
	)
}

func buildAPIServiceRuntimeDerivedSurfaceCoreInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportSurfaceCoreInput {
	return api.RolloutReportSurfaceCoreInput{
		Summary: buildAPIServiceRolloutSummary(),
		Mode:    completeness.Mode,
		Advisory: api.RolloutReportAdvisory{
			Decision: "hold",
			Status:   completeness.AdvisoryStatus,
			Ready:    completeness.AdvisoryReady,
			Reason:   completeness.Reason,
		},
		Policy: api.RolloutReportPolicy{
			Mode:         "report-only",
			Action:       "report-hold",
			Reason:       completeness.Reason,
			Acknowledged: false,
			AckState:     "pending",
		},
		Progression: api.RolloutReportStateReason{
			State:  "observe",
			Reason: completeness.Reason,
		},
	}
}

func buildAPIServiceRuntimeDerivedSurfaceCutoverInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportSurfaceCutoverInput {
	return api.RolloutReportSurfaceCutoverInput{
		CutoverDryRun: api.RolloutReportAction{
			Action: "would-hold",
			Reason: completeness.Reason,
		},
		CutoverCandidate: api.RolloutReportCandidate{
			Eligible: false,
			Reason:   completeness.Reason,
		},
	}
}

//lint:ignore U1000 These helpers stay in place for the runtime-derived rollout variants.
func buildAPIServiceSkeletonApprovalSection() api.RolloutReportApproval {
	return api.BuildRolloutReportApprovalSection(buildAPIServiceSkeletonApprovalInput())
}

func buildAPIServiceSkeletonApprovalInput() api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		buildAPIServiceSkeletonApprovalFlowInput(),
		buildAPIServiceSkeletonApprovalWorkItemInput(),
	)
}

func buildAPIServiceSkeletonApprovalFlowInput() api.RolloutReportApprovalFlowInput {
	return api.RolloutReportApprovalFlowInput{
		ManualApprovalCheckpoint: api.RolloutReportStateReason{
			State:  "unknown",
			Reason: apiServiceRolloutSkeletonReason,
		},
		OperatorHandoff: api.RolloutReportStateReason{
			State:  "investigate",
			Reason: apiServiceRolloutSkeletonReason,
		},
		Checklist: api.RolloutReportStateReason{
			State:  "investigate",
			Reason: apiServiceRolloutSkeletonReason,
		},
	}
}

func buildAPIServiceSkeletonApprovalWorkItemInput() api.RolloutReportApprovalWorkItemInput {
	state := api.BuildRouteOwnershipParityStateFromSource("api-service", buildAPIServiceOwnershipParitySource( /* runtimeSignalsPresent= */ false), "rollout_effective_state", "rollout_cutover_candidate")
	return api.BuildOwnershipParityApprovalWorkItem(api.OwnershipParityApprovalWorkItemInput{
		State:  state,
		Status: "investigate",
		Owner:  "platform-team/runtime-owners",
	})
}

//lint:ignore U1000 These helpers stay in place for the runtime-derived rollout variants.
func buildAPIServiceRuntimeDerivedApprovalSection(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportApproval {
	return api.BuildRolloutReportApprovalSection(buildAPIServiceRuntimeDerivedApprovalInput(completeness))
}

func buildAPIServiceRuntimeDerivedApprovalInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		buildAPIServiceRuntimeDerivedApprovalFlowInput(completeness),
		buildAPIServiceRuntimeDerivedApprovalWorkItemInput(completeness),
	)
}

func buildAPIServiceRuntimeDerivedApprovalFlowInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportApprovalFlowInput {
	return api.RolloutReportApprovalFlowInput{
		ManualApprovalCheckpoint: api.RolloutReportStateReason{
			State:  "inactive",
			Reason: completeness.Reason,
		},
		OperatorHandoff: api.RolloutReportStateReason{
			State:  "none",
			Reason: completeness.Reason,
		},
		Checklist: api.RolloutReportStateReason{
			State:  "incomplete",
			Reason: completeness.Reason,
		},
	}
}

func buildAPIServiceRuntimeDerivedApprovalWorkItemInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportApprovalWorkItemInput {
	return api.BuildOwnershipParityApprovalWorkItem(api.OwnershipParityApprovalWorkItemInput{
		State:  completeness.OwnershipState,
		Status: "none",
		Owner:  "none",
	})
}

//lint:ignore U1000 These helpers stay in place for the runtime-derived rollout variants.
func buildAPIServiceSkeletonGuardedSection() api.RolloutReportGuarded {
	return api.BuildRolloutReportGuardedSection(buildAPIServiceSkeletonGuardedInput())
}

func buildAPIServiceSkeletonGuardedInput() api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		buildAPIServiceSkeletonGuardedHookInput(),
		buildAPIServiceSkeletonGuardedEnforcementInput(),
	)
}

func buildAPIServiceSkeletonGuardedHookInput() api.RolloutReportGuardedHookInput {
	return api.RolloutReportGuardedHookInput{
		Hook: api.RolloutReportAction{
			Action: "noop-investigate",
			Reason: apiServiceRolloutSkeletonReason,
		},
		HookPolicy: api.RolloutReportModeAction{
			Mode:   "noop-only",
			Action: "noop-investigate",
			Reason: apiServiceRolloutSkeletonReason,
		},
	}
}

func buildAPIServiceSkeletonGuardedEnforcementInput() api.RolloutReportGuardedEnforcementInput {
	return api.RolloutReportGuardedEnforcementInput{
		WouldEnforce: api.RolloutReportAction{
			Action: "would-investigate",
			Reason: apiServiceRolloutSkeletonReason,
		},
		EnforceHint: api.RolloutReportStateReason{
			State:  "investigate-before-enforce",
			Reason: apiServiceRolloutSkeletonReason,
		},
		Overview: api.RolloutReportStateReason{
			State:  "investigate",
			Reason: apiServiceRolloutSkeletonReason,
		},
	}
}

//lint:ignore U1000 These helpers stay in place for the runtime-derived rollout variants.
func buildAPIServiceRuntimeDerivedGuardedSection(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportGuarded {
	return api.BuildRolloutReportGuardedSection(buildAPIServiceRuntimeDerivedGuardedInput(completeness))
}

func buildAPIServiceRuntimeDerivedGuardedInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		buildAPIServiceRuntimeDerivedGuardedHookInput(completeness),
		buildAPIServiceRuntimeDerivedGuardedEnforcementInput(completeness),
	)
}

func buildAPIServiceRuntimeDerivedGuardedHookInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportGuardedHookInput {
	return api.RolloutReportGuardedHookInput{
		Hook: api.RolloutReportAction{
			Action: "noop-hold",
			Reason: completeness.Reason,
		},
		HookPolicy: api.RolloutReportModeAction{
			Mode:   "noop-only",
			Action: "noop-hold",
			Reason: completeness.Reason,
		},
	}
}

func buildAPIServiceRuntimeDerivedGuardedEnforcementInput(completeness apiServiceRolloutWiringCompleteness) api.RolloutReportGuardedEnforcementInput {
	return api.RolloutReportGuardedEnforcementInput{
		WouldEnforce: api.RolloutReportAction{
			Action: "would-hold",
			Reason: completeness.Reason,
		},
		EnforceHint: api.RolloutReportStateReason{
			State:  "hold-before-enforce",
			Reason: completeness.Reason,
		},
		Overview: api.RolloutReportStateReason{
			State:  "hold",
			Reason: completeness.Reason,
		},
	}
}

func classifyAPIServiceRolloutWiringCompleteness(runtimeState apiServiceRolloutRuntimeState, ownershipSource api.RouteOwnershipParitySource) apiServiceRolloutWiringCompleteness {
	enabled := make([]string, 0, 5)
	missing := make([]string, 0, 5)

	collectAPIServiceRuntimeSignal(&enabled, &missing, "runtime_routes_enabled", runtimeState.RuntimeRoutesEnabled)
	collectAPIServiceRuntimeSignal(&enabled, &missing, "event_query_enabled", runtimeState.EventQueryEnabled)
	collectAPIServiceRuntimeSignal(&enabled, &missing, "event_subscription_enabled", runtimeState.EventSubscriptionEnabled)
	collectAPIServiceRuntimeSignal(&enabled, &missing, "health_check_routes_enabled", runtimeState.HealthCheckRoutesEnabled)
	collectAPIServiceRuntimeSignal(&enabled, &missing, "domain_bridge_enabled", runtimeState.DomainBridgeEnabled)

	mode := "partially-wired"
	advisoryStatus := "partial-runtime-wiring"
	advisoryReady := false
	if len(missing) == 0 && len(enabled) > 0 {
		mode = "runtime-wired"
		advisoryStatus, advisoryReady = classifyAPIServiceRuntimeWiringAdvisory(runtimeState.QueryServiceStatus)
	}

	var parts []string
	parts = append(parts, apiServiceRolloutRuntimeWiringReason)
	if len(enabled) > 0 {
		parts = append(parts, "enabled: "+strings.Join(enabled, ","))
	}
	if len(missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(missing, ","))
	}
	if runtimeState.QueryServiceStatus != "" {
		parts = append(parts, "query_service_status: "+runtimeState.QueryServiceStatus)
	}
	if runtimeState.QueryServiceMessage != "" {
		parts = append(parts, "query_service_message: "+runtimeState.QueryServiceMessage)
	}
	if hint := classifyAPIServiceQueryHealthHint(runtimeState.QueryServiceStatus); hint != "" {
		parts = append(parts, "query_service_hint: "+hint)
	}
	ownershipState := api.BuildRouteOwnershipParityStateFromSource("api-service", buildAPIServiceOwnershipParitySource(len(enabled) > 0), strings.Split(apiServiceOwnershipParityReviewFields, ",")...)
	monolithParity := api.RouteOwnershipParitySourceSnapshot{}
	if ownershipSource != nil {
		monolithParity = ownershipSource.SnapshotRouteOwnershipParity()
	}
	parts = api.AppendMonolithOwnershipParityReason(parts, monolithParity)
	parts, ownershipHint := api.AppendRouteOwnershipParityStateReason(parts, ownershipState)
	postureHint := classifyAPIServiceRolloutPostureHint(mode, advisoryStatus, len(missing))
	if postureHint != "" {
		parts = append(parts, "rollout_posture_hint: "+postureHint)
	}

	return apiServiceRolloutWiringCompleteness{
		Mode:           mode,
		AdvisoryStatus: advisoryStatus,
		AdvisoryReady:  advisoryReady,
		EnabledSignals: enabled,
		MissingSignals: missing,
		MonolithParity: monolithParity,
		OwnershipState: ownershipState,
		OwnershipHint:  ownershipHint,
		PostureHint:    postureHint,
		Reason:         strings.Join(parts, "; "),
	}
}

//nolint:unused
func classifyAPIServiceOwnershipParityHint(runtimeSignalsPresent bool) string {
	return api.BuildRouteOwnershipParityStateFromSource("api-service", buildAPIServiceOwnershipParitySource(runtimeSignalsPresent)).Hint
}

func buildAPIServiceOwnershipParitySource(runtimeSignalsPresent bool) api.RouteOwnershipParitySource {
	return api.RouteOwnershipParitySourceFunc(func() api.RouteOwnershipParitySourceSnapshot {
		return api.RouteOwnershipParitySourceSnapshot{
			RuntimeSignalsPresent: runtimeSignalsPresent,
		}
	})
}

//nolint:unused
func buildAPIServiceOwnershipParitySourceFromReadinessDetails(readinessDetailsProvider func() map[string]any) api.RouteOwnershipParitySource {
	return api.RouteOwnershipParitySourceFunc(func() api.RouteOwnershipParitySourceSnapshot {
		if readinessDetailsProvider == nil {
			return api.RouteOwnershipParitySourceSnapshot{}
		}
		return api.BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails(readinessDetailsProvider())
	})
}

func classifyAPIServiceRuntimeWiringAdvisory(queryServiceStatus string) (string, bool) {
	switch queryServiceStatus {
	case "healthy":
		return "runtime-wired", true
	case "degraded":
		return "runtime-wired-degraded", false
	case "unhealthy":
		return "runtime-wired-unhealthy", false
	default:
		return "runtime-wired-query-unknown", false
	}
}

func classifyAPIServiceQueryHealthHint(queryServiceStatus string) string {
	switch queryServiceStatus {
	case "healthy":
		return "query runtime is healthy enough to support runtime-wired api-service routes"
	case "degraded":
		return "investigate degraded query runtime before treating runtime-wired api-service routes as ready"
	case "unhealthy":
		return "restore query runtime health before relying on runtime-wired api-service routes"
	case "":
		return ""
	default:
		return "verify query runtime health before treating runtime-wired api-service routes as ready"
	}
}

func classifyAPIServiceRolloutPostureHint(mode, advisoryStatus string, missingSignals int) string {
	switch {
	case mode == "partially-wired" && missingSignals > 0:
		return "finish wiring the missing api-service runtime routes before treating this rollout as ready"
	case advisoryStatus == "runtime-wired":
		return "observe the runtime-wired api-service rollout while query runtime health remains healthy"
	case advisoryStatus == "runtime-wired-degraded":
		return "hold this rollout posture until degraded query runtime health is investigated and stabilized"
	case advisoryStatus == "runtime-wired-unhealthy":
		return "treat this rollout posture as blocked until query runtime health is restored"
	case advisoryStatus == "runtime-wired-query-unknown":
		return "investigate query runtime health before treating this runtime-wired rollout as ready"
	default:
		return ""
	}
}

func collectAPIServiceRuntimeSignal(enabled, missing *[]string, label string, active bool) {
	if active {
		*enabled = append(*enabled, label)
		return
	}
	*missing = append(*missing, label)
}
