package main

import (
	"fmt"
	"io"

	"chainpulse/pkg/core"
)

type ownershipRolloutPresenterLine struct {
	runningLabel  string
	shutdownLabel string
	value         func(ownershipRolloutSummarySnapshot) string
}

type ownershipRolloutLogDescriptor struct {
	message string
	fields  func(string, ownershipRolloutSummarySnapshot) []interface{}
}

type ownershipRolloutValueAccessor func(ownershipRolloutSummarySnapshot) string

type ownershipRolloutPresenterSections struct {
	Ownership []ownershipRolloutPresenterLine
	Approval  []ownershipRolloutPresenterLine
	Guarded   []ownershipRolloutPresenterLine
}

type ownershipRolloutLogSections struct {
	Ownership []ownershipRolloutLogDescriptor
	Approval  []ownershipRolloutLogDescriptor
	Guarded   []ownershipRolloutLogDescriptor
}

type ownershipRolloutOwnershipAccessors struct {
	ShadowOwnedEvents      ownershipRolloutValueAccessor
	LegacyOwnedEvents      ownershipRolloutValueAccessor
	OwnershipChains        ownershipRolloutValueAccessor
	Mode                   ownershipRolloutValueAccessor
	ProgressionState       ownershipRolloutValueAccessor
	ProgressionReason      ownershipRolloutValueAccessor
	CutoverDryRunAction    ownershipRolloutValueAccessor
	CutoverDryRunReason    ownershipRolloutValueAccessor
	CutoverCandidate       ownershipRolloutValueAccessor
	CutoverCandidateReason ownershipRolloutValueAccessor
}

type ownershipRolloutApprovalAccessors struct {
	ManualApprovalCheckpointState  ownershipRolloutValueAccessor
	ManualApprovalCheckpointReason ownershipRolloutValueAccessor
	OperatorHandoffState           ownershipRolloutValueAccessor
	OperatorHandoffReason          ownershipRolloutValueAccessor
	ApprovalWorkItemStatus         ownershipRolloutValueAccessor
	ApprovalWorkItemOwner          ownershipRolloutValueAccessor
	ApprovalWorkItemReviewFields   ownershipRolloutValueAccessor
	ApprovalWorkItemReason         ownershipRolloutValueAccessor
	ApprovalChecklistState         ownershipRolloutValueAccessor
	ApprovalChecklistReason        ownershipRolloutValueAccessor
}

type ownershipRolloutGuardedAccessors struct {
	GuardedCutoverHookAction         ownershipRolloutValueAccessor
	GuardedCutoverHookReason         ownershipRolloutValueAccessor
	GuardedCutoverHookPolicyMode     ownershipRolloutValueAccessor
	GuardedCutoverHookPolicyAction   ownershipRolloutValueAccessor
	GuardedCutoverHookPolicyReason   ownershipRolloutValueAccessor
	GuardedCutoverWouldEnforceAction ownershipRolloutValueAccessor
	GuardedCutoverWouldEnforceReason ownershipRolloutValueAccessor
	GuardedCutoverEnforceHintState   ownershipRolloutValueAccessor
	GuardedCutoverEnforceHintReason  ownershipRolloutValueAccessor
	GuardedCutoverOverviewState      ownershipRolloutValueAccessor
	GuardedCutoverOverviewReason     ownershipRolloutValueAccessor
}

type ownershipRolloutPresenterAccessors struct {
	Ownership ownershipRolloutOwnershipAccessors
	Approval  ownershipRolloutApprovalAccessors
	Guarded   ownershipRolloutGuardedAccessors
}

func logOwnershipRolloutSummary(logger core.Logger, phase string, summary ownershipRolloutSummarySnapshot) {
	for _, descriptor := range ownershipRolloutLogDescriptors() {
		logger.Info(descriptor.message, descriptor.fields(phase, summary)...)
	}
}

func printOwnershipRolloutSummary(w io.Writer, summary ownershipRolloutSummarySnapshot, lifecycle string) {
	prefix := ownershipRolloutPresenterPrefix(lifecycle)
	for _, line := range ownershipRolloutPresenterLines() {
		_, _ = fmt.Fprintf(w, "%s%s: %s\n", prefix, ownershipRolloutPresenterLabel(line, lifecycle), line.value(summary))
	}
}

func ownershipRolloutPresenterPrefix(lifecycle string) string {
	if lifecycle == "shutdown" {
		return "  "
	}
	return ""
}

func ownershipRolloutPresenterLabel(line ownershipRolloutPresenterLine, lifecycle string) string {
	if lifecycle == "shutdown" {
		return line.shutdownLabel
	}
	return line.runningLabel
}

func ownershipRolloutPresenterLines() []ownershipRolloutPresenterLine {
	sections := buildOwnershipRolloutPresenterSections()
	lines := []ownershipRolloutPresenterLine{}
	lines = append(lines, sections.Ownership...)
	lines = append(lines, sections.Approval...)
	lines = append(lines, sections.Guarded...)
	return lines
}

func buildOwnershipRolloutPresenterSections() ownershipRolloutPresenterSections {
	return ownershipRolloutPresenterSections{
		Ownership: ownershipRolloutOwnershipPresenterLines(),
		Approval:  ownershipRolloutApprovalPresenterLines(),
		Guarded:   ownershipRolloutGuardedCutoverPresenterLines(),
	}
}

func ownershipRolloutOwnershipPresenterLines() []ownershipRolloutPresenterLine {
	accessors := buildOwnershipRolloutPresenterAccessors().Ownership
	return []ownershipRolloutPresenterLine{
		{
			runningLabel:  "Shadow-Owned Events",
			shutdownLabel: "Shadow-owned events",
			value:         accessors.ShadowOwnedEvents,
		},
		{
			runningLabel:  "Legacy-Owned Events",
			shutdownLabel: "Legacy-owned events",
			value:         accessors.LegacyOwnedEvents,
		},
		{
			runningLabel:  "Ownership Chains",
			shutdownLabel: "Ownership chains",
			value:         accessors.OwnershipChains,
		},
		{
			runningLabel:  "Ownership Mode",
			shutdownLabel: "Ownership mode",
			value:         accessors.Mode,
		},
		{
			runningLabel:  "Rollout Progression",
			shutdownLabel: "Rollout progression",
			value:         accessors.ProgressionState,
		},
		{
			runningLabel:  "Rollout Progression Reason",
			shutdownLabel: "Rollout progression reason",
			value:         accessors.ProgressionReason,
		},
		{
			runningLabel:  "Cutover Dry-Run",
			shutdownLabel: "Cutover dry-run",
			value:         accessors.CutoverDryRunAction,
		},
		{
			runningLabel:  "Cutover Dry-Run Reason",
			shutdownLabel: "Cutover dry-run reason",
			value:         accessors.CutoverDryRunReason,
		},
		{
			runningLabel:  "Cutover Candidate",
			shutdownLabel: "Cutover candidate",
			value:         accessors.CutoverCandidate,
		},
		{
			runningLabel:  "Cutover Candidate Reason",
			shutdownLabel: "Cutover candidate reason",
			value:         accessors.CutoverCandidateReason,
		},
	}
}

func ownershipRolloutApprovalPresenterLines() []ownershipRolloutPresenterLine {
	accessors := buildOwnershipRolloutPresenterAccessors().Approval
	return []ownershipRolloutPresenterLine{
		{
			runningLabel:  "Manual Approval Checkpoint",
			shutdownLabel: "Manual approval checkpoint",
			value:         accessors.ManualApprovalCheckpointState,
		},
		{
			runningLabel:  "Manual Approval Checkpoint Reason",
			shutdownLabel: "Manual approval checkpoint reason",
			value:         accessors.ManualApprovalCheckpointReason,
		},
		{
			runningLabel:  "Operator Handoff",
			shutdownLabel: "Operator handoff",
			value:         accessors.OperatorHandoffState,
		},
		{
			runningLabel:  "Operator Handoff Reason",
			shutdownLabel: "Operator handoff reason",
			value:         accessors.OperatorHandoffReason,
		},
		{
			runningLabel:  "Approval Work Item",
			shutdownLabel: "Approval work item",
			value:         accessors.ApprovalWorkItemStatus,
		},
		{
			runningLabel:  "Approval Work Item Owner",
			shutdownLabel: "Approval work item owner",
			value:         accessors.ApprovalWorkItemOwner,
		},
		{
			runningLabel:  "Approval Work Item Review Fields",
			shutdownLabel: "Approval work item review fields",
			value:         accessors.ApprovalWorkItemReviewFields,
		},
		{
			runningLabel:  "Approval Work Item Reason",
			shutdownLabel: "Approval work item reason",
			value:         accessors.ApprovalWorkItemReason,
		},
		{
			runningLabel:  "Approval Checklist",
			shutdownLabel: "Approval checklist",
			value:         accessors.ApprovalChecklistState,
		},
		{
			runningLabel:  "Approval Checklist Reason",
			shutdownLabel: "Approval checklist reason",
			value:         accessors.ApprovalChecklistReason,
		},
	}
}

func ownershipRolloutGuardedCutoverPresenterLines() []ownershipRolloutPresenterLine {
	accessors := buildOwnershipRolloutPresenterAccessors().Guarded
	return []ownershipRolloutPresenterLine{
		{
			runningLabel:  "Guarded Cutover Hook",
			shutdownLabel: "Guarded cutover hook",
			value:         accessors.GuardedCutoverHookAction,
		},
		{
			runningLabel:  "Guarded Cutover Hook Reason",
			shutdownLabel: "Guarded cutover hook reason",
			value:         accessors.GuardedCutoverHookReason,
		},
		{
			runningLabel:  "Guarded Cutover Hook Policy Mode",
			shutdownLabel: "Guarded cutover hook policy mode",
			value:         accessors.GuardedCutoverHookPolicyMode,
		},
		{
			runningLabel:  "Guarded Cutover Hook Policy Action",
			shutdownLabel: "Guarded cutover hook policy action",
			value:         accessors.GuardedCutoverHookPolicyAction,
		},
		{
			runningLabel:  "Guarded Cutover Hook Policy Reason",
			shutdownLabel: "Guarded cutover hook policy reason",
			value:         accessors.GuardedCutoverHookPolicyReason,
		},
		{
			runningLabel:  "Guarded Cutover Would-Enforce",
			shutdownLabel: "Guarded cutover would-enforce",
			value:         accessors.GuardedCutoverWouldEnforceAction,
		},
		{
			runningLabel:  "Guarded Cutover Would-Enforce Reason",
			shutdownLabel: "Guarded cutover would-enforce reason",
			value:         accessors.GuardedCutoverWouldEnforceReason,
		},
		{
			runningLabel:  "Guarded Cutover Enforce Hint",
			shutdownLabel: "Guarded cutover enforce hint",
			value:         accessors.GuardedCutoverEnforceHintState,
		},
		{
			runningLabel:  "Guarded Cutover Enforce Hint Reason",
			shutdownLabel: "Guarded cutover enforce hint reason",
			value:         accessors.GuardedCutoverEnforceHintReason,
		},
		{
			runningLabel:  "Guarded Cutover Overview",
			shutdownLabel: "Guarded cutover overview",
			value:         accessors.GuardedCutoverOverviewState,
		},
		{
			runningLabel:  "Guarded Cutover Overview Reason",
			shutdownLabel: "Guarded cutover overview reason",
			value:         accessors.GuardedCutoverOverviewReason,
		},
	}
}

func buildOwnershipRolloutPresenterAccessors() ownershipRolloutPresenterAccessors {
	return ownershipRolloutPresenterAccessors{
		Ownership: ownershipRolloutOwnershipAccessors{
			ShadowOwnedEvents:      ownershipRolloutShadowOwnedEventsValue,
			LegacyOwnedEvents:      ownershipRolloutLegacyOwnedEventsValue,
			OwnershipChains:        ownershipRolloutOwnershipChainsValue,
			Mode:                   ownershipRolloutModeValue,
			ProgressionState:       ownershipRolloutProgressionStateValue,
			ProgressionReason:      ownershipRolloutProgressionReasonValue,
			CutoverDryRunAction:    ownershipRolloutCutoverDryRunActionValue,
			CutoverDryRunReason:    ownershipRolloutCutoverDryRunReasonValue,
			CutoverCandidate:       ownershipRolloutCutoverCandidateEligibleValue,
			CutoverCandidateReason: ownershipRolloutCutoverCandidateReasonValue,
		},
		Approval: ownershipRolloutApprovalAccessors{
			ManualApprovalCheckpointState:  ownershipRolloutManualApprovalCheckpointStateValue,
			ManualApprovalCheckpointReason: ownershipRolloutManualApprovalCheckpointReasonValue,
			OperatorHandoffState:           ownershipRolloutOperatorHandoffStateValue,
			OperatorHandoffReason:          ownershipRolloutOperatorHandoffReasonValue,
			ApprovalWorkItemStatus:         ownershipRolloutApprovalWorkItemStatusValue,
			ApprovalWorkItemOwner:          ownershipRolloutApprovalWorkItemOwnerValue,
			ApprovalWorkItemReviewFields:   ownershipRolloutApprovalWorkItemReviewFieldsValue,
			ApprovalWorkItemReason:         ownershipRolloutApprovalWorkItemReasonValue,
			ApprovalChecklistState:         ownershipRolloutApprovalChecklistStateValue,
			ApprovalChecklistReason:        ownershipRolloutApprovalChecklistReasonValue,
		},
		Guarded: ownershipRolloutGuardedAccessors{
			GuardedCutoverHookAction:         ownershipRolloutGuardedCutoverHookActionValue,
			GuardedCutoverHookReason:         ownershipRolloutGuardedCutoverHookReasonValue,
			GuardedCutoverHookPolicyMode:     ownershipRolloutGuardedCutoverHookPolicyModeValue,
			GuardedCutoverHookPolicyAction:   ownershipRolloutGuardedCutoverHookPolicyActionValue,
			GuardedCutoverHookPolicyReason:   ownershipRolloutGuardedCutoverHookPolicyReasonValue,
			GuardedCutoverWouldEnforceAction: ownershipRolloutGuardedCutoverWouldEnforceActionValue,
			GuardedCutoverWouldEnforceReason: ownershipRolloutGuardedCutoverWouldEnforceReasonValue,
			GuardedCutoverEnforceHintState:   ownershipRolloutGuardedCutoverEnforceHintStateValue,
			GuardedCutoverEnforceHintReason:  ownershipRolloutGuardedCutoverEnforceHintReasonValue,
			GuardedCutoverOverviewState:      ownershipRolloutGuardedCutoverOverviewStateValue,
			GuardedCutoverOverviewReason:     ownershipRolloutGuardedCutoverOverviewReasonValue,
		},
	}
}

func ownershipRolloutLogDescriptors() []ownershipRolloutLogDescriptor {
	sections := buildOwnershipRolloutLogSections()
	descriptors := []ownershipRolloutLogDescriptor{}
	descriptors = append(descriptors, sections.Ownership...)
	descriptors = append(descriptors, sections.Approval...)
	descriptors = append(descriptors, sections.Guarded...)
	return descriptors
}

func buildOwnershipRolloutLogSections() ownershipRolloutLogSections {
	return ownershipRolloutLogSections{
		Ownership: ownershipRolloutOwnershipLogDescriptors(),
		Approval:  ownershipRolloutApprovalLogDescriptors(),
		Guarded:   ownershipRolloutGuardedCutoverLogDescriptors(),
	}
}

func ownershipRolloutOwnershipLogDescriptors() []ownershipRolloutLogDescriptor {
	return []ownershipRolloutLogDescriptor{
		{
			message: "Ownership rollout cutover candidate evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"cutover_candidate", summary.CutoverCandidate.Eligible,
					"cutover_candidate_reason", summary.CutoverCandidate.Reason,
					"rollout_policy_mode", summary.Policy.Mode,
					"rollout_effective_state", summary.Progression.State,
					"cutover_dry_run_action", summary.CutoverDryRun.Action,
				}
			},
		},
	}
}

func ownershipRolloutApprovalLogDescriptors() []ownershipRolloutLogDescriptor {
	return []ownershipRolloutLogDescriptor{
		{
			message: "Ownership rollout manual approval checkpoint evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"manual_approval_checkpoint_state", summary.ManualApprovalCheckpoint.State,
					"manual_approval_checkpoint_reason", summary.ManualApprovalCheckpoint.Reason,
					"cutover_candidate", summary.CutoverCandidate.Eligible,
				}
			},
		},
		{
			message: "Ownership rollout operator handoff evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"operator_handoff_state", summary.OperatorHandoff.State,
					"operator_handoff_reason", summary.OperatorHandoff.Reason,
				}
			},
		},
		{
			message: "Ownership rollout approval work item evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"approval_work_item_status", summary.ApprovalWorkItem.Status,
					"approval_work_item_owner", summary.ApprovalWorkItem.Owner,
					"approval_work_item_review_fields", summary.ApprovalWorkItem.ReviewFields,
					"approval_work_item_reason", summary.ApprovalWorkItem.Reason,
				}
			},
		},
		{
			message: "Ownership rollout approval checklist evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"approval_checklist_state", summary.ApprovalChecklist.State,
					"approval_checklist_reason", summary.ApprovalChecklist.Reason,
				}
			},
		},
	}
}

func ownershipRolloutGuardedCutoverLogDescriptors() []ownershipRolloutLogDescriptor {
	return []ownershipRolloutLogDescriptor{
		{
			message: "Ownership rollout guarded cutover hook evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"guarded_cutover_hook_action", summary.GuardedCutoverHook.Action,
					"guarded_cutover_hook_reason", summary.GuardedCutoverHook.Reason,
					"guarded_cutover_hook_policy_mode", summary.GuardedCutoverHookPolicy.Mode,
					"guarded_cutover_hook_policy_action", summary.GuardedCutoverHookPolicy.Action,
				}
			},
		},
		{
			message: "Ownership rollout guarded cutover would-enforce evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"guarded_cutover_would_enforce_action", summary.GuardedCutoverWouldEnforce.Action,
					"guarded_cutover_would_enforce_reason", summary.GuardedCutoverWouldEnforce.Reason,
				}
			},
		},
		{
			message: "Ownership rollout guarded cutover enforce hint evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"guarded_cutover_enforce_hint_state", summary.GuardedCutoverEnforceHint.State,
					"guarded_cutover_enforce_hint_reason", summary.GuardedCutoverEnforceHint.Reason,
				}
			},
		},
		{
			message: "Ownership rollout guarded cutover overview evaluated",
			fields: func(phase string, summary ownershipRolloutSummarySnapshot) []interface{} {
				return []interface{}{
					"service", "monolithic",
					"phase", phase,
					"guarded_cutover_overview_state", summary.GuardedCutoverOverview.State,
					"guarded_cutover_overview_reason", summary.GuardedCutoverOverview.Reason,
				}
			},
		},
	}
}

func ownershipRolloutShadowOwnedEventsValue(summary ownershipRolloutSummarySnapshot) string {
	return fmt.Sprintf("%d", summary.Summary.ShadowOwnedEvents)
}

func ownershipRolloutLegacyOwnedEventsValue(summary ownershipRolloutSummarySnapshot) string {
	return fmt.Sprintf("%d", summary.Summary.LegacyOwnedEvents)
}

func ownershipRolloutOwnershipChainsValue(summary ownershipRolloutSummarySnapshot) string {
	return fmt.Sprintf("%d", summary.Summary.Chains)
}

func ownershipRolloutModeValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.Mode
}

func ownershipRolloutProgressionStateValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.Progression.State
}

func ownershipRolloutProgressionReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.Progression.Reason
}

func ownershipRolloutCutoverDryRunActionValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.CutoverDryRun.Action
}

func ownershipRolloutCutoverDryRunReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.CutoverDryRun.Reason
}

func ownershipRolloutCutoverCandidateEligibleValue(summary ownershipRolloutSummarySnapshot) string {
	return fmt.Sprintf("%t", summary.CutoverCandidate.Eligible)
}

func ownershipRolloutCutoverCandidateReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.CutoverCandidate.Reason
}

func ownershipRolloutManualApprovalCheckpointStateValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ManualApprovalCheckpoint.State
}

func ownershipRolloutManualApprovalCheckpointReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ManualApprovalCheckpoint.Reason
}

func ownershipRolloutOperatorHandoffStateValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.OperatorHandoff.State
}

func ownershipRolloutOperatorHandoffReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.OperatorHandoff.Reason
}

func ownershipRolloutApprovalWorkItemStatusValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ApprovalWorkItem.Status
}

func ownershipRolloutApprovalWorkItemOwnerValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ApprovalWorkItem.Owner
}

func ownershipRolloutApprovalWorkItemReviewFieldsValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ApprovalWorkItem.ReviewFields
}

func ownershipRolloutApprovalWorkItemReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ApprovalWorkItem.Reason
}

func ownershipRolloutApprovalChecklistStateValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ApprovalChecklist.State
}

func ownershipRolloutApprovalChecklistReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.ApprovalChecklist.Reason
}

func ownershipRolloutGuardedCutoverHookActionValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverHook.Action
}

func ownershipRolloutGuardedCutoverHookReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverHook.Reason
}

func ownershipRolloutGuardedCutoverHookPolicyModeValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverHookPolicy.Mode
}

func ownershipRolloutGuardedCutoverHookPolicyActionValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverHookPolicy.Action
}

func ownershipRolloutGuardedCutoverHookPolicyReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverHookPolicy.Reason
}

func ownershipRolloutGuardedCutoverWouldEnforceActionValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverWouldEnforce.Action
}

func ownershipRolloutGuardedCutoverWouldEnforceReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverWouldEnforce.Reason
}

func ownershipRolloutGuardedCutoverEnforceHintStateValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverEnforceHint.State
}

func ownershipRolloutGuardedCutoverEnforceHintReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverEnforceHint.Reason
}

func ownershipRolloutGuardedCutoverOverviewStateValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverOverview.State
}

func ownershipRolloutGuardedCutoverOverviewReasonValue(summary ownershipRolloutSummarySnapshot) string {
	return summary.GuardedCutoverOverview.Reason
}
