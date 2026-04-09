package main

import (
	"os"
	"strings"
)

type ownershipRolloutAdvisory struct {
	Decision string
	Status   string
	Ready    bool
	Reason   string
}

type ownershipRolloutPolicy struct {
	Mode         string
	Action       string
	Reason       string
	Acknowledged bool
	AckState     string
}

type ownershipEffectiveProgression struct {
	State  string
	Reason string
}

type ownershipCutoverDryRun struct {
	Action string
	Reason string
}

type ownershipCutoverCandidate struct {
	Eligible bool
	Reason   string
}

type ownershipManualApprovalCheckpoint struct {
	State  string
	Reason string
}

type ownershipOperatorHandoff struct {
	State  string
	Reason string
}

type ownershipApprovalWorkItem struct {
	Status       string
	Owner        string
	ReviewFields string
	Reason       string
}

type ownershipApprovalChecklist struct {
	State  string
	Reason string
}

type ownershipGuardedCutoverHook struct {
	Action string
	Reason string
}

type ownershipGuardedCutoverHookPolicy struct {
	Mode   string
	Action string
	Reason string
}

type ownershipGuardedCutoverWouldEnforce struct {
	Action string
	Reason string
}

type ownershipGuardedCutoverEnforceHint struct {
	State  string
	Reason string
}

type ownershipGuardedCutoverOverview struct {
	State  string
	Reason string
}

func ownershipPolicyModeCode(mode string) float64 {
	switch mode {
	case "report-only":
		return 1
	case "manual-gate":
		return 2
	default:
		return 9
	}
}

func ownershipAckStateCode(state string) float64 {
	switch state {
	case "pending":
		return 0
	case "acknowledged":
		return 1
	default:
		return 9
	}
}

func ownershipEffectiveProgressionCode(state string) float64 {
	switch state {
	case "observe":
		return 1
	case "review-required":
		return 2
	case "acknowledged":
		return 3
	case "ready-for-cutover":
		return 4
	case "unknown":
		return 9
	default:
		return 9
	}
}

func ownershipCutoverDryRunCode(action string) float64 {
	switch action {
	case "would-hold":
		return 1
	case "would-allow":
		return 2
	case "would-unknown":
		return 9
	default:
		return 9
	}
}

func ownershipCutoverCandidateCode(candidate ownershipCutoverCandidate) float64 {
	if candidate.Eligible {
		return 1
	}
	return 0
}

func ownershipManualApprovalCheckpointCode(checkpoint ownershipManualApprovalCheckpoint) float64 {
	switch checkpoint.State {
	case "inactive":
		return 0
	case "awaiting-approval":
		return 1
	case "unknown":
		return 9
	default:
		return 9
	}
}

func ownershipOperatorHandoffCode(handoff ownershipOperatorHandoff) float64 {
	switch handoff.State {
	case "none":
		return 0
	case "operator-review":
		return 1
	case "investigate":
		return 9
	default:
		return 9
	}
}

func ownershipApprovalWorkItemCode(item ownershipApprovalWorkItem) float64 {
	switch item.Status {
	case "none":
		return 0
	case "open":
		return 1
	case "investigate":
		return 9
	default:
		return 9
	}
}

func ownershipApprovalChecklistCode(checklist ownershipApprovalChecklist) float64 {
	switch checklist.State {
	case "incomplete":
		return 0
	case "ready":
		return 1
	case "investigate":
		return 9
	default:
		return 9
	}
}

func ownershipGuardedCutoverHookCode(hook ownershipGuardedCutoverHook) float64 {
	switch hook.Action {
	case "noop-hold":
		return 1
	case "noop-allow":
		return 2
	case "noop-investigate":
		return 9
	default:
		return 9
	}
}

func ownershipGuardedCutoverHookPolicyModeCode(mode string) float64 {
	switch mode {
	case "noop-only":
		return 1
	case "enforce-ready":
		return 2
	default:
		return 9
	}
}

func ownershipGuardedCutoverWouldEnforceCode(summary ownershipGuardedCutoverWouldEnforce) float64 {
	switch summary.Action {
	case "would-hold":
		return 1
	case "would-allow":
		return 2
	case "would-investigate":
		return 9
	default:
		return 9
	}
}

func ownershipGuardedCutoverEnforceHintCode(hint ownershipGuardedCutoverEnforceHint) float64 {
	switch hint.State {
	case "safe-to-observe":
		return 1
	case "hold-before-enforce":
		return 2
	case "investigate-before-enforce":
		return 9
	default:
		return 9
	}
}

func ownershipGuardedCutoverOverviewCode(overview ownershipGuardedCutoverOverview) float64 {
	switch overview.State {
	case "observe":
		return 1
	case "hold":
		return 2
	case "investigate":
		return 9
	default:
		return 9
	}
}

func classifyOwnershipCutoverDryRun(progression ownershipEffectiveProgression) ownershipCutoverDryRun {
	switch progression.State {
	case "ready-for-cutover":
		return ownershipCutoverDryRun{
			Action: "would-allow",
			Reason: "effective progression state indicates the service would allow cutover in dry-run mode",
		}
	case "unknown":
		return ownershipCutoverDryRun{
			Action: "would-unknown",
			Reason: "effective progression state is unknown so dry-run cutover decision is also unknown",
		}
	default:
		return ownershipCutoverDryRun{
			Action: "would-hold",
			Reason: "effective progression state does not yet satisfy cutover conditions in dry-run mode",
		}
	}
}

func classifyOwnershipCutoverCandidate(
	policy ownershipRolloutPolicy,
	progression ownershipEffectiveProgression,
	cutoverDryRun ownershipCutoverDryRun,
) ownershipCutoverCandidate {
	switch {
	case policy.Mode != "manual-gate":
		return ownershipCutoverCandidate{
			Eligible: false,
			Reason:   "cutover candidate requires manual-gate policy mode",
		}
	case !policy.Acknowledged:
		return ownershipCutoverCandidate{
			Eligible: false,
			Reason:   "cutover candidate requires recorded operator acknowledgment",
		}
	case progression.State != "ready-for-cutover":
		return ownershipCutoverCandidate{
			Eligible: false,
			Reason:   "cutover candidate requires ready-for-cutover effective progression state",
		}
	case cutoverDryRun.Action != "would-allow":
		return ownershipCutoverCandidate{
			Eligible: false,
			Reason:   "cutover candidate requires dry-run cutover decision would-allow",
		}
	default:
		return ownershipCutoverCandidate{
			Eligible: true,
			Reason:   "manual gate is acknowledged and dry-run cutover would allow progression",
		}
	}
}

func classifyOwnershipManualApprovalCheckpoint(
	progression ownershipEffectiveProgression,
	cutoverCandidate ownershipCutoverCandidate,
) ownershipManualApprovalCheckpoint {
	switch {
	case progression.State == "unknown":
		return ownershipManualApprovalCheckpoint{
			State:  "unknown",
			Reason: "manual approval checkpoint is unknown because rollout progression is unknown",
		}
	case cutoverCandidate.Eligible:
		return ownershipManualApprovalCheckpoint{
			State:  "awaiting-approval",
			Reason: "instance is a cutover candidate and is awaiting manual approval checkpoint",
		}
	default:
		return ownershipManualApprovalCheckpoint{
			State:  "inactive",
			Reason: "instance has not yet reached cutover candidate posture",
		}
	}
}

func classifyOwnershipOperatorHandoff(
	manualApprovalCheckpoint ownershipManualApprovalCheckpoint,
) ownershipOperatorHandoff {
	switch manualApprovalCheckpoint.State {
	case "awaiting-approval":
		return ownershipOperatorHandoff{
			State:  "operator-review",
			Reason: "manual approval checkpoint is active and requires operator review",
		}
	case "unknown":
		return ownershipOperatorHandoff{
			State:  "investigate",
			Reason: "manual approval checkpoint is unknown and rollout posture requires investigation",
		}
	default:
		return ownershipOperatorHandoff{
			State:  "none",
			Reason: "operator handoff is not required for the current rollout posture",
		}
	}
}

func classifyOwnershipApprovalWorkItem(
	operatorHandoff ownershipOperatorHandoff,
) ownershipApprovalWorkItem {
	const reviewFields = "rollout_effective_state,rollout_cutover_candidate,rollout_manual_approval_checkpoint_state,rollout_operator_handoff_state"

	switch operatorHandoff.State {
	case "operator-review":
		return ownershipApprovalWorkItem{
			Status:       "open",
			Owner:        "platform-team/manual-approver",
			ReviewFields: reviewFields,
			Reason:       "operator handoff requires approval review before any future cutover action",
		}
	case "investigate":
		return ownershipApprovalWorkItem{
			Status:       "investigate",
			Owner:        "platform-team/runtime-owners",
			ReviewFields: reviewFields,
			Reason:       "operator handoff requires investigation before approval can proceed",
		}
	default:
		return ownershipApprovalWorkItem{
			Status:       "none",
			Owner:        "none",
			ReviewFields: reviewFields,
			Reason:       "no approval work item is required for the current rollout posture",
		}
	}
}

func classifyOwnershipApprovalChecklist(
	cutoverCandidate ownershipCutoverCandidate,
	manualApprovalCheckpoint ownershipManualApprovalCheckpoint,
	operatorHandoff ownershipOperatorHandoff,
	approvalWorkItem ownershipApprovalWorkItem,
) ownershipApprovalChecklist {
	switch {
	case approvalWorkItem.Status == "investigate" || operatorHandoff.State == "investigate" || manualApprovalCheckpoint.State == "unknown":
		return ownershipApprovalChecklist{
			State:  "investigate",
			Reason: "approval checklist requires investigation before review can proceed",
		}
	case cutoverCandidate.Eligible &&
		manualApprovalCheckpoint.State == "awaiting-approval" &&
		operatorHandoff.State == "operator-review" &&
		approvalWorkItem.Status == "open" &&
		approvalWorkItem.ReviewFields != "":
		return ownershipApprovalChecklist{
			State:  "ready",
			Reason: "approval checklist prerequisites are present for manual review",
		}
	default:
		return ownershipApprovalChecklist{
			State:  "incomplete",
			Reason: "approval checklist prerequisites are not fully satisfied yet",
		}
	}
}

func classifyOwnershipGuardedCutoverHook(
	cutoverDryRun ownershipCutoverDryRun,
	cutoverCandidate ownershipCutoverCandidate,
	approvalChecklist ownershipApprovalChecklist,
) ownershipGuardedCutoverHook {
	switch {
	case approvalChecklist.State == "investigate" || cutoverDryRun.Action == "would-unknown":
		return ownershipGuardedCutoverHook{
			Action: "noop-investigate",
			Reason: "guarded cutover hook requires investigation before any future control action",
		}
	case approvalChecklist.State == "ready" && cutoverCandidate.Eligible && cutoverDryRun.Action == "would-allow":
		return ownershipGuardedCutoverHook{
			Action: "noop-allow",
			Reason: "guarded cutover hook would allow progression, but remains non-blocking in dry-run mode",
		}
	default:
		return ownershipGuardedCutoverHook{
			Action: "noop-hold",
			Reason: "guarded cutover hook would still hold because approval prerequisites are not fully satisfied",
		}
	}
}

func resolveOwnershipGuardedCutoverHookPolicyFromEnv(
	hook ownershipGuardedCutoverHook,
) ownershipGuardedCutoverHookPolicy {
	mode := normalizeOwnershipGuardedCutoverHookPolicyMode(os.Getenv("CHAINPULSE_OWNERSHIP_GUARDED_CUTOVER_HOOK_POLICY_MODE"))
	switch mode {
	case "enforce-ready":
		return ownershipGuardedCutoverHookPolicy{
			Mode:   mode,
			Action: enforceReadyGuardedHookPolicyAction(hook.Action),
			Reason: "enforce-ready policy is configured for the guarded cutover hook, but remains advisory until execution gating is introduced",
		}
	default:
		return ownershipGuardedCutoverHookPolicy{
			Mode:   "noop-only",
			Action: "noop-report",
			Reason: "noop-only policy reports guarded cutover hook outcomes without affecting execution",
		}
	}
}

func normalizeOwnershipGuardedCutoverHookPolicyMode(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "enforce-ready", "enforce_ready", "enforce":
		return "enforce-ready"
	case "", "noop-only", "noop_only", "noop":
		return "noop-only"
	default:
		return "noop-only"
	}
}

func enforceReadyGuardedHookPolicyAction(hookAction string) string {
	switch hookAction {
	case "noop-allow":
		return "enforce-would-allow"
	case "noop-investigate":
		return "enforce-would-investigate"
	default:
		return "enforce-would-hold"
	}
}

func classifyOwnershipGuardedCutoverWouldEnforce(
	hook ownershipGuardedCutoverHook,
	policy ownershipGuardedCutoverHookPolicy,
) ownershipGuardedCutoverWouldEnforce {
	switch {
	case hook.Action == "noop-investigate" || policy.Action == "enforce-would-investigate":
		return ownershipGuardedCutoverWouldEnforce{
			Action: "would-investigate",
			Reason: "future enforcement posture requires investigation before any guarded cutover decision",
		}
	case policy.Mode == "enforce-ready" && policy.Action == "enforce-would-allow" && hook.Action == "noop-allow":
		return ownershipGuardedCutoverWouldEnforce{
			Action: "would-allow",
			Reason: "future enforcement posture would allow this instance to proceed",
		}
	default:
		return ownershipGuardedCutoverWouldEnforce{
			Action: "would-hold",
			Reason: "future enforcement posture would still hold this instance",
		}
	}
}

func classifyOwnershipGuardedCutoverEnforceHint(
	wouldEnforce ownershipGuardedCutoverWouldEnforce,
) ownershipGuardedCutoverEnforceHint {
	switch wouldEnforce.Action {
	case "would-allow":
		return ownershipGuardedCutoverEnforceHint{
			State:  "safe-to-observe",
			Reason: "future enforcement posture is healthy enough to keep observing toward eventual enforcement",
		}
	case "would-investigate":
		return ownershipGuardedCutoverEnforceHint{
			State:  "investigate-before-enforce",
			Reason: "future enforcement posture needs investigation before any enforce decision is considered",
		}
	default:
		return ownershipGuardedCutoverEnforceHint{
			State:  "hold-before-enforce",
			Reason: "future enforcement posture should remain on hold before any enforce decision is considered",
		}
	}
}

func classifyOwnershipGuardedCutoverOverview(
	wouldEnforce ownershipGuardedCutoverWouldEnforce,
	enforceHint ownershipGuardedCutoverEnforceHint,
) ownershipGuardedCutoverOverview {
	switch {
	case wouldEnforce.Action == "would-investigate" || enforceHint.State == "investigate-before-enforce":
		return ownershipGuardedCutoverOverview{
			State:  "investigate",
			Reason: "guarded cutover posture requires investigation before any future enforcement decision",
		}
	case wouldEnforce.Action == "would-allow" && enforceHint.State == "safe-to-observe":
		return ownershipGuardedCutoverOverview{
			State:  "observe",
			Reason: "guarded cutover posture is healthy enough to keep observing toward possible enforcement",
		}
	default:
		return ownershipGuardedCutoverOverview{
			State:  "hold",
			Reason: "guarded cutover posture should remain on hold before any future enforcement decision",
		}
	}
}

func classifyOwnershipEffectiveProgression(
	advisory ownershipRolloutAdvisory,
	policy ownershipRolloutPolicy,
) ownershipEffectiveProgression {
	switch {
	case advisory.Decision == "unknown":
		return ownershipEffectiveProgression{
			State:  "unknown",
			Reason: "ownership rollout decision is unknown",
		}
	case policy.Mode == "manual-gate" && policy.Acknowledged && advisory.Decision == "allow":
		return ownershipEffectiveProgression{
			State:  "ready-for-cutover",
			Reason: "manual gate is acknowledged and advisory allows progression",
		}
	case policy.Mode == "manual-gate" && policy.Acknowledged:
		return ownershipEffectiveProgression{
			State:  "acknowledged",
			Reason: "manual gate is acknowledged but advisory still holds progression",
		}
	case policy.Mode == "manual-gate":
		return ownershipEffectiveProgression{
			State:  "review-required",
			Reason: "manual gate requires operator review before progression",
		}
	case policy.Mode == "report-only":
		return ownershipEffectiveProgression{
			State:  "observe",
			Reason: "report-only mode observes rollout state without changing execution",
		}
	default:
		return ownershipEffectiveProgression{
			State:  "unknown",
			Reason: "ownership effective progression state is unknown",
		}
	}
}

func classifyOwnershipRolloutAdvisory(summary ownershipSummary) ownershipRolloutAdvisory {
	switch mode := classifyOwnershipMode(summary); mode {
	case "runtime-owned":
		return ownershipRolloutAdvisory{
			Decision: "allow",
			Status:   "ready",
			Ready:    true,
			Reason:   "shared runtime owns observed writes",
		}
	case "shadow":
		return ownershipRolloutAdvisory{
			Decision: "hold",
			Status:   "shadow-observe",
			Ready:    false,
			Reason:   "shared runtime still coexists with legacy writes",
		}
	case "legacy-only":
		return ownershipRolloutAdvisory{
			Decision: "hold",
			Status:   "legacy-only",
			Ready:    false,
			Reason:   "shared runtime has not yet claimed writes",
		}
	case "idle":
		return ownershipRolloutAdvisory{
			Decision: "hold",
			Status:   "idle",
			Ready:    false,
			Reason:   "no ownership activity observed yet",
		}
	default:
		return ownershipRolloutAdvisory{
			Decision: "unknown",
			Status:   "unknown",
			Ready:    false,
			Reason:   "ownership rollout state is unknown",
		}
	}
}

func resolveOwnershipRolloutPolicyFromEnv(advisory ownershipRolloutAdvisory) ownershipRolloutPolicy {
	mode := normalizeOwnershipRolloutPolicyMode(os.Getenv("CHAINPULSE_OWNERSHIP_ROLLOUT_POLICY_MODE"))
	acknowledged := resolveOwnershipRolloutAcknowledgedFromEnv()

	switch mode {
	case "report-only":
		return ownershipRolloutPolicy{
			Mode:         mode,
			Action:       reportOnlyPolicyAction(advisory.Decision),
			Reason:       "report-only mode does not block ownership rollout",
			Acknowledged: acknowledged,
			AckState:     acknowledgmentState(acknowledged),
		}
	case "manual-gate":
		return ownershipRolloutPolicy{
			Mode:         mode,
			Action:       manualGatePolicyAction(advisory.Decision, acknowledged),
			Reason:       manualGatePolicyReason(acknowledged),
			Acknowledged: acknowledged,
			AckState:     acknowledgmentState(acknowledged),
		}
	default:
		return ownershipRolloutPolicy{
			Mode:         "report-only",
			Action:       reportOnlyPolicyAction(advisory.Decision),
			Reason:       "report-only mode does not block ownership rollout",
			Acknowledged: acknowledged,
			AckState:     acknowledgmentState(acknowledged),
		}
	}
}

func normalizeOwnershipRolloutPolicyMode(raw string) string {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "manual-gate", "manual_gate", "manual":
		return "manual-gate"
	case "", "report-only", "report_only", "report":
		return "report-only"
	default:
		return "report-only"
	}
}

func reportOnlyPolicyAction(decision string) string {
	switch decision {
	case "allow":
		return "report-allow"
	case "hold":
		return "report-hold"
	default:
		return "report-unknown"
	}
}

func manualGatePolicyAction(decision string, acknowledged bool) string {
	prefix := "manual-review"
	if acknowledged {
		prefix = "manual-acknowledged"
	}
	switch decision {
	case "allow":
		return prefix + "-allow"
	case "hold":
		return prefix + "-hold"
	default:
		return prefix + "-unknown"
	}
}

func manualGatePolicyReason(acknowledged bool) string {
	if acknowledged {
		return "manual-gate mode has recorded operator acknowledgment for ownership progression"
	}
	return "manual-gate mode requires operator review before ownership progression"
}

func resolveOwnershipRolloutAcknowledgedFromEnv() bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv("CHAINPULSE_OWNERSHIP_ROLLOUT_ACKNOWLEDGED"))) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func acknowledgmentState(acknowledged bool) string {
	if acknowledged {
		return "acknowledged"
	}
	return "pending"
}
