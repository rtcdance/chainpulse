package rollout

import "testing"

func TestValidateMicroserviceRolloutMetadataParity(t *testing.T) {
	t.Parallel()
	report := NewRolloutReportDetailsFromMetadata(NewOwnershipRolloutReportMetadata(
		"api-service",
		"api-service-ownership-rollout-runtime",
		"microservice:api-service-1",
		"microservice",
		1700000000,
	))

	if err := ValidateMicroserviceRolloutMetadataParity(
		report,
		"api-service",
		"api-service-ownership-rollout-runtime",
		"microservice:api-service-1",
	); err != nil {
		t.Fatalf("expected metadata parity validation to succeed: %v", err)
	}
}

func TestValidateMicroserviceRuntimeDerivedRolloutParity(t *testing.T) {
	t.Parallel()
	report := &RolloutReportDetails{
		Summary: RolloutReportSummary{
			ShadowOwnedEvents: 0,
			LegacyOwnedEvents: 0,
			OwnershipChains:   0,
		},
		Advisory: RolloutReportAdvisory{
			Decision: "hold",
		},
		Policy: RolloutReportPolicy{
			Mode:         "report-only",
			Action:       "report-hold",
			Acknowledged: false,
			AckState:     "pending",
		},
		Progression: RolloutReportStateReason{
			State: "observe",
		},
		CutoverDryRun: RolloutReportAction{
			Action: "would-hold",
		},
		CutoverCandidate: RolloutReportCandidate{
			Eligible: false,
		},
		Approval: RolloutReportApproval{
			ManualApprovalCheckpoint: RolloutReportStateReason{State: "inactive"},
			OperatorHandoff:          RolloutReportStateReason{State: "none"},
			WorkItem:                 RolloutReportApprovalItem{Status: "none"},
			Checklist:                RolloutReportStateReason{State: "incomplete"},
		},
		GuardedCutover: RolloutReportGuarded{
			Hook:         RolloutReportAction{Action: "noop-hold"},
			HookPolicy:   RolloutReportModeAction{Mode: "noop-only", Action: "noop-hold"},
			WouldEnforce: RolloutReportAction{Action: "would-hold"},
			EnforceHint:  RolloutReportStateReason{State: "hold-before-enforce"},
			Overview:     RolloutReportStateReason{State: "hold"},
		},
	}

	if err := ValidateMicroserviceRuntimeDerivedRolloutParity(report); err != nil {
		t.Fatalf("expected runtime parity validation to succeed: %v", err)
	}
}

func TestValidateMicroserviceOwnershipParityMarker(t *testing.T) {
	t.Parallel()
	report := &RolloutReportDetails{
		Advisory: RolloutReportAdvisory{
			Reason: "enabled: runtime_routes_enabled; ownership_parity_hint: api-service runtime wiring is present, but ownership runtime parity with monolith is still pending",
		},
		Approval: RolloutReportApproval{
			WorkItem: RolloutReportApprovalItem{
				ReviewFields: "runtime_routes_enabled,ownership_runtime_parity",
				Reason:       "api-service runtime wiring is present, but ownership runtime parity with monolith is still pending",
			},
		},
	}

	if err := ValidateMicroserviceOwnershipParityMarker(report); err != nil {
		t.Fatalf("expected ownership parity marker validation to succeed: %v", err)
	}
}

func TestValidateRouteMonolithOwnershipParityReason(t *testing.T) {
	t.Parallel()
	report := &RolloutReportDetails{
		Advisory: RolloutReportAdvisory{
			Reason: "enabled: runtime_routes_enabled; monolith_parity_posture: monolith-shadow-observe; monolith_parity_hint: monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet; monolith_parity_target_decision: target-shadow; monolith_parity_action_guidance: keep route parity in observe mode until the monolith exits shadow posture",
		},
	}

	if err := ValidateRouteMonolithOwnershipParityReason(
		report,
		"monolith-shadow-observe",
		"monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet",
		"target-shadow",
		"keep route parity in observe mode until the monolith exits shadow posture",
	); err != nil {
		t.Fatalf("expected monolith parity reason validation to succeed: %v", err)
	}
}

func TestValidateRouteMonolithOwnershipParityRecommendationBundle(t *testing.T) {
	t.Parallel()
	report := &RolloutReportDetails{
		Advisory: RolloutReportAdvisory{
			Reason: "enabled: runtime_routes_enabled; monolith_parity_posture: monolith-shadow-observe; monolith_parity_hint: monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet; monolith_parity_target_decision: target-shadow; monolith_parity_action_guidance: keep route parity in observe mode until the monolith exits shadow posture",
		},
	}

	if err := ValidateRouteMonolithOwnershipParityRecommendationBundle(report, MonolithOwnershipParityRecommendationBundle{
		Posture:        "monolith-shadow-observe",
		Hint:           "monolith ownership rollout is still in shadow observe posture; do not treat route parity as complete yet",
		TargetDecision: "target-shadow",
		ActionGuidance: "keep route parity in observe mode until the monolith exits shadow posture",
	}); err != nil {
		t.Fatalf("expected recommendation bundle validation to succeed: %v", err)
	}
}
