package main

import (
	"context"
	"strings"
	"time"

	"github.com/rtcdance/chainpulse/pkg/plugins/api"
)

const (
	apiGatewayRolloutSkeletonReason       = "api-gateway rollout producer skeleton is not yet wired to ownership runtime state"
	apiGatewayRolloutRuntimeWiringReason  = "api-gateway rollout producer reflects local gateway runtime wiring; ownership runtime state is not yet wired"
	apiGatewayOwnershipParityReviewFields = "runtime_routes_enabled,event_query_enabled,event_subscription_enabled,health_check_enabled,domain_bridge_enabled"
)

type apiGatewayRolloutRuntimeState struct {
	DomainBridgeEnabled      bool
	EventQueryEnabled        bool
	EventSubscriptionEnabled bool
	HealthCheckEnabled       bool
	RuntimeRoutesEnabled     bool
}

type apiGatewayRolloutWiringCompleteness struct {
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

func buildAPIGatewayRolloutSummary() api.RolloutReportSummary {
	return api.RolloutReportSummary{
		ShadowOwnedEvents: 0,
		LegacyOwnedEvents: 0,
		OwnershipChains:   0,
	}
}

func newAPIGatewayRolloutReportProducer(instanceID string, stateProvider func() apiGatewayRolloutRuntimeState) api.RolloutReportProducer {
	return newAPIGatewayRolloutReportProducerWithOwnershipSource(instanceID, stateProvider, nil)
}

func newAPIGatewayRolloutReportProducerWithOwnershipSource(
	instanceID string,
	stateProvider func() apiGatewayRolloutRuntimeState,
	ownershipSource api.RouteOwnershipParitySource,
) api.RolloutReportProducer {
	return api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx

		details := api.NewRolloutReportDetailsFromMetadata(api.NewOwnershipRolloutReportMetadata(
			"api-gateway",
			"api-gateway-ownership-rollout-runtime",
			"microservice",
			"microservice",
			time.Now().Unix(),
		))
		if instanceID != "" {
			details.ReportSource = "microservice:" + instanceID
		}

		runtimeState := apiGatewayRolloutRuntimeState{}
		if stateProvider != nil {
			runtimeState = stateProvider()
		}

		if runtimeState.RuntimeRoutesEnabled ||
			runtimeState.EventQueryEnabled ||
			runtimeState.EventSubscriptionEnabled ||
			runtimeState.HealthCheckEnabled ||
			runtimeState.DomainBridgeEnabled {
			applyAPIGatewayRuntimeDerivedRollout(details, runtimeState, ownershipSource)
			return details
		}

		applyAPIGatewaySkeletonRollout(details)
		return details
	})
}

func applyAPIGatewaySkeletonRollout(details *api.RolloutReportDetails) {
	if details == nil {
		return
	}

	applyAPIGatewayRolloutReportSections(details, buildAPIGatewaySkeletonSections())
}

func applyAPIGatewayRuntimeDerivedRollout(details *api.RolloutReportDetails, runtimeState apiGatewayRolloutRuntimeState, ownershipSource api.RouteOwnershipParitySource) {
	if details == nil {
		return
	}

	applyAPIGatewayRolloutReportSections(details, buildAPIGatewayRuntimeDerivedSections(runtimeState, ownershipSource))
}

func buildAPIGatewaySkeletonSurfaceInput() api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		api.RolloutReportSurfaceCoreInput{
			Summary: buildAPIGatewayRolloutSummary(),
			Mode:    "unavailable",
			Advisory: api.RolloutReportAdvisory{
				Decision: "unknown",
				Status:   "unavailable",
				Ready:    false,
				Reason:   apiGatewayRolloutSkeletonReason,
			},
			Policy: api.RolloutReportPolicy{
				Mode:         "report-only",
				Action:       "report-unknown",
				Reason:       apiGatewayRolloutSkeletonReason,
				Acknowledged: false,
				AckState:     "pending",
			},
			Progression: api.RolloutReportStateReason{
				State:  "unknown",
				Reason: apiGatewayRolloutSkeletonReason,
			},
		},
		api.RolloutReportSurfaceCutoverInput{
			CutoverDryRun: api.RolloutReportAction{
				Action: "would-unknown",
				Reason: apiGatewayRolloutSkeletonReason,
			},
			CutoverCandidate: api.RolloutReportCandidate{
				Eligible: false,
				Reason:   apiGatewayRolloutSkeletonReason,
			},
		},
	)
}

func buildAPIGatewayRuntimeDerivedSurfaceInput(completeness apiGatewayRolloutWiringCompleteness) api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		api.RolloutReportSurfaceCoreInput{
			Summary: buildAPIGatewayRolloutSummary(),
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
		},
		api.RolloutReportSurfaceCutoverInput{
			CutoverDryRun: api.RolloutReportAction{
				Action: "would-hold",
				Reason: completeness.Reason,
			},
			CutoverCandidate: api.RolloutReportCandidate{
				Eligible: false,
				Reason:   completeness.Reason,
			},
		},
	)
}

func buildAPIGatewaySkeletonApprovalInput() api.RolloutReportApprovalInput {
	state := api.BuildRouteOwnershipParityStateFromSource("api-gateway", buildAPIGatewayOwnershipParitySource( /* runtimeSignalsPresent= */ false), strings.Split(apiGatewayOwnershipParityReviewFields, ",")...)
	return api.BuildRolloutReportApprovalInput(
		api.RolloutReportApprovalFlowInput{
			ManualApprovalCheckpoint: api.RolloutReportStateReason{
				State:  "unknown",
				Reason: apiGatewayRolloutSkeletonReason,
			},
			OperatorHandoff: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: apiGatewayRolloutSkeletonReason,
			},
			Checklist: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: apiGatewayRolloutSkeletonReason,
			},
		},
		api.BuildOwnershipParityApprovalWorkItem(api.OwnershipParityApprovalWorkItemInput{
			State:  state,
			Status: "investigate",
			Owner:  "platform-team/runtime-owners",
		}),
	)
}

func buildAPIGatewayRuntimeDerivedApprovalInput(completeness apiGatewayRolloutWiringCompleteness) api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		api.RolloutReportApprovalFlowInput{
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
		},
		api.BuildOwnershipParityApprovalWorkItem(api.OwnershipParityApprovalWorkItemInput{
			State:  completeness.OwnershipState,
			Status: "none",
			Owner:  "none",
		}),
	)
}

func buildAPIGatewaySkeletonGuardedInput() api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		api.RolloutReportGuardedHookInput{
			Hook: api.RolloutReportAction{
				Action: "noop-investigate",
				Reason: apiGatewayRolloutSkeletonReason,
			},
			HookPolicy: api.RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-investigate",
				Reason: apiGatewayRolloutSkeletonReason,
			},
		},
		api.RolloutReportGuardedEnforcementInput{
			WouldEnforce: api.RolloutReportAction{
				Action: "would-investigate",
				Reason: apiGatewayRolloutSkeletonReason,
			},
			EnforceHint: api.RolloutReportStateReason{
				State:  "investigate-before-enforce",
				Reason: apiGatewayRolloutSkeletonReason,
			},
			Overview: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: apiGatewayRolloutSkeletonReason,
			},
		},
	)
}

func buildAPIGatewayRuntimeDerivedGuardedInput(completeness apiGatewayRolloutWiringCompleteness) api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		api.RolloutReportGuardedHookInput{
			Hook: api.RolloutReportAction{
				Action: "noop-hold",
				Reason: completeness.Reason,
			},
			HookPolicy: api.RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-hold",
				Reason: completeness.Reason,
			},
		},
		api.RolloutReportGuardedEnforcementInput{
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
		},
	)
}

func classifyAPIGatewayRolloutWiringCompleteness(runtimeState apiGatewayRolloutRuntimeState, ownershipSource api.RouteOwnershipParitySource) apiGatewayRolloutWiringCompleteness {
	enabled := make([]string, 0, 5)
	missing := make([]string, 0, 5)

	collectAPIGatewayRuntimeSignal(&enabled, &missing, "runtime_routes_enabled", runtimeState.RuntimeRoutesEnabled)
	collectAPIGatewayRuntimeSignal(&enabled, &missing, "event_query_enabled", runtimeState.EventQueryEnabled)
	collectAPIGatewayRuntimeSignal(&enabled, &missing, "event_subscription_enabled", runtimeState.EventSubscriptionEnabled)
	collectAPIGatewayRuntimeSignal(&enabled, &missing, "health_check_enabled", runtimeState.HealthCheckEnabled)
	collectAPIGatewayRuntimeSignal(&enabled, &missing, "domain_bridge_enabled", runtimeState.DomainBridgeEnabled)

	mode := "partially-wired"
	advisoryStatus := "partial-runtime-wiring"
	advisoryReady := false
	if len(missing) == 0 && len(enabled) > 0 {
		mode = "runtime-wired"
		advisoryStatus = "runtime-wired"
		advisoryReady = true
	}

	parts := []string{apiGatewayRolloutRuntimeWiringReason}
	if len(enabled) > 0 {
		parts = append(parts, "enabled: "+strings.Join(enabled, ","))
	}
	if len(missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(missing, ","))
	}
	ownershipState := api.BuildRouteOwnershipParityStateFromSource("api-gateway", buildAPIGatewayOwnershipParitySource(len(enabled) > 0), strings.Split(apiGatewayOwnershipParityReviewFields, ",")...)
	monolithParity := api.RouteOwnershipParitySourceSnapshot{}
	if ownershipSource != nil {
		monolithParity = ownershipSource.SnapshotRouteOwnershipParity()
	}
	parts = api.AppendMonolithOwnershipParityReason(parts, monolithParity)
	parts, ownershipHint := api.AppendRouteOwnershipParityStateReason(parts, ownershipState)
	postureHint := classifyAPIGatewayRolloutPostureHint(mode, advisoryStatus, len(missing))
	if postureHint != "" {
		parts = append(parts, "rollout_posture_hint: "+postureHint)
	}

	return apiGatewayRolloutWiringCompleteness{
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

func buildAPIGatewayOwnershipParitySource(runtimeSignalsPresent bool) api.RouteOwnershipParitySource {
	return api.RouteOwnershipParitySourceFunc(func() api.RouteOwnershipParitySourceSnapshot {
		return api.RouteOwnershipParitySourceSnapshot{
			RuntimeSignalsPresent: runtimeSignalsPresent,
		}
	})
}

func classifyAPIGatewayRolloutPostureHint(mode, advisoryStatus string, missingSignals int) string {
	switch {
	case mode == "partially-wired" && missingSignals > 0:
		return "finish wiring the missing api-gateway runtime routes before treating this rollout as ready"
	case advisoryStatus == "runtime-wired":
		return "observe the runtime-wired api-gateway rollout while local route composition remains healthy"
	default:
		return ""
	}
}

func collectAPIGatewayRuntimeSignal(enabled, missing *[]string, label string, active bool) {
	if active {
		*enabled = append(*enabled, label)
		return
	}
	*missing = append(*missing, label)
}
