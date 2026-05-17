package main

import "github.com/rtcdance/chainpulse/pkg/core"

type ownershipRolloutSurface struct {
	Summary          ownershipSummary
	Mode             string
	Advisory         ownershipRolloutAdvisory
	Policy           ownershipRolloutPolicy
	Progression      ownershipEffectiveProgression
	CutoverDryRun    ownershipCutoverDryRun
	CutoverCandidate ownershipCutoverCandidate
}

func (surface ownershipRolloutSurface) applyReadinessDetails(details map[string]any) {
	details["service"] = "monolithic"
	details["ownership_mode"] = surface.Mode
	details["rollout_ready_for_runtime_owned"] = surface.Advisory.Ready
	details["rollout_status"] = surface.Advisory.Status
	details["rollout_reason"] = surface.Advisory.Reason
	details["rollout_gate_decision"] = surface.Advisory.Decision
	details["rollout_gate_reason"] = surface.Advisory.Reason
	details["rollout_policy_mode"] = surface.Policy.Mode
	details["rollout_policy_action"] = surface.Policy.Action
	details["rollout_policy_reason"] = surface.Policy.Reason
	details["rollout_policy_acknowledged"] = surface.Policy.Acknowledged
	details["rollout_policy_ack_state"] = surface.Policy.AckState
	details["rollout_effective_state"] = surface.Progression.State
	details["rollout_effective_reason"] = surface.Progression.Reason
	details["rollout_cutover_dry_run_action"] = surface.CutoverDryRun.Action
	details["rollout_cutover_dry_run_reason"] = surface.CutoverDryRun.Reason
	details["rollout_cutover_candidate"] = surface.CutoverCandidate.Eligible
	details["rollout_cutover_candidate_reason"] = surface.CutoverCandidate.Reason
	details["shadow_owned_events"] = surface.Summary.ShadowOwnedEvents
	details["legacy_owned_events"] = surface.Summary.LegacyOwnedEvents
	details["ownership_chains"] = surface.Summary.Chains
}

func (surface ownershipRolloutSurface) emitMetrics(metrics core.MetricsCollector, tags map[string]string) {
	if metrics == nil {
		return
	}

	metrics.RecordGauge("indexing_runtime_shadow_owned_events", float64(surface.Summary.ShadowOwnedEvents), tags)
	metrics.RecordGauge("indexing_runtime_legacy_owned_events", float64(surface.Summary.LegacyOwnedEvents), tags)
	metrics.RecordGauge("indexing_runtime_ownership_chains", float64(surface.Summary.Chains), tags)
	metrics.RecordGauge("indexing_runtime_ownership_mode_code", ownershipModeCode(surface.Mode), tags)
	metrics.RecordGauge("indexing_runtime_rollout_policy_mode_code", ownershipPolicyModeCode(surface.Policy.Mode), tags)
	metrics.RecordGauge("indexing_runtime_rollout_policy_ack_state_code", ownershipAckStateCode(surface.Policy.AckState), tags)
	metrics.RecordGauge("indexing_runtime_rollout_effective_state_code", ownershipEffectiveProgressionCode(surface.Progression.State), tags)
	metrics.RecordGauge("indexing_runtime_rollout_cutover_dry_run_code", ownershipCutoverDryRunCode(surface.CutoverDryRun.Action), tags)
	metrics.RecordGauge("indexing_runtime_rollout_cutover_candidate", ownershipCutoverCandidateCode(surface.CutoverCandidate), tags)
}
