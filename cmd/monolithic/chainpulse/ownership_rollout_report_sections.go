package main

import "github.com/rtcdance/chainpulse/pkg/plugins/api"

func buildOwnershipRolloutReportSections(snapshot ownershipRolloutSummarySnapshot) api.RolloutReportSections {
	return api.BuildRolloutReportSections(api.RolloutReportSectionsInput{
		Surface:        buildOwnershipRolloutReportSurfaceInput(snapshot),
		Approval:       buildOwnershipRolloutReportApprovalInput(snapshot),
		GuardedCutover: buildOwnershipRolloutReportGuardedInput(snapshot),
	})
}
