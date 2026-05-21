package main

import "github.com/rtcdance/chainpulse/pkg/core"

type ownershipGuardedCutoverSummary struct {
	Hook         ownershipGuardedCutoverHook
	HookPolicy   ownershipGuardedCutoverHookPolicy
	WouldEnforce ownershipGuardedCutoverWouldEnforce
	EnforceHint  ownershipGuardedCutoverEnforceHint
	Overview     ownershipGuardedCutoverOverview
}

func buildOwnershipGuardedCutoverSummary(
	cutoverDryRun ownershipCutoverDryRun,
	cutoverCandidate ownershipCutoverCandidate,
	approvalChecklist ownershipApprovalChecklist,
) ownershipGuardedCutoverSummary {
	hook := classifyOwnershipGuardedCutoverHook(cutoverDryRun, cutoverCandidate, approvalChecklist)
	hookPolicy := resolveOwnershipGuardedCutoverHookPolicyFromEnv(hook)
	wouldEnforce := classifyOwnershipGuardedCutoverWouldEnforce(hook, hookPolicy)
	enforceHint := classifyOwnershipGuardedCutoverEnforceHint(wouldEnforce)
	overview := classifyOwnershipGuardedCutoverOverview(wouldEnforce, enforceHint)

	return ownershipGuardedCutoverSummary{
		Hook:         hook,
		HookPolicy:   hookPolicy,
		WouldEnforce: wouldEnforce,
		EnforceHint:  enforceHint,
		Overview:     overview,
	}
}

func (summary ownershipGuardedCutoverSummary) applyReadinessDetails(details map[string]any) {
	details["rollout_guarded_cutover_hook_action"] = summary.Hook.Action
	details["rollout_guarded_cutover_hook_reason"] = summary.Hook.Reason
	details["rollout_guarded_cutover_hook_policy_mode"] = summary.HookPolicy.Mode
	details["rollout_guarded_cutover_hook_policy_action"] = summary.HookPolicy.Action
	details["rollout_guarded_cutover_hook_policy_reason"] = summary.HookPolicy.Reason
	details["rollout_guarded_cutover_would_enforce_action"] = summary.WouldEnforce.Action
	details["rollout_guarded_cutover_would_enforce_reason"] = summary.WouldEnforce.Reason
	details["rollout_guarded_cutover_enforce_hint_state"] = summary.EnforceHint.State
	details["rollout_guarded_cutover_enforce_hint_reason"] = summary.EnforceHint.Reason
	details["rollout_guarded_cutover_overview_state"] = summary.Overview.State
	details["rollout_guarded_cutover_overview_reason"] = summary.Overview.Reason
}

func (summary ownershipGuardedCutoverSummary) emitMetrics(metrics core.MetricsCollector, tags map[string]string) {
	if metrics == nil {
		return
	}

	metrics.RecordGauge("indexing_runtime_rollout_guarded_cutover_hook_code", ownershipGuardedCutoverHookCode(summary.Hook), tags)
	metrics.RecordGauge("indexing_runtime_rollout_guarded_cutover_hook_policy_mode_code", ownershipGuardedCutoverHookPolicyModeCode(summary.HookPolicy.Mode), tags)
	metrics.RecordGauge("indexing_runtime_rollout_guarded_cutover_would_enforce_code", ownershipGuardedCutoverWouldEnforceCode(summary.WouldEnforce), tags)
	metrics.RecordGauge("indexing_runtime_rollout_guarded_cutover_enforce_hint_code", ownershipGuardedCutoverEnforceHintCode(summary.EnforceHint), tags)
	metrics.RecordGauge("indexing_runtime_rollout_guarded_cutover_overview_code", ownershipGuardedCutoverOverviewCode(summary.Overview), tags)
}
