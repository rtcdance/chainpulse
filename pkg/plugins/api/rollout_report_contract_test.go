package api

import (
	"context"
	"testing"
)

func TestNewRolloutReportDetailsFromMetadata(t *testing.T) {
	details := NewRolloutReportDetailsFromMetadata(RolloutReportMetadata{
		ReportID:       "monolithic-ownership-rollout-runtime",
		SchemaFamily:   OwnershipRolloutSchemaFamily,
		ReportVersion:  OwnershipRolloutReportVersion,
		Service:        "monolithic",
		ReportScope:    OwnershipRolloutReportScope,
		ReportSource:   "monolithic",
		ReportMode:     OwnershipRolloutReportMode,
		DeploymentMode: "monolithic",
		GeneratedAt:    1700000000,
	})

	if details == nil {
		t.Fatal("expected details to be created")
	}
	if got := details.ReportID; got != "monolithic-ownership-rollout-runtime" {
		t.Fatalf("expected report id, got %q", got)
	}
	if got := details.SchemaFamily; got != "ownership-rollout-report" {
		t.Fatalf("expected schema family, got %q", got)
	}
	if got := details.ReportVersion; got != "v1" {
		t.Fatalf("expected report version, got %q", got)
	}
	if got := details.Service; got != "monolithic" {
		t.Fatalf("expected service, got %q", got)
	}
	if got := details.ReportScope; got != "ownership-rollout" {
		t.Fatalf("expected report scope, got %q", got)
	}
	if got := details.ReportSource; got != "monolithic" {
		t.Fatalf("expected report source, got %q", got)
	}
	if got := details.ReportMode; got != "runtime" {
		t.Fatalf("expected report mode, got %q", got)
	}
	if got := details.DeploymentMode; got != "monolithic" {
		t.Fatalf("expected deployment mode, got %q", got)
	}
	if got := details.GeneratedAt; got != 1700000000 {
		t.Fatalf("expected generated_at, got %d", got)
	}
}

func TestNewOwnershipRolloutReportMetadata(t *testing.T) {
	metadata := NewOwnershipRolloutReportMetadata(
		"api-service",
		"api-service-ownership-rollout-runtime",
		"microservice",
		"microservice",
		1700000001,
	)

	if got := metadata.SchemaFamily; got != OwnershipRolloutSchemaFamily {
		t.Fatalf("expected schema family %q, got %q", OwnershipRolloutSchemaFamily, got)
	}
	if got := metadata.ReportVersion; got != OwnershipRolloutReportVersion {
		t.Fatalf("expected report version %q, got %q", OwnershipRolloutReportVersion, got)
	}
	if got := metadata.ReportScope; got != OwnershipRolloutReportScope {
		t.Fatalf("expected report scope %q, got %q", OwnershipRolloutReportScope, got)
	}
	if got := metadata.ReportMode; got != OwnershipRolloutReportMode {
		t.Fatalf("expected report mode %q, got %q", OwnershipRolloutReportMode, got)
	}
	if got := metadata.Service; got != "api-service" {
		t.Fatalf("expected service api-service, got %q", got)
	}
}

func TestRolloutReportProducerFunc(t *testing.T) {
	producer := RolloutReportProducerFunc(func(ctx context.Context) *RolloutReportDetails {
		return &RolloutReportDetails{ReportID: "producer-id"}
	})

	report := producer.BuildRolloutReport(context.Background())
	if report == nil {
		t.Fatal("expected rollout report")
	}
	if got := report.ReportID; got != "producer-id" {
		t.Fatalf("expected producer-id, got %q", got)
	}
}

func TestApplyRolloutReportSurfaceSection(t *testing.T) {
	details := &RolloutReportDetails{}
	section := RolloutReportSurfaceSection{
		Summary: RolloutReportSummary{
			ShadowOwnedEvents: 3,
			LegacyOwnedEvents: 4,
			OwnershipChains:   2,
		},
		Mode: "shadow",
		Advisory: RolloutReportAdvisory{
			Decision: "hold",
			Status:   "shadow-observe",
			Ready:    false,
			Reason:   "surface reason",
		},
		Policy: RolloutReportPolicy{
			Mode:         "report-only",
			Action:       "report-hold",
			Reason:       "policy reason",
			Acknowledged: false,
			AckState:     "pending",
		},
		Progression: RolloutReportStateReason{
			State:  "observe",
			Reason: "progression reason",
		},
		CutoverDryRun: RolloutReportAction{
			Action: "would-hold",
			Reason: "dry-run reason",
		},
		CutoverCandidate: RolloutReportCandidate{
			Eligible: false,
			Reason:   "candidate reason",
		},
	}

	ApplyRolloutReportSurfaceSection(details, section)

	if got := details.Summary.ShadowOwnedEvents; got != 3 {
		t.Fatalf("expected shadow owned 3, got %d", got)
	}
	if got := details.Mode; got != "shadow" {
		t.Fatalf("expected mode shadow, got %q", got)
	}
	if got := details.Advisory.Status; got != "shadow-observe" {
		t.Fatalf("expected advisory status shadow-observe, got %q", got)
	}
	if got := details.Policy.Action; got != "report-hold" {
		t.Fatalf("expected policy action report-hold, got %q", got)
	}
	if got := details.Progression.State; got != "observe" {
		t.Fatalf("expected progression observe, got %q", got)
	}
	if got := details.CutoverDryRun.Action; got != "would-hold" {
		t.Fatalf("expected cutover dry-run would-hold, got %q", got)
	}
	if got := details.CutoverCandidate.Reason; got != "candidate reason" {
		t.Fatalf("expected candidate reason, got %q", got)
	}
}

func TestBuildRolloutReportSurfaceSection(t *testing.T) {
	section := BuildRolloutReportSurfaceSection(RolloutReportSurfaceInput{
		Summary: RolloutReportSummary{
			ShadowOwnedEvents: 8,
			LegacyOwnedEvents: 2,
			OwnershipChains:   4,
		},
		Mode: "shadow",
		Advisory: RolloutReportAdvisory{
			Decision: "hold",
			Status:   "shadow-observe",
			Ready:    false,
			Reason:   "advisory reason",
		},
		Policy: RolloutReportPolicy{
			Mode:         "report-only",
			Action:       "report-hold",
			Reason:       "policy reason",
			Acknowledged: false,
			AckState:     "pending",
		},
		Progression: RolloutReportStateReason{
			State:  "observe",
			Reason: "progression reason",
		},
		CutoverDryRun: RolloutReportAction{
			Action: "would-hold",
			Reason: "dry-run reason",
		},
		CutoverCandidate: RolloutReportCandidate{
			Eligible: false,
			Reason:   "candidate reason",
		},
	})

	if got := section.Summary.ShadowOwnedEvents; got != 8 {
		t.Fatalf("expected shadow owned 8, got %d", got)
	}
	if got := section.Mode; got != "shadow" {
		t.Fatalf("expected mode shadow, got %q", got)
	}
	if got := section.Advisory.Status; got != "shadow-observe" {
		t.Fatalf("expected advisory status shadow-observe, got %q", got)
	}
	if got := section.Policy.Mode; got != "report-only" {
		t.Fatalf("expected policy mode report-only, got %q", got)
	}
	if got := section.Progression.State; got != "observe" {
		t.Fatalf("expected progression observe, got %q", got)
	}
	if got := section.CutoverDryRun.Action; got != "would-hold" {
		t.Fatalf("expected cutover dry-run would-hold, got %q", got)
	}
	if got := section.CutoverCandidate.Reason; got != "candidate reason" {
		t.Fatalf("expected candidate reason, got %q", got)
	}
}

func TestBuildRolloutReportSurfaceInput(t *testing.T) {
	input := BuildRolloutReportSurfaceInput(
		RolloutReportSurfaceCoreInput{
			Summary: RolloutReportSummary{
				ShadowOwnedEvents: 8,
				LegacyOwnedEvents: 2,
				OwnershipChains:   4,
			},
			Mode: "shadow",
			Advisory: RolloutReportAdvisory{
				Decision: "hold",
				Status:   "shadow-observe",
				Ready:    false,
				Reason:   "advisory reason",
			},
			Policy: RolloutReportPolicy{
				Mode:         "report-only",
				Action:       "report-hold",
				Reason:       "policy reason",
				Acknowledged: false,
				AckState:     "pending",
			},
			Progression: RolloutReportStateReason{
				State:  "observe",
				Reason: "progression reason",
			},
		},
		RolloutReportSurfaceCutoverInput{
			CutoverDryRun: RolloutReportAction{
				Action: "would-hold",
				Reason: "dry-run reason",
			},
			CutoverCandidate: RolloutReportCandidate{
				Eligible: false,
				Reason:   "candidate reason",
			},
		},
	)

	if got := input.Summary.ShadowOwnedEvents; got != 8 {
		t.Fatalf("expected shadow owned 8, got %d", got)
	}
	if got := input.Mode; got != "shadow" {
		t.Fatalf("expected mode shadow, got %q", got)
	}
	if got := input.Advisory.Status; got != "shadow-observe" {
		t.Fatalf("expected advisory status shadow-observe, got %q", got)
	}
	if got := input.Progression.State; got != "observe" {
		t.Fatalf("expected progression observe, got %q", got)
	}
	if got := input.CutoverDryRun.Action; got != "would-hold" {
		t.Fatalf("expected cutover dry-run would-hold, got %q", got)
	}
	if got := input.CutoverCandidate.Reason; got != "candidate reason" {
		t.Fatalf("expected candidate reason, got %q", got)
	}
}

func TestApplyRolloutReportApprovalSection(t *testing.T) {
	details := &RolloutReportDetails{}
	section := RolloutReportApproval{
		ManualApprovalCheckpoint: RolloutReportStateReason{
			State:  "inactive",
			Reason: "checkpoint reason",
		},
		OperatorHandoff: RolloutReportStateReason{
			State:  "none",
			Reason: "handoff reason",
		},
		WorkItem: RolloutReportApprovalItem{
			Status:       "none",
			Owner:        "none",
			ReviewFields: "field-a,field-b",
			Reason:       "work item reason",
		},
		Checklist: RolloutReportStateReason{
			State:  "incomplete",
			Reason: "checklist reason",
		},
	}

	ApplyRolloutReportApprovalSection(details, section)

	if got := details.Approval.ManualApprovalCheckpoint.State; got != "inactive" {
		t.Fatalf("expected checkpoint inactive, got %q", got)
	}
	if got := details.Approval.OperatorHandoff.State; got != "none" {
		t.Fatalf("expected operator handoff none, got %q", got)
	}
	if got := details.Approval.WorkItem.ReviewFields; got != "field-a,field-b" {
		t.Fatalf("expected review fields field-a,field-b, got %q", got)
	}
	if got := details.Approval.Checklist.Reason; got != "checklist reason" {
		t.Fatalf("expected checklist reason, got %q", got)
	}
}

func TestBuildRolloutReportApprovalSection(t *testing.T) {
	section := BuildRolloutReportApprovalSection(RolloutReportApprovalInput{
		ManualApprovalCheckpoint: RolloutReportStateReason{
			State:  "inactive",
			Reason: "checkpoint reason",
		},
		OperatorHandoff: RolloutReportStateReason{
			State:  "none",
			Reason: "handoff reason",
		},
		WorkItem: RolloutReportApprovalItem{
			Status:       "none",
			Owner:        "none",
			ReviewFields: "field-a,field-b",
			Reason:       "work item reason",
		},
		Checklist: RolloutReportStateReason{
			State:  "incomplete",
			Reason: "checklist reason",
		},
	})

	if got := section.ManualApprovalCheckpoint.State; got != "inactive" {
		t.Fatalf("expected checkpoint inactive, got %q", got)
	}
	if got := section.OperatorHandoff.State; got != "none" {
		t.Fatalf("expected operator handoff none, got %q", got)
	}
	if got := section.WorkItem.ReviewFields; got != "field-a,field-b" {
		t.Fatalf("expected review fields field-a,field-b, got %q", got)
	}
	if got := section.Checklist.Reason; got != "checklist reason" {
		t.Fatalf("expected checklist reason, got %q", got)
	}
}

func TestBuildRolloutReportApprovalInput(t *testing.T) {
	input := BuildRolloutReportApprovalInput(
		RolloutReportApprovalFlowInput{
			ManualApprovalCheckpoint: RolloutReportStateReason{
				State:  "inactive",
				Reason: "checkpoint reason",
			},
			OperatorHandoff: RolloutReportStateReason{
				State:  "none",
				Reason: "handoff reason",
			},
			Checklist: RolloutReportStateReason{
				State:  "incomplete",
				Reason: "checklist reason",
			},
		},
		RolloutReportApprovalWorkItemInput{
			WorkItem: RolloutReportApprovalItem{
				Status:       "none",
				Owner:        "none",
				ReviewFields: "field-a,field-b",
				Reason:       "work item reason",
			},
		},
	)

	if got := input.ManualApprovalCheckpoint.State; got != "inactive" {
		t.Fatalf("expected checkpoint inactive, got %q", got)
	}
	if got := input.OperatorHandoff.State; got != "none" {
		t.Fatalf("expected operator handoff none, got %q", got)
	}
	if got := input.WorkItem.ReviewFields; got != "field-a,field-b" {
		t.Fatalf("expected review fields field-a,field-b, got %q", got)
	}
	if got := input.Checklist.Reason; got != "checklist reason" {
		t.Fatalf("expected checklist reason, got %q", got)
	}
}

func TestApplyRolloutReportGuardedSection(t *testing.T) {
	details := &RolloutReportDetails{}
	section := RolloutReportGuarded{
		Hook: RolloutReportAction{
			Action: "noop-hold",
			Reason: "hook reason",
		},
		HookPolicy: RolloutReportModeAction{
			Mode:   "noop-only",
			Action: "noop-hold",
			Reason: "policy reason",
		},
		WouldEnforce: RolloutReportAction{
			Action: "would-hold",
			Reason: "would-enforce reason",
		},
		EnforceHint: RolloutReportStateReason{
			State:  "hold-before-enforce",
			Reason: "enforce hint reason",
		},
		Overview: RolloutReportStateReason{
			State:  "hold",
			Reason: "overview reason",
		},
	}

	ApplyRolloutReportGuardedSection(details, section)

	if got := details.GuardedCutover.Hook.Action; got != "noop-hold" {
		t.Fatalf("expected hook noop-hold, got %q", got)
	}
	if got := details.GuardedCutover.HookPolicy.Mode; got != "noop-only" {
		t.Fatalf("expected hook policy noop-only, got %q", got)
	}
	if got := details.GuardedCutover.WouldEnforce.Action; got != "would-hold" {
		t.Fatalf("expected would-enforce would-hold, got %q", got)
	}
	if got := details.GuardedCutover.EnforceHint.State; got != "hold-before-enforce" {
		t.Fatalf("expected enforce hint hold-before-enforce, got %q", got)
	}
	if got := details.GuardedCutover.Overview.Reason; got != "overview reason" {
		t.Fatalf("expected overview reason, got %q", got)
	}
}

func TestBuildRolloutReportGuardedSection(t *testing.T) {
	section := BuildRolloutReportGuardedSection(RolloutReportGuardedInput{
		Hook: RolloutReportAction{
			Action: "noop-hold",
			Reason: "hook reason",
		},
		HookPolicy: RolloutReportModeAction{
			Mode:   "noop-only",
			Action: "noop-hold",
			Reason: "policy reason",
		},
		WouldEnforce: RolloutReportAction{
			Action: "would-hold",
			Reason: "would-enforce reason",
		},
		EnforceHint: RolloutReportStateReason{
			State:  "hold-before-enforce",
			Reason: "enforce hint reason",
		},
		Overview: RolloutReportStateReason{
			State:  "hold",
			Reason: "overview reason",
		},
	})

	if got := section.Hook.Action; got != "noop-hold" {
		t.Fatalf("expected hook noop-hold, got %q", got)
	}
	if got := section.HookPolicy.Mode; got != "noop-only" {
		t.Fatalf("expected hook policy noop-only, got %q", got)
	}
	if got := section.WouldEnforce.Action; got != "would-hold" {
		t.Fatalf("expected would-enforce would-hold, got %q", got)
	}
	if got := section.EnforceHint.State; got != "hold-before-enforce" {
		t.Fatalf("expected enforce hint hold-before-enforce, got %q", got)
	}
	if got := section.Overview.Reason; got != "overview reason" {
		t.Fatalf("expected overview reason, got %q", got)
	}
}

func TestBuildRolloutReportGuardedInput(t *testing.T) {
	input := BuildRolloutReportGuardedInput(
		RolloutReportGuardedHookInput{
			Hook: RolloutReportAction{
				Action: "noop-hold",
				Reason: "hook reason",
			},
			HookPolicy: RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-hold",
				Reason: "policy reason",
			},
		},
		RolloutReportGuardedEnforcementInput{
			WouldEnforce: RolloutReportAction{
				Action: "would-hold",
				Reason: "would-enforce reason",
			},
			EnforceHint: RolloutReportStateReason{
				State:  "hold-before-enforce",
				Reason: "enforce hint reason",
			},
			Overview: RolloutReportStateReason{
				State:  "hold",
				Reason: "overview reason",
			},
		},
	)

	if got := input.Hook.Action; got != "noop-hold" {
		t.Fatalf("expected hook noop-hold, got %q", got)
	}
	if got := input.HookPolicy.Mode; got != "noop-only" {
		t.Fatalf("expected hook policy noop-only, got %q", got)
	}
	if got := input.WouldEnforce.Action; got != "would-hold" {
		t.Fatalf("expected would-enforce would-hold, got %q", got)
	}
	if got := input.EnforceHint.State; got != "hold-before-enforce" {
		t.Fatalf("expected enforce hint hold-before-enforce, got %q", got)
	}
	if got := input.Overview.Reason; got != "overview reason" {
		t.Fatalf("expected overview reason, got %q", got)
	}
}

func TestBuildRolloutReportSections(t *testing.T) {
	sections := BuildRolloutReportSections(RolloutReportSectionsInput{
		Surface: RolloutReportSurfaceInput{
			Summary: RolloutReportSummary{
				ShadowOwnedEvents: 5,
				LegacyOwnedEvents: 6,
				OwnershipChains:   2,
			},
			Mode: "shadow",
			Advisory: RolloutReportAdvisory{
				Decision: "hold",
				Status:   "shadow-observe",
				Ready:    false,
				Reason:   "surface reason",
			},
			Policy: RolloutReportPolicy{
				Mode:         "report-only",
				Action:       "report-hold",
				Reason:       "policy reason",
				Acknowledged: false,
				AckState:     "pending",
			},
			Progression: RolloutReportStateReason{
				State:  "observe",
				Reason: "progression reason",
			},
			CutoverDryRun: RolloutReportAction{
				Action: "would-hold",
				Reason: "dry-run reason",
			},
			CutoverCandidate: RolloutReportCandidate{
				Eligible: false,
				Reason:   "candidate reason",
			},
		},
		Approval: RolloutReportApprovalInput{
			ManualApprovalCheckpoint: RolloutReportStateReason{
				State:  "inactive",
				Reason: "checkpoint reason",
			},
			OperatorHandoff: RolloutReportStateReason{
				State:  "none",
				Reason: "handoff reason",
			},
			WorkItem: RolloutReportApprovalItem{
				Status:       "none",
				Owner:        "none",
				ReviewFields: "field-a,field-b",
				Reason:       "work item reason",
			},
			Checklist: RolloutReportStateReason{
				State:  "incomplete",
				Reason: "checklist reason",
			},
		},
		GuardedCutover: RolloutReportGuardedInput{
			Hook: RolloutReportAction{
				Action: "noop-hold",
				Reason: "hook reason",
			},
			HookPolicy: RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-hold",
				Reason: "policy reason",
			},
			WouldEnforce: RolloutReportAction{
				Action: "would-hold",
				Reason: "would-enforce reason",
			},
			EnforceHint: RolloutReportStateReason{
				State:  "hold-before-enforce",
				Reason: "enforce hint reason",
			},
			Overview: RolloutReportStateReason{
				State:  "hold",
				Reason: "overview reason",
			},
		},
	})

	if got := sections.Surface.Mode; got != "shadow" {
		t.Fatalf("expected surface mode shadow, got %q", got)
	}
	if got := sections.Approval.WorkItem.ReviewFields; got != "field-a,field-b" {
		t.Fatalf("expected approval review fields field-a,field-b, got %q", got)
	}
	if got := sections.GuardedCutover.Overview.State; got != "hold" {
		t.Fatalf("expected guarded overview hold, got %q", got)
	}
}
