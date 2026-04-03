package main

import (
	"testing"

	"chainpulse/pkg/plugins/api"
)

func TestBuildOwnershipRolloutReportBody(t *testing.T) {
	details := api.NewRolloutReportDetailsFromMetadata(api.RolloutReportMetadata{
		ReportID:       "monolithic-ownership-rollout-runtime",
		SchemaFamily:   "ownership-rollout-report",
		ReportVersion:  "v1",
		Service:        "monolithic",
		ReportScope:    "ownership-rollout",
		ReportSource:   "monolithic",
		ReportMode:     "runtime",
		DeploymentMode: "monolithic",
		GeneratedAt:    1700000000,
	})

	snapshot := ownershipRolloutSummarySnapshot{
		Summary: ownershipSummary{
			ShadowOwnedEvents: 3,
			LegacyOwnedEvents: 5,
			Chains:            2,
		},
		Mode: "shadow",
		Advisory: ownershipRolloutAdvisory{
			Decision: "hold",
			Status:   "shadow",
			Ready:    false,
			Reason:   "legacy path still active",
		},
		Policy: ownershipRolloutPolicy{
			Mode:         "manual-gate",
			Action:       "manual-review-hold",
			Reason:       "manual review required",
			Acknowledged: true,
			AckState:     "acknowledged",
		},
		Progression: ownershipEffectiveProgression{
			State:  "acknowledged",
			Reason: "manual gate acknowledged",
		},
		CutoverDryRun: ownershipCutoverDryRun{
			Action: "would-hold",
			Reason: "not ready for cutover",
		},
		CutoverCandidate: ownershipCutoverCandidate{
			Eligible: false,
			Reason:   "candidate not satisfied",
		},
		ManualApprovalCheckpoint: ownershipManualApprovalCheckpoint{
			State:  "inactive",
			Reason: "not a cutover candidate",
		},
		OperatorHandoff: ownershipOperatorHandoff{
			State:  "none",
			Reason: "no operator action required",
		},
		ApprovalWorkItem: ownershipApprovalWorkItem{
			Status:       "open",
			Owner:        "platform-team/manual-approver",
			ReviewFields: "field-a,field-b",
			Reason:       "review required",
		},
		ApprovalChecklist: ownershipApprovalChecklist{
			State:  "incomplete",
			Reason: "candidate missing",
		},
		GuardedCutoverHook: ownershipGuardedCutoverHook{
			Action: "noop-hold",
			Reason: "guard not satisfied",
		},
		GuardedCutoverHookPolicy: ownershipGuardedCutoverHookPolicy{
			Mode:   "noop-only",
			Action: "noop-hold",
			Reason: "policy is observational",
		},
		GuardedCutoverWouldEnforce: ownershipGuardedCutoverWouldEnforce{
			Action: "would-hold",
			Reason: "enforce would still hold",
		},
		GuardedCutoverEnforceHint: ownershipGuardedCutoverEnforceHint{
			State:  "hold-before-enforce",
			Reason: "wait before enforce",
		},
		GuardedCutoverOverview: ownershipGuardedCutoverOverview{
			State:  "hold",
			Reason: "overview indicates hold",
		},
	}

	buildOwnershipRolloutReportBody(details, snapshot)

	if got := details.Mode; got != "shadow" {
		t.Fatalf("expected mode shadow, got %q", got)
	}
	if got := details.Summary.ShadowOwnedEvents; got != 3 {
		t.Fatalf("expected shadow owned events 3, got %d", got)
	}
	if got := details.Policy.Mode; got != "manual-gate" {
		t.Fatalf("expected policy mode manual-gate, got %q", got)
	}
	if got := details.Approval.WorkItem.ReviewFields; got != "field-a,field-b" {
		t.Fatalf("expected review fields field-a,field-b, got %q", got)
	}
	if got := details.GuardedCutover.Overview.State; got != "hold" {
		t.Fatalf("expected guarded overview hold, got %q", got)
	}
}

func TestBuildOwnershipRolloutReportBodySections(t *testing.T) {
	snapshot := ownershipRolloutSummarySnapshot{
		Summary: ownershipSummary{
			ShadowOwnedEvents: 7,
			LegacyOwnedEvents: 11,
			Chains:            3,
		},
		Mode: "shadow",
		Advisory: ownershipRolloutAdvisory{
			Decision: "hold",
			Status:   "shadow",
			Ready:    false,
			Reason:   "shadow mode active",
		},
		Policy: ownershipRolloutPolicy{
			Mode:         "report-only",
			Action:       "report-hold",
			Reason:       "report only",
			Acknowledged: false,
			AckState:     "pending",
		},
		Progression: ownershipEffectiveProgression{
			State:  "observe",
			Reason: "still observing",
		},
		CutoverDryRun: ownershipCutoverDryRun{
			Action: "would-hold",
			Reason: "not ready",
		},
		CutoverCandidate: ownershipCutoverCandidate{
			Eligible: false,
			Reason:   "not eligible",
		},
		ManualApprovalCheckpoint: ownershipManualApprovalCheckpoint{
			State:  "inactive",
			Reason: "no checkpoint",
		},
		OperatorHandoff: ownershipOperatorHandoff{
			State:  "none",
			Reason: "no handoff",
		},
		ApprovalWorkItem: ownershipApprovalWorkItem{
			Status:       "none",
			Owner:        "none",
			ReviewFields: "field-x",
			Reason:       "no work item",
		},
		ApprovalChecklist: ownershipApprovalChecklist{
			State:  "incomplete",
			Reason: "not ready",
		},
		GuardedCutoverHook: ownershipGuardedCutoverHook{
			Action: "noop-hold",
			Reason: "noop hold",
		},
		GuardedCutoverHookPolicy: ownershipGuardedCutoverHookPolicy{
			Mode:   "noop-only",
			Action: "noop-hold",
			Reason: "noop policy",
		},
		GuardedCutoverWouldEnforce: ownershipGuardedCutoverWouldEnforce{
			Action: "would-hold",
			Reason: "would hold",
		},
		GuardedCutoverEnforceHint: ownershipGuardedCutoverEnforceHint{
			State:  "hold-before-enforce",
			Reason: "hold first",
		},
		GuardedCutoverOverview: ownershipGuardedCutoverOverview{
			State:  "hold",
			Reason: "hold overview",
		},
	}

	t.Run("surface", func(t *testing.T) {
		section := buildOwnershipRolloutReportSurfaceSection(snapshot)
		if got := section.Summary.ShadowOwnedEvents; got != 7 {
			t.Fatalf("expected surface summary 7, got %d", got)
		}
		if got := section.Policy.Mode; got != "report-only" {
			t.Fatalf("expected surface policy report-only, got %q", got)
		}
	})

	t.Run("approval", func(t *testing.T) {
		section := buildOwnershipRolloutReportApprovalSection(snapshot)
		if got := section.WorkItem.ReviewFields; got != "field-x" {
			t.Fatalf("expected approval review fields field-x, got %q", got)
		}
		if got := section.Checklist.State; got != "incomplete" {
			t.Fatalf("expected approval checklist incomplete, got %q", got)
		}
	})

	t.Run("guarded", func(t *testing.T) {
		section := buildOwnershipRolloutReportGuardedSection(snapshot)
		if got := section.Hook.Action; got != "noop-hold" {
			t.Fatalf("expected guarded hook noop-hold, got %q", got)
		}
		if got := section.EnforceHint.State; got != "hold-before-enforce" {
			t.Fatalf("expected guarded hint hold-before-enforce, got %q", got)
		}
	})

	t.Run("assembler", func(t *testing.T) {
		sections := buildOwnershipRolloutReportSections(snapshot)
		details := &api.RolloutReportDetails{}
		applyOwnershipRolloutReportSections(details, sections)

		if got := details.Summary.OwnershipChains; got != 3 {
			t.Fatalf("expected ownership chains 3, got %d", got)
		}
		if got := details.Approval.WorkItem.Owner; got != "none" {
			t.Fatalf("expected approval owner none, got %q", got)
		}
		if got := details.GuardedCutover.Overview.State; got != "hold" {
			t.Fatalf("expected guarded overview hold, got %q", got)
		}
	})
}
