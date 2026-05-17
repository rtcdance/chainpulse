package main

import "github.com/rtcdance/chainpulse/pkg/plugins/api"

func buildEventProcessorSkeletonSections() api.RolloutReportSections {
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildEventProcessorSkeletonSurfaceInput(),
		Approval:       buildEventProcessorSkeletonApprovalInput(),
		GuardedCutover: buildEventProcessorSkeletonGuardedInput(),
	})
}

func buildEventProcessorRuntimeDerivedSections(runtimeState eventProcessorRolloutRuntimeState) api.RolloutReportSections {
	completeness := classifyEventProcessorRolloutWiringCompleteness(runtimeState)
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildEventProcessorRuntimeDerivedSurfaceInput(completeness),
		Approval:       buildEventProcessorRuntimeDerivedApprovalInput(completeness),
		GuardedCutover: buildEventProcessorRuntimeDerivedGuardedInput(completeness),
	})
}

func applyEventProcessorRolloutReportSections(details *api.RolloutReportDetails, sections api.RolloutReportSections) {
	if details == nil {
		return
	}

	api.ApplyRolloutReportSurfaceSection(details, sections.Surface)
	api.ApplyRolloutReportApprovalSection(details, sections.Approval)
	api.ApplyRolloutReportGuardedSection(details, sections.GuardedCutover)
}
