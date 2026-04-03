package main

type ownershipRolloutSummarySections struct {
	Surface  ownershipRolloutSurface
	Approval ownershipApprovalSummary
	Guarded  ownershipGuardedCutoverSummary
}

func buildOwnershipRolloutSummarySections(
	summary ownershipSummary,
	mode string,
	advisory ownershipRolloutAdvisory,
	policy ownershipRolloutPolicy,
	progression ownershipEffectiveProgression,
	cutoverDryRun ownershipCutoverDryRun,
	cutoverCandidate ownershipCutoverCandidate,
) ownershipRolloutSummarySections {
	surface := ownershipRolloutSurface{
		Summary:          summary,
		Mode:             mode,
		Advisory:         advisory,
		Policy:           policy,
		Progression:      progression,
		CutoverDryRun:    cutoverDryRun,
		CutoverCandidate: cutoverCandidate,
	}
	approval := buildOwnershipApprovalSummary(progression, cutoverCandidate)
	guarded := buildOwnershipGuardedCutoverSummary(cutoverDryRun, cutoverCandidate, approval.ApprovalChecklist)

	return ownershipRolloutSummarySections{
		Surface:  surface,
		Approval: approval,
		Guarded:  guarded,
	}
}
