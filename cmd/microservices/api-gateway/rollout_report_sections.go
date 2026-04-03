package main

import "chainpulse/pkg/plugins/api"

func buildAPIGatewaySkeletonSections() api.RolloutReportSections {
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildAPIGatewaySkeletonSurfaceInput(),
		Approval:       buildAPIGatewaySkeletonApprovalInput(),
		GuardedCutover: buildAPIGatewaySkeletonGuardedInput(),
	})
}

func buildAPIGatewayRuntimeDerivedSections(runtimeState apiGatewayRolloutRuntimeState, ownershipSource api.RouteOwnershipParitySource) api.RolloutReportSections {
	completeness := classifyAPIGatewayRolloutWiringCompleteness(runtimeState, ownershipSource)
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildAPIGatewayRuntimeDerivedSurfaceInput(completeness),
		Approval:       buildAPIGatewayRuntimeDerivedApprovalInput(completeness),
		GuardedCutover: buildAPIGatewayRuntimeDerivedGuardedInput(completeness),
	})
}

func applyAPIGatewayRolloutReportSections(details *api.RolloutReportDetails, sections api.RolloutReportSections) {
	if details == nil {
		return
	}

	api.ApplyRolloutReportSurfaceSection(details, sections.Surface)
	api.ApplyRolloutReportApprovalSection(details, sections.Approval)
	api.ApplyRolloutReportGuardedSection(details, sections.GuardedCutover)
}
