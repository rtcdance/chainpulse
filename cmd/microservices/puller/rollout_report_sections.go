package main

import "github.com/rtcdance/chainpulse/pkg/plugins/api"

func buildPullerSkeletonSections() api.RolloutReportSections {
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildPullerSkeletonSurfaceInput(),
		Approval:       buildPullerSkeletonApprovalInput(),
		GuardedCutover: buildPullerSkeletonGuardedInput(),
	})
}

func buildPullerRuntimeDerivedSections(runtimeState pullerRolloutRuntimeState) api.RolloutReportSections {
	completeness := classifyPullerRolloutWiringCompleteness(runtimeState)
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildPullerRuntimeDerivedSurfaceInput(completeness),
		Approval:       buildPullerRuntimeDerivedApprovalInput(completeness),
		GuardedCutover: buildPullerRuntimeDerivedGuardedInput(completeness),
	})
}

func applyPullerRolloutReportSections(details *api.RolloutReportDetails, sections api.RolloutReportSections) {
	if details == nil {
		return
	}

	api.ApplyRolloutReportSurfaceSection(details, sections.Surface)
	api.ApplyRolloutReportApprovalSection(details, sections.Approval)
	api.ApplyRolloutReportGuardedSection(details, sections.GuardedCutover)
}
