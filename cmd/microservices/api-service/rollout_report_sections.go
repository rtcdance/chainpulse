package main

import "github.com/rtcdance/chainpulse/pkg/plugins/api"

func buildAPIServiceSkeletonSections() api.RolloutReportSections {
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildAPIServiceSkeletonSurfaceInput(),
		Approval:       buildAPIServiceSkeletonApprovalInput(),
		GuardedCutover: buildAPIServiceSkeletonGuardedInput(),
	})
}

func buildAPIServiceRuntimeDerivedSections(runtimeState apiServiceRolloutRuntimeState, ownershipSource api.RouteOwnershipParitySource) api.RolloutReportSections {
	completeness := classifyAPIServiceRolloutWiringCompleteness(runtimeState, ownershipSource)
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildAPIServiceRuntimeDerivedSurfaceInput(completeness),
		Approval:       buildAPIServiceRuntimeDerivedApprovalInput(completeness),
		GuardedCutover: buildAPIServiceRuntimeDerivedGuardedInput(completeness),
	})
}

func applyAPIServiceRolloutReportSections(details *api.RolloutReportDetails, sections api.RolloutReportSections) {
	if details == nil {
		return
	}

	api.ApplyRolloutReportSurfaceSection(details, sections.Surface)
	api.ApplyRolloutReportApprovalSection(details, sections.Approval)
	api.ApplyRolloutReportGuardedSection(details, sections.GuardedCutover)
}
