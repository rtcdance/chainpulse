package main

import (
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

type ownershipRolloutSummarySnapshot struct {
	Summary                    ownershipSummary
	Mode                       string
	Advisory                   ownershipRolloutAdvisory
	Policy                     ownershipRolloutPolicy
	Progression                ownershipEffectiveProgression
	CutoverDryRun              ownershipCutoverDryRun
	CutoverCandidate           ownershipCutoverCandidate
	ManualApprovalCheckpoint   ownershipManualApprovalCheckpoint
	OperatorHandoff            ownershipOperatorHandoff
	ApprovalWorkItem           ownershipApprovalWorkItem
	ApprovalChecklist          ownershipApprovalChecklist
	GuardedCutoverHook         ownershipGuardedCutoverHook
	GuardedCutoverHookPolicy   ownershipGuardedCutoverHookPolicy
	GuardedCutoverWouldEnforce ownershipGuardedCutoverWouldEnforce
	GuardedCutoverEnforceHint  ownershipGuardedCutoverEnforceHint
	GuardedCutoverOverview     ownershipGuardedCutoverOverview
}

func buildOwnershipRolloutSummary(status map[string]map[string]any) ownershipRolloutSummarySnapshot {
	summary := aggregateIndexerOwnership(status)
	mode := classifyOwnershipMode(summary)
	advisory := classifyOwnershipRolloutAdvisory(summary)
	policy := resolveOwnershipRolloutPolicyFromEnv(advisory)
	progression := classifyOwnershipEffectiveProgression(advisory, policy)
	cutoverDryRun := classifyOwnershipCutoverDryRun(progression)
	cutoverCandidate := classifyOwnershipCutoverCandidate(policy, progression, cutoverDryRun)
	sections := buildOwnershipRolloutSummarySections(
		summary,
		mode,
		advisory,
		policy,
		progression,
		cutoverDryRun,
		cutoverCandidate,
	)

	return ownershipRolloutSummarySnapshot{
		Summary:                    sections.Surface.Summary,
		Mode:                       sections.Surface.Mode,
		Advisory:                   sections.Surface.Advisory,
		Policy:                     sections.Surface.Policy,
		Progression:                sections.Surface.Progression,
		CutoverDryRun:              sections.Surface.CutoverDryRun,
		CutoverCandidate:           sections.Surface.CutoverCandidate,
		ManualApprovalCheckpoint:   sections.Approval.ManualApprovalCheckpoint,
		OperatorHandoff:            sections.Approval.OperatorHandoff,
		ApprovalWorkItem:           sections.Approval.ApprovalWorkItem,
		ApprovalChecklist:          sections.Approval.ApprovalChecklist,
		GuardedCutoverHook:         sections.Guarded.Hook,
		GuardedCutoverHookPolicy:   sections.Guarded.HookPolicy,
		GuardedCutoverWouldEnforce: sections.Guarded.WouldEnforce,
		GuardedCutoverEnforceHint:  sections.Guarded.EnforceHint,
		GuardedCutoverOverview:     sections.Guarded.Overview,
	}
}

func (snapshot ownershipRolloutSummarySnapshot) readinessDetails() map[string]any {
	details := map[string]any{}

	ownershipRolloutSurface{
		Summary:          snapshot.Summary,
		Mode:             snapshot.Mode,
		Advisory:         snapshot.Advisory,
		Policy:           snapshot.Policy,
		Progression:      snapshot.Progression,
		CutoverDryRun:    snapshot.CutoverDryRun,
		CutoverCandidate: snapshot.CutoverCandidate,
	}.applyReadinessDetails(details)

	ownershipApprovalSummary{
		ManualApprovalCheckpoint: snapshot.ManualApprovalCheckpoint,
		OperatorHandoff:          snapshot.OperatorHandoff,
		ApprovalWorkItem:         snapshot.ApprovalWorkItem,
		ApprovalChecklist:        snapshot.ApprovalChecklist,
	}.applyReadinessDetails(details)

	ownershipGuardedCutoverSummary{
		Hook:         snapshot.GuardedCutoverHook,
		HookPolicy:   snapshot.GuardedCutoverHookPolicy,
		WouldEnforce: snapshot.GuardedCutoverWouldEnforce,
		EnforceHint:  snapshot.GuardedCutoverEnforceHint,
		Overview:     snapshot.GuardedCutoverOverview,
	}.applyReadinessDetails(details)

	return details
}

func (snapshot ownershipRolloutSummarySnapshot) reportDetails() *api.RolloutReportDetails {
	details := api.NewRolloutReportDetailsFromMetadata(api.RolloutReportMetadata{
		ReportID:       "monolithic-ownership-rollout-runtime",
		SchemaFamily:   api.OwnershipRolloutSchemaFamily,
		ReportVersion:  api.OwnershipRolloutReportVersion,
		Service:        "monolithic",
		ReportScope:    api.OwnershipRolloutReportScope,
		ReportSource:   "monolithic",
		ReportMode:     api.OwnershipRolloutReportMode,
		DeploymentMode: "monolithic",
		GeneratedAt:    time.Now().Unix(),
	})

	buildOwnershipRolloutReportBody(details, snapshot)

	return details
}

func emitOwnershipRolloutSummaryMetrics(metrics core.MetricsCollector, snapshot ownershipRolloutSummarySnapshot, operation string) {
	if metrics == nil {
		return
	}

	tags := map[string]string{
		"service":   "monolithic",
		"operation": operation,
	}
	ownershipRolloutSurface{
		Summary:          snapshot.Summary,
		Mode:             snapshot.Mode,
		Advisory:         snapshot.Advisory,
		Policy:           snapshot.Policy,
		Progression:      snapshot.Progression,
		CutoverDryRun:    snapshot.CutoverDryRun,
		CutoverCandidate: snapshot.CutoverCandidate,
	}.emitMetrics(metrics, tags)
	ownershipApprovalSummary{
		ManualApprovalCheckpoint: snapshot.ManualApprovalCheckpoint,
		OperatorHandoff:          snapshot.OperatorHandoff,
		ApprovalWorkItem:         snapshot.ApprovalWorkItem,
		ApprovalChecklist:        snapshot.ApprovalChecklist,
	}.emitMetrics(metrics, tags)
	ownershipGuardedCutoverSummary{
		Hook:         snapshot.GuardedCutoverHook,
		HookPolicy:   snapshot.GuardedCutoverHookPolicy,
		WouldEnforce: snapshot.GuardedCutoverWouldEnforce,
		EnforceHint:  snapshot.GuardedCutoverEnforceHint,
		Overview:     snapshot.GuardedCutoverOverview,
	}.emitMetrics(metrics, tags)
}
