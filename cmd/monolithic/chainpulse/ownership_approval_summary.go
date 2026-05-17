package main

import "chainpulse/pkg/core"

type ownershipApprovalSummary struct {
	ManualApprovalCheckpoint ownershipManualApprovalCheckpoint
	OperatorHandoff          ownershipOperatorHandoff
	ApprovalWorkItem         ownershipApprovalWorkItem
	ApprovalChecklist        ownershipApprovalChecklist
}

func buildOwnershipApprovalSummary(
	progression ownershipEffectiveProgression,
	cutoverCandidate ownershipCutoverCandidate,
) ownershipApprovalSummary {
	manualApprovalCheckpoint := classifyOwnershipManualApprovalCheckpoint(progression, cutoverCandidate)
	operatorHandoff := classifyOwnershipOperatorHandoff(manualApprovalCheckpoint)
	approvalWorkItem := classifyOwnershipApprovalWorkItem(operatorHandoff)
	approvalChecklist := classifyOwnershipApprovalChecklist(cutoverCandidate, manualApprovalCheckpoint, operatorHandoff, approvalWorkItem)

	return ownershipApprovalSummary{
		ManualApprovalCheckpoint: manualApprovalCheckpoint,
		OperatorHandoff:          operatorHandoff,
		ApprovalWorkItem:         approvalWorkItem,
		ApprovalChecklist:        approvalChecklist,
	}
}

func (summary ownershipApprovalSummary) applyReadinessDetails(details map[string]any) {
	details["rollout_manual_approval_checkpoint_state"] = summary.ManualApprovalCheckpoint.State
	details["rollout_manual_approval_checkpoint_reason"] = summary.ManualApprovalCheckpoint.Reason
	details["rollout_operator_handoff_state"] = summary.OperatorHandoff.State
	details["rollout_operator_handoff_reason"] = summary.OperatorHandoff.Reason
	details["rollout_approval_work_item_status"] = summary.ApprovalWorkItem.Status
	details["rollout_approval_work_item_owner"] = summary.ApprovalWorkItem.Owner
	details["rollout_approval_work_item_review_fields"] = summary.ApprovalWorkItem.ReviewFields
	details["rollout_approval_work_item_reason"] = summary.ApprovalWorkItem.Reason
	details["rollout_approval_checklist_state"] = summary.ApprovalChecklist.State
	details["rollout_approval_checklist_reason"] = summary.ApprovalChecklist.Reason
}

func (summary ownershipApprovalSummary) emitMetrics(metrics core.MetricsCollector, tags map[string]string) {
	if metrics == nil {
		return
	}

	metrics.RecordGauge("indexing_runtime_rollout_manual_approval_checkpoint_code", ownershipManualApprovalCheckpointCode(summary.ManualApprovalCheckpoint), tags)
	metrics.RecordGauge("indexing_runtime_rollout_operator_handoff_code", ownershipOperatorHandoffCode(summary.OperatorHandoff), tags)
	metrics.RecordGauge("indexing_runtime_rollout_approval_work_item_code", ownershipApprovalWorkItemCode(summary.ApprovalWorkItem), tags)
	metrics.RecordGauge("indexing_runtime_rollout_approval_checklist_code", ownershipApprovalChecklistCode(summary.ApprovalChecklist), tags)
}
