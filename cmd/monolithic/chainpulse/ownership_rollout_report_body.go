package main

import "github.com/rtcdance/chainpulse/pkg/plugins/api"

func buildOwnershipRolloutReportBody(details *api.RolloutReportDetails, snapshot ownershipRolloutSummarySnapshot) {
	if details == nil {
		return
	}

	sections := buildOwnershipRolloutReportSections(snapshot)
	applyOwnershipRolloutReportSections(details, sections)
}

//nolint:unused
func buildOwnershipRolloutReportSurfaceSection(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportSurfaceSection {
	return api.BuildRolloutReportSurfaceSection(buildOwnershipRolloutReportSurfaceInput(snapshot))
}

func buildOwnershipRolloutReportSurfaceInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		buildOwnershipRolloutReportSurfaceCoreInput(snapshot),
		buildOwnershipRolloutReportSurfaceCutoverInput(snapshot),
	)
}

func buildOwnershipRolloutReportSurfaceCoreInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportSurfaceCoreInput {
	return api.RolloutReportSurfaceCoreInput{
		Summary: api.RolloutReportSummary{
			ShadowOwnedEvents: snapshot.Summary.ShadowOwnedEvents,
			LegacyOwnedEvents: snapshot.Summary.LegacyOwnedEvents,
			OwnershipChains:   snapshot.Summary.Chains,
		},
		Mode: snapshot.Mode,
		Advisory: api.RolloutReportAdvisory{
			Decision: snapshot.Advisory.Decision,
			Status:   snapshot.Advisory.Status,
			Ready:    snapshot.Advisory.Ready,
			Reason:   snapshot.Advisory.Reason,
		},
		Policy: api.RolloutReportPolicy{
			Mode:         snapshot.Policy.Mode,
			Action:       snapshot.Policy.Action,
			Reason:       snapshot.Policy.Reason,
			Acknowledged: snapshot.Policy.Acknowledged,
			AckState:     snapshot.Policy.AckState,
		},
		Progression: api.RolloutReportStateReason{
			State:  snapshot.Progression.State,
			Reason: snapshot.Progression.Reason,
		},
	}
}

func buildOwnershipRolloutReportSurfaceCutoverInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportSurfaceCutoverInput {
	return api.RolloutReportSurfaceCutoverInput{
		CutoverDryRun: api.RolloutReportAction{
			Action: snapshot.CutoverDryRun.Action,
			Reason: snapshot.CutoverDryRun.Reason,
		},
		CutoverCandidate: api.RolloutReportCandidate{
			Eligible: snapshot.CutoverCandidate.Eligible,
			Reason:   snapshot.CutoverCandidate.Reason,
		},
	}
}

//nolint:unused
func buildOwnershipRolloutReportApprovalSection(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportApproval {
	return api.BuildRolloutReportApprovalSection(buildOwnershipRolloutReportApprovalInput(snapshot))
}

func buildOwnershipRolloutReportApprovalInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		buildOwnershipRolloutReportApprovalFlowInput(snapshot),
		buildOwnershipRolloutReportApprovalWorkItemInput(snapshot),
	)
}

func buildOwnershipRolloutReportApprovalFlowInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportApprovalFlowInput {
	return api.RolloutReportApprovalFlowInput{
		ManualApprovalCheckpoint: api.RolloutReportStateReason{
			State:  snapshot.ManualApprovalCheckpoint.State,
			Reason: snapshot.ManualApprovalCheckpoint.Reason,
		},
		OperatorHandoff: api.RolloutReportStateReason{
			State:  snapshot.OperatorHandoff.State,
			Reason: snapshot.OperatorHandoff.Reason,
		},
		Checklist: api.RolloutReportStateReason{
			State:  snapshot.ApprovalChecklist.State,
			Reason: snapshot.ApprovalChecklist.Reason,
		},
	}
}

func buildOwnershipRolloutReportApprovalWorkItemInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportApprovalWorkItemInput {
	return api.RolloutReportApprovalWorkItemInput{
		WorkItem: api.RolloutReportApprovalItem{
			Status:       snapshot.ApprovalWorkItem.Status,
			Owner:        snapshot.ApprovalWorkItem.Owner,
			ReviewFields: snapshot.ApprovalWorkItem.ReviewFields,
			Reason:       snapshot.ApprovalWorkItem.Reason,
		},
	}
}

//nolint:unused
func buildOwnershipRolloutReportGuardedSection(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportGuarded {
	return api.BuildRolloutReportGuardedSection(buildOwnershipRolloutReportGuardedInput(snapshot))
}

func buildOwnershipRolloutReportGuardedInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		buildOwnershipRolloutReportGuardedHookInput(snapshot),
		buildOwnershipRolloutReportGuardedEnforcementInput(snapshot),
	)
}

func buildOwnershipRolloutReportGuardedHookInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportGuardedHookInput {
	return api.RolloutReportGuardedHookInput{
		Hook: api.RolloutReportAction{
			Action: snapshot.GuardedCutoverHook.Action,
			Reason: snapshot.GuardedCutoverHook.Reason,
		},
		HookPolicy: api.RolloutReportModeAction{
			Mode:   snapshot.GuardedCutoverHookPolicy.Mode,
			Action: snapshot.GuardedCutoverHookPolicy.Action,
			Reason: snapshot.GuardedCutoverHookPolicy.Reason,
		},
	}
}

func buildOwnershipRolloutReportGuardedEnforcementInput(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportGuardedEnforcementInput {
	return api.RolloutReportGuardedEnforcementInput{
		WouldEnforce: api.RolloutReportAction{
			Action: snapshot.GuardedCutoverWouldEnforce.Action,
			Reason: snapshot.GuardedCutoverWouldEnforce.Reason,
		},
		EnforceHint: api.RolloutReportStateReason{
			State:  snapshot.GuardedCutoverEnforceHint.State,
			Reason: snapshot.GuardedCutoverEnforceHint.Reason,
		},
		Overview: api.RolloutReportStateReason{
			State:  snapshot.GuardedCutoverOverview.State,
			Reason: snapshot.GuardedCutoverOverview.Reason,
		},
	}
}

func applyOwnershipRolloutReportSections(details *api.RolloutReportDetails, sections api.RolloutReportSections) {
	api.ApplyRolloutReportSurfaceSection(details, sections.Surface)
	api.ApplyRolloutReportApprovalSection(details, sections.Approval)
	api.ApplyRolloutReportGuardedSection(details, sections.GuardedCutover)
}
