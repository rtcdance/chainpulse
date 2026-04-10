package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/api"
)

func TestResolveDeploymentModeProfile(t *testing.T) {
	t.Run("defaults to monolithic", func(t *testing.T) {
		profile := resolveDeploymentModeProfile("")
		if profile.Mode != deploymentModeMonolithic {
			t.Fatalf("expected monolithic mode, got %s", profile.Mode)
		}
		if profile.Posture != "deployment-mode-monolithic" {
			t.Fatalf("expected monolithic posture, got %s", profile.Posture)
		}
	})

	t.Run("accepts microservice", func(t *testing.T) {
		profile := resolveDeploymentModeProfile("microservice")
		if profile.Mode != deploymentModeMicroservice {
			t.Fatalf("expected microservice mode, got %s", profile.Mode)
		}
		if profile.Posture != "deployment-mode-microservice-intent" {
			t.Fatalf("expected microservice posture, got %s", profile.Posture)
		}
	})

	t.Run("falls back for invalid values", func(t *testing.T) {
		profile := resolveDeploymentModeProfile("weird-mode")
		if profile.Mode != deploymentModeMonolithic {
			t.Fatalf("expected fallback monolithic mode, got %s", profile.Mode)
		}
		if profile.Recognized {
			t.Fatal("expected invalid mode to be unrecognized")
		}
		if profile.Posture != "deployment-mode-fallback" {
			t.Fatalf("expected fallback posture, got %s", profile.Posture)
		}
	})
}

func TestResolveMonolithicAdapterProfile(t *testing.T) {
	t.Run("monolithic profile", func(t *testing.T) {
		profile := resolveMonolithicAdapterProfile(deploymentModeMonolithic, nil)
		if profile.ProfileName != "monolithic-runtime-profile" {
			t.Fatalf("expected monolithic profile, got %s", profile.ProfileName)
		}
		if profile.SelectionPosture != "adapter-profile-ready" {
			t.Fatalf("expected ready posture, got %s", profile.SelectionPosture)
		}
		if profile.QueryRuntimeAdapter != "indexing-backed-query-surface" {
			t.Fatalf("expected indexing-backed query adapter, got %s", profile.QueryRuntimeAdapter)
		}
	})

	t.Run("microservice profile remains partial", func(t *testing.T) {
		profile := resolveMonolithicAdapterProfile(deploymentModeMicroservice, nil)
		if profile.ProfileName != "microservice-target-profile" {
			t.Fatalf("expected microservice target profile, got %s", profile.ProfileName)
		}
		if profile.SelectionPosture != "adapter-profile-partial" {
			t.Fatalf("expected partial posture, got %s", profile.SelectionPosture)
		}
		if profile.IndexingStorageAdapter != "compatibility-mock-indexing-storage" {
			t.Fatalf("expected compatibility mock indexing storage, got %s", profile.IndexingStorageAdapter)
		}
	})

	t.Run("microservice profile becomes bridged with upstreams", func(t *testing.T) {
		profile := resolveMonolithicAdapterProfile(deploymentModeMicroservice, []string{"http://localhost:8081"})
		if profile.SelectionPosture != "adapter-profile-bridged" {
			t.Fatalf("expected bridged posture, got %s", profile.SelectionPosture)
		}
		if profile.TransportAdapterBoundary != "upstream-query-bridge-gateway-intent" {
			t.Fatalf("expected upstream query bridge boundary, got %s", profile.TransportAdapterBoundary)
		}
	})
}

func TestClassifyMonolithicTransportBoundary(t *testing.T) {
	cases := []struct {
		name               string
		boundary           string
		gatewaySurfaceMode string
		configured         int
		attached           int
		available          int
		wantPosture        string
	}{
		{
			name:               "monolithic in process ready",
			boundary:           "monolithic-in-process-runtime",
			gatewaySurfaceMode: "full-in-process",
			wantPosture:        "transport-boundary-in-process-ready",
		},
		{
			name:               "microservice runtime only",
			boundary:           "runtime-operator-only-gateway-intent",
			gatewaySurfaceMode: "runtime-operator-only",
			wantPosture:        "transport-boundary-runtime-operator-only",
		},
		{
			name:               "microservice bridge unconfigured",
			boundary:           "upstream-query-bridge-gateway-intent",
			gatewaySurfaceMode: "upstream-query-bridge",
			wantPosture:        "transport-boundary-bridge-unconfigured",
		},
		{
			name:               "microservice bridge unattached",
			boundary:           "upstream-query-bridge-gateway-intent",
			gatewaySurfaceMode: "upstream-query-bridge",
			configured:         1,
			wantPosture:        "transport-boundary-bridge-unattached",
		},
		{
			name:               "microservice bridge unavailable",
			boundary:           "upstream-query-bridge-gateway-intent",
			gatewaySurfaceMode: "upstream-query-bridge",
			configured:         1,
			attached:           1,
			wantPosture:        "transport-boundary-bridge-unavailable",
		},
		{
			name:               "microservice bridge degraded",
			boundary:           "upstream-query-bridge-gateway-intent",
			gatewaySurfaceMode: "upstream-query-bridge",
			configured:         2,
			attached:           2,
			available:          1,
			wantPosture:        "transport-boundary-bridge-degraded",
		},
		{
			name:               "microservice bridge ready",
			boundary:           "upstream-query-bridge-gateway-intent",
			gatewaySurfaceMode: "upstream-query-bridge",
			configured:         1,
			attached:           1,
			available:          1,
			wantPosture:        "transport-boundary-bridge-ready",
		},
		{
			name:               "unclassified fallback",
			boundary:           "unknown-boundary",
			gatewaySurfaceMode: "full-in-process",
			wantPosture:        "transport-boundary-unclassified",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := classifyMonolithicTransportBoundary(tc.boundary, tc.gatewaySurfaceMode, tc.configured, tc.attached, tc.available)
			if status.Posture != tc.wantPosture {
				t.Fatalf("expected %s, got %s", tc.wantPosture, status.Posture)
			}
			if status.Hint == "" {
				t.Fatal("expected transport boundary hint to be populated")
			}
		})
	}
}

func TestLoadConfigurationDeploymentMode(t *testing.T) {
	t.Setenv("DEPLOYMENT_MODE", "microservice")

	cfg := loadConfiguration()
	if cfg.DeploymentMode != deploymentModeMicroservice {
		t.Fatalf("expected deployment mode microservice, got %s", cfg.DeploymentMode)
	}
	if cfg.DeploymentPosture != "deployment-mode-microservice-intent" {
		t.Fatalf("expected deployment posture microservice intent, got %s", cfg.DeploymentPosture)
	}
	if cfg.DeploymentHint == "" {
		t.Fatal("expected deployment hint to be populated")
	}
	if cfg.AdapterProfile != "microservice-target-profile" {
		t.Fatalf("expected adapter profile microservice-target-profile, got %s", cfg.AdapterProfile)
	}
	if cfg.DLQRetention != "168h" {
		t.Fatalf("expected default DLQ retention 168h, got %s", cfg.DLQRetention)
	}
}

func TestParseMonolithicDLQRetention(t *testing.T) {
	t.Run("accepts valid duration", func(t *testing.T) {
		retention, err := parseMonolithicDLQRetention("72h")
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if retention != 72*time.Hour {
			t.Fatalf("expected 72h retention, got %s", retention)
		}
	})

	t.Run("rejects invalid duration", func(t *testing.T) {
		if _, err := parseMonolithicDLQRetention("forever"); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestResolveMonolithicGatewaySurface(t *testing.T) {
	t.Run("monolithic keeps full surface", func(t *testing.T) {
		surface := resolveMonolithicGatewaySurface(Configuration{DeploymentMode: deploymentModeMonolithic})
		if surface.SurfaceMode != "full-in-process" {
			t.Fatalf("expected full-in-process, got %s", surface.SurfaceMode)
		}
		if surface.SurfacePosture != "gateway-surface-full" {
			t.Fatalf("expected gateway-surface-full, got %s", surface.SurfacePosture)
		}
	})

	t.Run("microservice intent keeps runtime-only surface", func(t *testing.T) {
		surface := resolveMonolithicGatewaySurface(Configuration{DeploymentMode: deploymentModeMicroservice})
		if surface.SurfaceMode != "runtime-operator-only" {
			t.Fatalf("expected runtime-operator-only, got %s", surface.SurfaceMode)
		}
		if surface.SurfacePosture != "gateway-surface-runtime-only" {
			t.Fatalf("expected gateway-surface-runtime-only, got %s", surface.SurfacePosture)
		}
	})
}

func TestApplyMonolithicGatewaySurfaceRuntimeOnly(t *testing.T) {
	logger := core.NewDefaultLogger(core.LogLevelError)
	metrics := core.NewDefaultMetricsCollector()
	gateway := api.NewAPIGatewayPlugin(logger, metrics)

	healthHandler := api.NewHealthCheckHandler(nil, logger, metrics)
	applyMonolithicGatewaySurface(gateway, resolveMonolithicGatewaySurface(Configuration{DeploymentMode: deploymentModeMicroservice}), gatewayRuntimeWiring{
		eventQueryHandler:        api.NewEventQueryHandler(nil, logger, metrics),
		eventSubscriptionHandler: api.NewEventSubscriptionHandler(nil, logger, metrics),
		healthCheckHandler:       healthHandler,
	})

	if gateway.IsEventQueryHandlerEnabled() {
		t.Fatal("expected event query handler to stay disabled in runtime-only gateway surface mode")
	}
	if gateway.IsEventSubscriptionHandlerEnabled() {
		t.Fatal("expected subscription handler to stay disabled in runtime-only gateway surface mode")
	}
	if !gateway.IsHealthCheckHandlerEnabled() {
		t.Fatal("expected health handler to stay enabled in runtime-only gateway surface mode")
	}
}

func TestAggregateIndexerOwnership(t *testing.T) {
	summary := aggregateIndexerOwnership(map[string]map[string]interface{}{
		"ethereum": {
			"shadow_owned_events": int64(3),
			"legacy_owned_events": int64(7),
		},
		"polygon": {
			"shadow_owned_events": float64(2),
			"legacy_owned_events": uint64(5),
		},
		"arb": {},
	})

	if summary.ShadowOwnedEvents != 5 {
		t.Fatalf("expected shadow owned 5, got %d", summary.ShadowOwnedEvents)
	}
	if summary.LegacyOwnedEvents != 12 {
		t.Fatalf("expected legacy owned 12, got %d", summary.LegacyOwnedEvents)
	}
	if summary.Chains != 3 {
		t.Fatalf("expected chains 3, got %d", summary.Chains)
	}
}

func TestInt64ValueUnsupportedDefaultsToZero(t *testing.T) {
	if got := int64Value("bad"); got != 0 {
		t.Fatalf("expected zero for unsupported type, got %d", got)
	}
}

func TestClassifyOwnershipMode(t *testing.T) {
	cases := []struct {
		name    string
		summary ownershipSummary
		want    string
	}{
		{name: "idle", summary: ownershipSummary{}, want: "idle"},
		{name: "legacy-only", summary: ownershipSummary{LegacyOwnedEvents: 3}, want: "legacy-only"},
		{name: "runtime-owned", summary: ownershipSummary{ShadowOwnedEvents: 3}, want: "runtime-owned"},
		{name: "shadow", summary: ownershipSummary{ShadowOwnedEvents: 3, LegacyOwnedEvents: 7}, want: "shadow"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipMode(tc.summary)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

func TestOwnershipModeCode(t *testing.T) {
	cases := map[string]float64{
		"idle":          0,
		"legacy-only":   1,
		"shadow":        2,
		"runtime-owned": 3,
		"unknown":       9,
	}

	for mode, want := range cases {
		if got := ownershipModeCode(mode); got != want {
			t.Fatalf("mode %s expected %v, got %v", mode, want, got)
		}
	}
}

func TestEmitOwnershipRolloutSummaryMetrics(t *testing.T) {
	metrics := core.NewDefaultMetricsCollector()
	snapshot := ownershipRolloutSummarySnapshot{
		Summary: ownershipSummary{
			ShadowOwnedEvents: 4,
			LegacyOwnedEvents: 9,
			Chains:            2,
		},
		Mode:          "shadow",
		Policy:        ownershipRolloutPolicy{Mode: "manual-gate", AckState: "acknowledged"},
		Progression:   ownershipEffectiveProgression{State: "ready-for-cutover"},
		CutoverDryRun: ownershipCutoverDryRun{Action: "would-allow"},
		CutoverCandidate: ownershipCutoverCandidate{
			Eligible: true,
			Reason:   "manual gate is acknowledged and dry-run cutover would allow progression",
		},
		ManualApprovalCheckpoint: ownershipManualApprovalCheckpoint{
			State:  "awaiting-approval",
			Reason: "instance is a cutover candidate and is awaiting manual approval checkpoint",
		},
		OperatorHandoff: ownershipOperatorHandoff{
			State:  "operator-review",
			Reason: "manual approval checkpoint is active and requires operator review",
		},
		ApprovalWorkItem: ownershipApprovalWorkItem{
			Status:       "open",
			Owner:        "platform-team/manual-approver",
			ReviewFields: "rollout_effective_state,rollout_cutover_candidate,rollout_manual_approval_checkpoint_state,rollout_operator_handoff_state",
			Reason:       "operator handoff requires approval review before any future cutover action",
		},
		ApprovalChecklist: ownershipApprovalChecklist{
			State:  "ready",
			Reason: "approval checklist prerequisites are present for manual review",
		},
		GuardedCutoverHook: ownershipGuardedCutoverHook{
			Action: "noop-allow",
			Reason: "guarded cutover hook would allow progression, but remains non-blocking in dry-run mode",
		},
		GuardedCutoverHookPolicy: ownershipGuardedCutoverHookPolicy{
			Mode:   "enforce-ready",
			Action: "enforce-would-allow",
			Reason: "enforce-ready policy is configured for the guarded cutover hook, but remains advisory until execution gating is introduced",
		},
		GuardedCutoverWouldEnforce: ownershipGuardedCutoverWouldEnforce{
			Action: "would-allow",
			Reason: "future enforcement posture would allow this instance to proceed",
		},
		GuardedCutoverEnforceHint: ownershipGuardedCutoverEnforceHint{
			State:  "safe-to-observe",
			Reason: "future enforcement posture is healthy enough to keep observing toward eventual enforcement",
		},
		GuardedCutoverOverview: ownershipGuardedCutoverOverview{
			State:  "observe",
			Reason: "guarded cutover posture is healthy enough to keep observing toward possible enforcement",
		},
	}

	emitOwnershipRolloutSummaryMetrics(metrics, snapshot, "running")

	tags := map[string]string{
		"service":   "monolithic",
		"operation": "running",
	}
	if got := metrics.GetGauge("indexing_runtime_shadow_owned_events", tags); got != 4 {
		t.Fatalf("expected shadow owned gauge 4, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_legacy_owned_events", tags); got != 9 {
		t.Fatalf("expected legacy owned gauge 9, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_ownership_chains", tags); got != 2 {
		t.Fatalf("expected ownership chains gauge 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_ownership_mode_code", tags); got != 2 {
		t.Fatalf("expected ownership mode code 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_policy_mode_code", tags); got != 2 {
		t.Fatalf("expected rollout policy mode code 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_policy_ack_state_code", tags); got != 1 {
		t.Fatalf("expected rollout policy ack state code 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_effective_state_code", tags); got != 4 {
		t.Fatalf("expected rollout effective state code 4, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_cutover_dry_run_code", tags); got != 2 {
		t.Fatalf("expected rollout cutover dry-run code 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_cutover_candidate", tags); got != 1 {
		t.Fatalf("expected rollout cutover candidate gauge 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_manual_approval_checkpoint_code", tags); got != 1 {
		t.Fatalf("expected rollout manual approval checkpoint gauge 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_operator_handoff_code", tags); got != 1 {
		t.Fatalf("expected rollout operator handoff gauge 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_approval_work_item_code", tags); got != 1 {
		t.Fatalf("expected rollout approval work item gauge 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_approval_checklist_code", tags); got != 1 {
		t.Fatalf("expected rollout approval checklist gauge 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_guarded_cutover_hook_code", tags); got != 2 {
		t.Fatalf("expected rollout guarded cutover hook gauge 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_guarded_cutover_hook_policy_mode_code", tags); got != 2 {
		t.Fatalf("expected rollout guarded cutover hook policy mode gauge 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_guarded_cutover_would_enforce_code", tags); got != 2 {
		t.Fatalf("expected rollout guarded cutover would-enforce gauge 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_guarded_cutover_enforce_hint_code", tags); got != 1 {
		t.Fatalf("expected rollout guarded cutover enforce hint gauge 1, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_guarded_cutover_overview_code", tags); got != 1 {
		t.Fatalf("expected rollout guarded cutover overview gauge 1, got %v", got)
	}
}

func TestOwnershipGuardedCutoverSummaryHelper(t *testing.T) {
	t.Setenv("CHAINPULSE_OWNERSHIP_GUARDED_CUTOVER_HOOK_POLICY_MODE", "enforce-ready")

	summary := buildOwnershipGuardedCutoverSummary(
		ownershipCutoverDryRun{Action: "would-allow"},
		ownershipCutoverCandidate{Eligible: true},
		ownershipApprovalChecklist{State: "ready"},
	)

	if summary.Hook.Action != "noop-allow" {
		t.Fatalf("expected hook noop-allow, got %s", summary.Hook.Action)
	}
	if summary.HookPolicy.Mode != "enforce-ready" {
		t.Fatalf("expected policy mode enforce-ready, got %s", summary.HookPolicy.Mode)
	}
	if summary.WouldEnforce.Action != "would-allow" {
		t.Fatalf("expected would-enforce would-allow, got %s", summary.WouldEnforce.Action)
	}
	if summary.EnforceHint.State != "safe-to-observe" {
		t.Fatalf("expected enforce hint safe-to-observe, got %s", summary.EnforceHint.State)
	}
	if summary.Overview.State != "observe" {
		t.Fatalf("expected overview observe, got %s", summary.Overview.State)
	}

	details := map[string]interface{}{}
	summary.applyReadinessDetails(details)
	if got := details["rollout_guarded_cutover_overview_state"]; got != "observe" {
		t.Fatalf("expected overview state observe in details, got %v", got)
	}

	metrics := core.NewDefaultMetricsCollector()
	tags := map[string]string{"service": "monolithic", "operation": "unit"}
	summary.emitMetrics(metrics, tags)
	if got := metrics.GetGauge("indexing_runtime_rollout_guarded_cutover_overview_code", tags); got != 1 {
		t.Fatalf("expected guarded cutover overview gauge 1, got %v", got)
	}
}

func TestOwnershipApprovalSummaryHelper(t *testing.T) {
	summary := buildOwnershipApprovalSummary(
		ownershipEffectiveProgression{State: "ready-for-cutover"},
		ownershipCutoverCandidate{Eligible: true},
	)

	if summary.ManualApprovalCheckpoint.State != "awaiting-approval" {
		t.Fatalf("expected checkpoint awaiting-approval, got %s", summary.ManualApprovalCheckpoint.State)
	}
	if summary.OperatorHandoff.State != "operator-review" {
		t.Fatalf("expected operator handoff operator-review, got %s", summary.OperatorHandoff.State)
	}
	if summary.ApprovalWorkItem.Status != "open" {
		t.Fatalf("expected approval work item open, got %s", summary.ApprovalWorkItem.Status)
	}
	if summary.ApprovalChecklist.State != "ready" {
		t.Fatalf("expected approval checklist ready, got %s", summary.ApprovalChecklist.State)
	}

	details := map[string]interface{}{}
	summary.applyReadinessDetails(details)
	if got := details["rollout_approval_checklist_state"]; got != "ready" {
		t.Fatalf("expected approval checklist ready in details, got %v", got)
	}

	metrics := core.NewDefaultMetricsCollector()
	tags := map[string]string{"service": "monolithic", "operation": "unit"}
	summary.emitMetrics(metrics, tags)
	if got := metrics.GetGauge("indexing_runtime_rollout_approval_checklist_code", tags); got != 1 {
		t.Fatalf("expected approval checklist gauge 1, got %v", got)
	}
}

func TestOwnershipRolloutSurfaceHelper(t *testing.T) {
	surface := ownershipRolloutSurface{
		Summary: ownershipSummary{
			ShadowOwnedEvents: 3,
			LegacyOwnedEvents: 1,
			Chains:            2,
		},
		Mode:             "shadow",
		Advisory:         ownershipRolloutAdvisory{Decision: "hold", Status: "shadow-observe", Ready: false, Reason: "shared runtime still coexists with legacy writes"},
		Policy:           ownershipRolloutPolicy{Mode: "manual-gate", Action: "manual-review-hold", Reason: "manual-gate mode requires operator review before ownership progression", AckState: "pending"},
		Progression:      ownershipEffectiveProgression{State: "review-required", Reason: "manual gate requires operator review before progression"},
		CutoverDryRun:    ownershipCutoverDryRun{Action: "would-hold", Reason: "effective progression state does not yet satisfy cutover conditions in dry-run mode"},
		CutoverCandidate: ownershipCutoverCandidate{Eligible: false, Reason: "cutover candidate requires recorded operator acknowledgment"},
	}

	details := map[string]interface{}{}
	surface.applyReadinessDetails(details)
	if got := details["rollout_policy_mode"]; got != "manual-gate" {
		t.Fatalf("expected rollout policy mode manual-gate, got %v", got)
	}
	if got := details["rollout_cutover_candidate"]; got != false {
		t.Fatalf("expected rollout cutover candidate false, got %v", got)
	}

	metrics := core.NewDefaultMetricsCollector()
	tags := map[string]string{"service": "monolithic", "operation": "unit"}
	surface.emitMetrics(metrics, tags)
	if got := metrics.GetGauge("indexing_runtime_ownership_mode_code", tags); got != 2 {
		t.Fatalf("expected ownership mode code 2, got %v", got)
	}
	if got := metrics.GetGauge("indexing_runtime_rollout_cutover_dry_run_code", tags); got != 1 {
		t.Fatalf("expected cutover dry-run code 1, got %v", got)
	}
}

func TestBuildOwnershipRolloutSummarySections(t *testing.T) {
	sections := buildOwnershipRolloutSummarySections(
		ownershipSummary{ShadowOwnedEvents: 2, LegacyOwnedEvents: 1, Chains: 1},
		"shadow",
		ownershipRolloutAdvisory{Decision: "hold", Status: "shadow-observe", Ready: false, Reason: "shared runtime still coexists with legacy writes"},
		ownershipRolloutPolicy{Mode: "manual-gate", Action: "manual-review-hold", Reason: "manual-gate mode requires operator review before ownership progression", AckState: "pending"},
		ownershipEffectiveProgression{State: "review-required", Reason: "manual gate requires operator review before progression"},
		ownershipCutoverDryRun{Action: "would-hold", Reason: "effective progression state does not yet satisfy cutover conditions in dry-run mode"},
		ownershipCutoverCandidate{Eligible: false, Reason: "cutover candidate requires recorded operator acknowledgment"},
	)

	if sections.Surface.Mode != "shadow" {
		t.Fatalf("expected surface mode shadow, got %s", sections.Surface.Mode)
	}
	if sections.Approval.ApprovalChecklist.State != "incomplete" {
		t.Fatalf("expected approval checklist incomplete, got %s", sections.Approval.ApprovalChecklist.State)
	}
	if sections.Guarded.Overview.State != "hold" {
		t.Fatalf("expected guarded overview hold, got %s", sections.Guarded.Overview.State)
	}
}

func TestBuildOwnershipRolloutSummary(t *testing.T) {
	t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_POLICY_MODE", "manual-gate")
	t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_ACKNOWLEDGED", "true")
	t.Setenv("CHAINPULSE_OWNERSHIP_GUARDED_CUTOVER_HOOK_POLICY_MODE", "enforce-ready")

	snapshot := buildOwnershipRolloutSummary(map[string]map[string]interface{}{
		"ethereum": {
			"shadow_owned_events": int64(5),
			"legacy_owned_events": int64(0),
		},
	})

	if snapshot.Mode != "runtime-owned" {
		t.Fatalf("expected mode runtime-owned, got %s", snapshot.Mode)
	}
	if snapshot.Advisory.Decision != "allow" {
		t.Fatalf("expected advisory allow, got %s", snapshot.Advisory.Decision)
	}
	if snapshot.Policy.Mode != "manual-gate" {
		t.Fatalf("expected policy manual-gate, got %s", snapshot.Policy.Mode)
	}
	if snapshot.Progression.State != "ready-for-cutover" {
		t.Fatalf("expected progression ready-for-cutover, got %s", snapshot.Progression.State)
	}
	if !snapshot.CutoverCandidate.Eligible {
		t.Fatalf("expected cutover candidate eligible")
	}
	if snapshot.ManualApprovalCheckpoint.State != "awaiting-approval" {
		t.Fatalf("expected checkpoint awaiting-approval, got %s", snapshot.ManualApprovalCheckpoint.State)
	}
	if snapshot.OperatorHandoff.State != "operator-review" {
		t.Fatalf("expected operator handoff operator-review, got %s", snapshot.OperatorHandoff.State)
	}
	if snapshot.ApprovalWorkItem.Status != "open" {
		t.Fatalf("expected approval work item open, got %s", snapshot.ApprovalWorkItem.Status)
	}
	if snapshot.ApprovalChecklist.State != "ready" {
		t.Fatalf("expected approval checklist ready, got %s", snapshot.ApprovalChecklist.State)
	}
	if snapshot.GuardedCutoverHook.Action != "noop-allow" {
		t.Fatalf("expected guarded cutover hook noop-allow, got %s", snapshot.GuardedCutoverHook.Action)
	}
	if snapshot.GuardedCutoverHookPolicy.Mode != "enforce-ready" {
		t.Fatalf("expected guarded cutover hook policy enforce-ready, got %s", snapshot.GuardedCutoverHookPolicy.Mode)
	}
	if snapshot.GuardedCutoverHookPolicy.Action != "enforce-would-allow" {
		t.Fatalf("expected guarded cutover hook policy action enforce-would-allow, got %s", snapshot.GuardedCutoverHookPolicy.Action)
	}
	if snapshot.GuardedCutoverWouldEnforce.Action != "would-allow" {
		t.Fatalf("expected guarded cutover would-enforce would-allow, got %s", snapshot.GuardedCutoverWouldEnforce.Action)
	}
	if snapshot.GuardedCutoverEnforceHint.State != "safe-to-observe" {
		t.Fatalf("expected guarded cutover enforce hint safe-to-observe, got %s", snapshot.GuardedCutoverEnforceHint.State)
	}
	if snapshot.GuardedCutoverOverview.State != "observe" {
		t.Fatalf("expected guarded cutover overview observe, got %s", snapshot.GuardedCutoverOverview.State)
	}
}

func TestBuildOwnershipHealthComponent(t *testing.T) {
	now := time.Unix(1700000000, 0)

	component := buildOwnershipHealthComponent(map[string]map[string]interface{}{
		"ethereum": {
			"shadow_owned_events": int64(4),
			"legacy_owned_events": int64(6),
		},
		"polygon": {
			"shadow_owned_events": int64(2),
			"legacy_owned_events": int64(0),
		},
	}, now)

	if component == nil {
		t.Fatal("expected component")
	}
	if component.Name != "Indexing Runtime" {
		t.Fatalf("expected name Indexing Runtime, got %s", component.Name)
	}
	if component.Status != "healthy" {
		t.Fatalf("expected healthy status, got %s", component.Status)
	}
	if component.Timestamp != now.Unix() {
		t.Fatalf("expected timestamp %d, got %d", now.Unix(), component.Timestamp)
	}
	if got := component.Details["ownership_mode"]; got != "shadow" {
		t.Fatalf("expected ownership mode shadow, got %v", got)
	}
	if got := component.Details["shadow_owned_events"]; got != int64(6) {
		t.Fatalf("expected shadow_owned_events 6, got %v", got)
	}
	if got := component.Details["legacy_owned_events"]; got != int64(6) {
		t.Fatalf("expected legacy_owned_events 6, got %v", got)
	}
	if got := component.Details["ownership_chains"]; got != 2 {
		t.Fatalf("expected ownership_chains 2, got %v", got)
	}
	if got := component.Details["service"]; got != "monolithic" {
		t.Fatalf("expected service monolithic, got %v", got)
	}
}

func TestBuildOwnershipHealthComponentUnknownModeIsDegraded(t *testing.T) {
	component := buildOwnershipHealthComponent(map[string]map[string]interface{}{
		"ethereum": {
			"shadow_owned_events": "bad",
			"legacy_owned_events": int64(0),
		},
	}, time.Unix(1700000001, 0))

	if component.Status != "healthy" {
		t.Fatalf("expected malformed counters to coerce to a known healthy mode, got %s", component.Status)
	}
	if got := component.Details["ownership_mode"]; got != "idle" {
		t.Fatalf("expected idle ownership mode fallback, got %v", got)
	}
}

func TestBuildOwnershipHealthComponentMatchesAPIComponentContract(t *testing.T) {
	_ = buildOwnershipHealthComponent(nil, time.Unix(1700000002, 0))
}

func TestBuildOwnershipReadinessDetailsShadow(t *testing.T) {
	details := buildOwnershipReadinessDetails(map[string]map[string]interface{}{
		"ethereum": {
			"shadow_owned_events": int64(3),
			"legacy_owned_events": int64(4),
		},
	})

	if got := details["ownership_mode"]; got != "shadow" {
		t.Fatalf("expected ownership_mode shadow, got %v", got)
	}
	if got := details["rollout_ready_for_runtime_owned"]; got != false {
		t.Fatalf("expected rollout readiness false, got %v", got)
	}
	if got := details["rollout_status"]; got != "shadow-observe" {
		t.Fatalf("expected rollout_status shadow-observe, got %v", got)
	}
	if got := details["rollout_reason"]; got != "shared runtime still coexists with legacy writes" {
		t.Fatalf("unexpected rollout_reason %v", got)
	}
	if got := details["rollout_gate_decision"]; got != "hold" {
		t.Fatalf("expected rollout_gate_decision hold, got %v", got)
	}
	if got := details["rollout_policy_mode"]; got != "report-only" {
		t.Fatalf("expected rollout_policy_mode report-only, got %v", got)
	}
	if got := details["rollout_policy_action"]; got != "report-hold" {
		t.Fatalf("expected rollout_policy_action report-hold, got %v", got)
	}
	if got := details["rollout_policy_ack_state"]; got != "pending" {
		t.Fatalf("expected rollout_policy_ack_state pending, got %v", got)
	}
	if got := details["rollout_effective_state"]; got != "observe" {
		t.Fatalf("expected rollout_effective_state observe, got %v", got)
	}
	if got := details["rollout_manual_approval_checkpoint_state"]; got != "inactive" {
		t.Fatalf("expected manual approval checkpoint inactive, got %v", got)
	}
	if got := details["rollout_operator_handoff_state"]; got != "none" {
		t.Fatalf("expected operator handoff none, got %v", got)
	}
	if got := details["rollout_approval_work_item_status"]; got != "none" {
		t.Fatalf("expected approval work item none, got %v", got)
	}
	if got := details["rollout_approval_checklist_state"]; got != "incomplete" {
		t.Fatalf("expected approval checklist incomplete, got %v", got)
	}
	if got := details["rollout_guarded_cutover_hook_action"]; got != "noop-hold" {
		t.Fatalf("expected guarded cutover hook noop-hold, got %v", got)
	}
	if got := details["rollout_guarded_cutover_hook_policy_mode"]; got != "noop-only" {
		t.Fatalf("expected guarded hook policy mode noop-only, got %v", got)
	}
	if got := details["rollout_guarded_cutover_would_enforce_action"]; got != "would-hold" {
		t.Fatalf("expected guarded cutover would-enforce would-hold, got %v", got)
	}
	if got := details["rollout_guarded_cutover_enforce_hint_state"]; got != "hold-before-enforce" {
		t.Fatalf("expected guarded cutover enforce hint hold-before-enforce, got %v", got)
	}
	if got := details["rollout_guarded_cutover_overview_state"]; got != "hold" {
		t.Fatalf("expected guarded cutover overview hold, got %v", got)
	}
}

func TestBuildOwnershipReadinessDetailsRuntimeOwned(t *testing.T) {
	details := buildOwnershipReadinessDetails(map[string]map[string]interface{}{
		"ethereum": {
			"shadow_owned_events": int64(5),
			"legacy_owned_events": int64(0),
		},
	})

	if got := details["ownership_mode"]; got != "runtime-owned" {
		t.Fatalf("expected ownership_mode runtime-owned, got %v", got)
	}
	if got := details["rollout_ready_for_runtime_owned"]; got != true {
		t.Fatalf("expected rollout readiness true, got %v", got)
	}
	if got := details["rollout_status"]; got != "ready" {
		t.Fatalf("expected rollout_status ready, got %v", got)
	}
	if got := details["rollout_reason"]; got != "shared runtime owns observed writes" {
		t.Fatalf("unexpected rollout_reason %v", got)
	}
	if got := details["rollout_gate_decision"]; got != "allow" {
		t.Fatalf("expected rollout_gate_decision allow, got %v", got)
	}
	if got := details["rollout_policy_action"]; got != "report-allow" {
		t.Fatalf("expected rollout_policy_action report-allow, got %v", got)
	}
	if got := details["rollout_effective_state"]; got != "observe" {
		t.Fatalf("expected rollout_effective_state observe, got %v", got)
	}
	if got := details["rollout_manual_approval_checkpoint_state"]; got != "inactive" {
		t.Fatalf("expected manual approval checkpoint inactive, got %v", got)
	}
	if got := details["rollout_operator_handoff_state"]; got != "none" {
		t.Fatalf("expected operator handoff none, got %v", got)
	}
	if got := details["rollout_approval_work_item_status"]; got != "none" {
		t.Fatalf("expected approval work item none, got %v", got)
	}
	if got := details["rollout_approval_checklist_state"]; got != "incomplete" {
		t.Fatalf("expected approval checklist incomplete, got %v", got)
	}
	if got := details["rollout_guarded_cutover_hook_action"]; got != "noop-hold" {
		t.Fatalf("expected guarded cutover hook noop-hold, got %v", got)
	}
	if got := details["rollout_guarded_cutover_hook_policy_mode"]; got != "noop-only" {
		t.Fatalf("expected guarded hook policy mode noop-only, got %v", got)
	}
	if got := details["rollout_guarded_cutover_would_enforce_action"]; got != "would-hold" {
		t.Fatalf("expected guarded cutover would-enforce would-hold, got %v", got)
	}
	if got := details["rollout_guarded_cutover_enforce_hint_state"]; got != "hold-before-enforce" {
		t.Fatalf("expected guarded cutover enforce hint hold-before-enforce, got %v", got)
	}
	if got := details["rollout_guarded_cutover_overview_state"]; got != "hold" {
		t.Fatalf("expected guarded cutover overview hold, got %v", got)
	}
}

func TestClassifyOwnershipRolloutAdvisoryUnknown(t *testing.T) {
	advisory := classifyOwnershipRolloutAdvisory(ownershipSummary{
		ShadowOwnedEvents: -1,
		LegacyOwnedEvents: -1,
		Chains:            1,
	})

	if advisory.Decision != "unknown" {
		t.Fatalf("expected decision unknown, got %s", advisory.Decision)
	}
	if advisory.Status != "unknown" {
		t.Fatalf("expected status unknown, got %s", advisory.Status)
	}
	if advisory.Ready {
		t.Fatalf("expected ready false")
	}
	if advisory.Reason != "ownership rollout state is unknown" {
		t.Fatalf("unexpected reason %s", advisory.Reason)
	}
}

func TestNormalizeOwnershipRolloutPolicyModeDefaultsToReportOnly(t *testing.T) {
	cases := []string{"", "report-only", "report_only", "report", "weird"}
	for _, raw := range cases {
		if got := normalizeOwnershipRolloutPolicyMode(raw); got != "report-only" {
			t.Fatalf("raw=%q expected report-only, got %s", raw, got)
		}
	}
}

func TestNormalizeOwnershipRolloutPolicyModeManualGate(t *testing.T) {
	cases := []string{"manual-gate", "manual_gate", "manual"}
	for _, raw := range cases {
		if got := normalizeOwnershipRolloutPolicyMode(raw); got != "manual-gate" {
			t.Fatalf("raw=%q expected manual-gate, got %s", raw, got)
		}
	}
}

func TestOwnershipProgressionMetricCodes(t *testing.T) {
	if got := ownershipPolicyModeCode("report-only"); got != 1 {
		t.Fatalf("expected report-only code 1, got %v", got)
	}
	if got := ownershipPolicyModeCode("manual-gate"); got != 2 {
		t.Fatalf("expected manual-gate code 2, got %v", got)
	}
	if got := ownershipAckStateCode("pending"); got != 0 {
		t.Fatalf("expected pending code 0, got %v", got)
	}
	if got := ownershipAckStateCode("acknowledged"); got != 1 {
		t.Fatalf("expected acknowledged code 1, got %v", got)
	}
	if got := ownershipEffectiveProgressionCode("observe"); got != 1 {
		t.Fatalf("expected observe code 1, got %v", got)
	}
	if got := ownershipEffectiveProgressionCode("review-required"); got != 2 {
		t.Fatalf("expected review-required code 2, got %v", got)
	}
	if got := ownershipEffectiveProgressionCode("acknowledged"); got != 3 {
		t.Fatalf("expected acknowledged code 3, got %v", got)
	}
	if got := ownershipEffectiveProgressionCode("ready-for-cutover"); got != 4 {
		t.Fatalf("expected ready-for-cutover code 4, got %v", got)
	}
	if got := ownershipEffectiveProgressionCode("unknown"); got != 9 {
		t.Fatalf("expected unknown code 9, got %v", got)
	}
	if got := ownershipCutoverDryRunCode("would-hold"); got != 1 {
		t.Fatalf("expected would-hold code 1, got %v", got)
	}
	if got := ownershipCutoverDryRunCode("would-allow"); got != 2 {
		t.Fatalf("expected would-allow code 2, got %v", got)
	}
	if got := ownershipCutoverDryRunCode("would-unknown"); got != 9 {
		t.Fatalf("expected would-unknown code 9, got %v", got)
	}
	if got := ownershipCutoverCandidateCode(ownershipCutoverCandidate{Eligible: false}); got != 0 {
		t.Fatalf("expected ineligible candidate code 0, got %v", got)
	}
	if got := ownershipCutoverCandidateCode(ownershipCutoverCandidate{Eligible: true}); got != 1 {
		t.Fatalf("expected eligible candidate code 1, got %v", got)
	}
	if got := ownershipManualApprovalCheckpointCode(ownershipManualApprovalCheckpoint{State: "inactive"}); got != 0 {
		t.Fatalf("expected inactive checkpoint code 0, got %v", got)
	}
	if got := ownershipManualApprovalCheckpointCode(ownershipManualApprovalCheckpoint{State: "awaiting-approval"}); got != 1 {
		t.Fatalf("expected awaiting-approval checkpoint code 1, got %v", got)
	}
	if got := ownershipManualApprovalCheckpointCode(ownershipManualApprovalCheckpoint{State: "unknown"}); got != 9 {
		t.Fatalf("expected unknown checkpoint code 9, got %v", got)
	}
	if got := ownershipOperatorHandoffCode(ownershipOperatorHandoff{State: "none"}); got != 0 {
		t.Fatalf("expected no handoff code 0, got %v", got)
	}
	if got := ownershipOperatorHandoffCode(ownershipOperatorHandoff{State: "operator-review"}); got != 1 {
		t.Fatalf("expected operator-review handoff code 1, got %v", got)
	}
	if got := ownershipOperatorHandoffCode(ownershipOperatorHandoff{State: "investigate"}); got != 9 {
		t.Fatalf("expected investigate handoff code 9, got %v", got)
	}
	if got := ownershipApprovalWorkItemCode(ownershipApprovalWorkItem{Status: "none"}); got != 0 {
		t.Fatalf("expected none work item code 0, got %v", got)
	}
	if got := ownershipApprovalWorkItemCode(ownershipApprovalWorkItem{Status: "open"}); got != 1 {
		t.Fatalf("expected open work item code 1, got %v", got)
	}
	if got := ownershipApprovalWorkItemCode(ownershipApprovalWorkItem{Status: "investigate"}); got != 9 {
		t.Fatalf("expected investigate work item code 9, got %v", got)
	}
	if got := ownershipApprovalChecklistCode(ownershipApprovalChecklist{State: "incomplete"}); got != 0 {
		t.Fatalf("expected incomplete checklist code 0, got %v", got)
	}
	if got := ownershipApprovalChecklistCode(ownershipApprovalChecklist{State: "ready"}); got != 1 {
		t.Fatalf("expected ready checklist code 1, got %v", got)
	}
	if got := ownershipApprovalChecklistCode(ownershipApprovalChecklist{State: "investigate"}); got != 9 {
		t.Fatalf("expected investigate checklist code 9, got %v", got)
	}
	if got := ownershipGuardedCutoverHookCode(ownershipGuardedCutoverHook{Action: "noop-hold"}); got != 1 {
		t.Fatalf("expected noop-hold guarded hook code 1, got %v", got)
	}
	if got := ownershipGuardedCutoverHookCode(ownershipGuardedCutoverHook{Action: "noop-allow"}); got != 2 {
		t.Fatalf("expected noop-allow guarded hook code 2, got %v", got)
	}
	if got := ownershipGuardedCutoverHookCode(ownershipGuardedCutoverHook{Action: "noop-investigate"}); got != 9 {
		t.Fatalf("expected noop-investigate guarded hook code 9, got %v", got)
	}
	if got := ownershipGuardedCutoverHookPolicyModeCode("noop-only"); got != 1 {
		t.Fatalf("expected noop-only guarded hook policy mode code 1, got %v", got)
	}
	if got := ownershipGuardedCutoverHookPolicyModeCode("enforce-ready"); got != 2 {
		t.Fatalf("expected enforce-ready guarded hook policy mode code 2, got %v", got)
	}
	if got := ownershipGuardedCutoverHookPolicyModeCode("weird"); got != 9 {
		t.Fatalf("expected unknown guarded hook policy mode code 9, got %v", got)
	}
	if got := ownershipGuardedCutoverWouldEnforceCode(ownershipGuardedCutoverWouldEnforce{Action: "would-hold"}); got != 1 {
		t.Fatalf("expected would-hold guarded cutover would-enforce code 1, got %v", got)
	}
	if got := ownershipGuardedCutoverWouldEnforceCode(ownershipGuardedCutoverWouldEnforce{Action: "would-allow"}); got != 2 {
		t.Fatalf("expected would-allow guarded cutover would-enforce code 2, got %v", got)
	}
	if got := ownershipGuardedCutoverWouldEnforceCode(ownershipGuardedCutoverWouldEnforce{Action: "would-investigate"}); got != 9 {
		t.Fatalf("expected would-investigate guarded cutover would-enforce code 9, got %v", got)
	}
	if got := ownershipGuardedCutoverEnforceHintCode(ownershipGuardedCutoverEnforceHint{State: "safe-to-observe"}); got != 1 {
		t.Fatalf("expected safe-to-observe enforce hint code 1, got %v", got)
	}
	if got := ownershipGuardedCutoverEnforceHintCode(ownershipGuardedCutoverEnforceHint{State: "hold-before-enforce"}); got != 2 {
		t.Fatalf("expected hold-before-enforce enforce hint code 2, got %v", got)
	}
	if got := ownershipGuardedCutoverEnforceHintCode(ownershipGuardedCutoverEnforceHint{State: "investigate-before-enforce"}); got != 9 {
		t.Fatalf("expected investigate-before-enforce enforce hint code 9, got %v", got)
	}
	if got := ownershipGuardedCutoverOverviewCode(ownershipGuardedCutoverOverview{State: "observe"}); got != 1 {
		t.Fatalf("expected observe overview code 1, got %v", got)
	}
	if got := ownershipGuardedCutoverOverviewCode(ownershipGuardedCutoverOverview{State: "hold"}); got != 2 {
		t.Fatalf("expected hold overview code 2, got %v", got)
	}
	if got := ownershipGuardedCutoverOverviewCode(ownershipGuardedCutoverOverview{State: "investigate"}); got != 9 {
		t.Fatalf("expected investigate overview code 9, got %v", got)
	}
}

func TestOwnershipProgressionConsoleSummaryFormatting(t *testing.T) {
	snapshot := ownershipRolloutSummarySnapshot{
		Summary: ownershipSummary{ShadowOwnedEvents: 4, LegacyOwnedEvents: 2, Chains: 1},
		Mode:    "shadow",
		Progression: ownershipEffectiveProgression{
			State:  "ready-for-cutover",
			Reason: "manual gate is acknowledged and advisory allows progression",
		},
		CutoverDryRun: ownershipCutoverDryRun{
			Action: "would-allow",
			Reason: "effective progression state indicates the service would allow cutover in dry-run mode",
		},
		CutoverCandidate: ownershipCutoverCandidate{
			Eligible: true,
			Reason:   "manual gate is acknowledged and dry-run cutover would allow progression",
		},
		ManualApprovalCheckpoint: ownershipManualApprovalCheckpoint{
			State:  "awaiting-approval",
			Reason: "instance is a cutover candidate and is awaiting manual approval checkpoint",
		},
		OperatorHandoff: ownershipOperatorHandoff{
			State:  "operator-review",
			Reason: "manual approval checkpoint is active and requires operator review",
		},
		ApprovalWorkItem: ownershipApprovalWorkItem{
			Status:       "open",
			Owner:        "platform-team/manual-approver",
			ReviewFields: "rollout_effective_state,rollout_cutover_candidate,rollout_manual_approval_checkpoint_state,rollout_operator_handoff_state",
			Reason:       "operator handoff requires approval review before any future cutover action",
		},
		ApprovalChecklist: ownershipApprovalChecklist{
			State:  "ready",
			Reason: "approval checklist prerequisites are present for manual review",
		},
		GuardedCutoverHook: ownershipGuardedCutoverHook{
			Action: "noop-allow",
			Reason: "guarded cutover hook would allow progression, but remains non-blocking in dry-run mode",
		},
		GuardedCutoverHookPolicy: ownershipGuardedCutoverHookPolicy{
			Mode:   "enforce-ready",
			Action: "enforce-would-allow",
			Reason: "enforce-ready policy is configured for the guarded cutover hook, but remains advisory until execution gating is introduced",
		},
		GuardedCutoverWouldEnforce: ownershipGuardedCutoverWouldEnforce{
			Action: "would-allow",
			Reason: "future enforcement posture would allow this instance to proceed",
		},
		GuardedCutoverEnforceHint: ownershipGuardedCutoverEnforceHint{
			State:  "safe-to-observe",
			Reason: "future enforcement posture is healthy enough to keep observing toward eventual enforcement",
		},
		GuardedCutoverOverview: ownershipGuardedCutoverOverview{
			State:  "observe",
			Reason: "guarded cutover posture is healthy enough to keep observing toward possible enforcement",
		},
	}

	var running bytes.Buffer
	var shutdown bytes.Buffer
	printOwnershipRolloutSummary(&running, snapshot, "running")
	printOwnershipRolloutSummary(&shutdown, snapshot, "shutdown")

	for _, line := range []string{
		"Shadow-Owned Events: 4",
		"Legacy-Owned Events: 2",
		"Ownership Chains: 1",
		"Ownership Mode: shadow",
		"Rollout Progression: ready-for-cutover",
		"Rollout Progression Reason: manual gate is acknowledged and advisory allows progression",
		"Cutover Dry-Run: would-allow",
		"Cutover Candidate: true",
		"Manual Approval Checkpoint: awaiting-approval",
		"Operator Handoff: operator-review",
		"Approval Work Item Owner: platform-team/manual-approver",
		"Approval Checklist: ready",
		"Guarded Cutover Hook Policy Mode: enforce-ready",
		"Guarded Cutover Would-Enforce: would-allow",
		"Guarded Cutover Enforce Hint: safe-to-observe",
		"Guarded Cutover Overview: observe",
	} {
		if !strings.Contains(running.String(), line) {
			t.Fatalf("expected running rollout summary to contain %q", line)
		}
	}

	for _, line := range []string{
		"Shadow-owned events: 4",
		"Legacy-owned events: 2",
		"Ownership chains: 1",
		"Ownership mode: shadow",
		"Rollout progression: ready-for-cutover",
		"Rollout progression reason: manual gate is acknowledged and advisory allows progression",
		"Cutover dry-run: would-allow",
		"Cutover candidate: true",
		"Manual approval checkpoint: awaiting-approval",
		"Operator handoff: operator-review",
		"Approval work item owner: platform-team/manual-approver",
		"Approval checklist: ready",
		"Guarded cutover hook policy mode: enforce-ready",
		"Guarded cutover would-enforce: would-allow",
		"Guarded cutover enforce hint: safe-to-observe",
		"Guarded cutover overview: observe",
	} {
		if !strings.Contains(shutdown.String(), "  "+line) {
			t.Fatalf("expected shutdown rollout summary to contain %q", "  "+line)
		}
	}
}

func TestOwnershipRolloutPresenterLabels(t *testing.T) {
	lines := ownershipRolloutPresenterLines()
	if len(lines) == 0 {
		t.Fatal("expected presenter lines")
	}
	if lines[0].runningLabel != "Shadow-Owned Events" {
		t.Fatalf("expected first running label Shadow-Owned Events, got %s", lines[0].runningLabel)
	}
	if lines[0].shutdownLabel != "Shadow-owned events" {
		t.Fatalf("expected first shutdown label Shadow-owned events, got %s", lines[0].shutdownLabel)
	}
	last := lines[len(lines)-1]
	if last.runningLabel != "Guarded Cutover Overview Reason" {
		t.Fatalf("expected last running label Guarded Cutover Overview Reason, got %s", last.runningLabel)
	}
	if last.shutdownLabel != "Guarded cutover overview reason" {
		t.Fatalf("expected last shutdown label Guarded cutover overview reason, got %s", last.shutdownLabel)
	}
	if got := ownershipRolloutPresenterPrefix("running"); got != "" {
		t.Fatalf("expected running prefix empty, got %q", got)
	}
	if got := ownershipRolloutPresenterPrefix("shutdown"); got != "  " {
		t.Fatalf("expected shutdown prefix two spaces, got %q", got)
	}
}

func TestOwnershipRolloutLogDescriptors(t *testing.T) {
	descriptors := ownershipRolloutLogDescriptors()
	if len(descriptors) != 9 {
		t.Fatalf("expected 9 log descriptors, got %d", len(descriptors))
	}
	if descriptors[0].message != "Ownership rollout cutover candidate evaluated" {
		t.Fatalf("unexpected first log descriptor message %q", descriptors[0].message)
	}
	if descriptors[len(descriptors)-1].message != "Ownership rollout guarded cutover overview evaluated" {
		t.Fatalf("unexpected last log descriptor message %q", descriptors[len(descriptors)-1].message)
	}

	snapshot := ownershipRolloutSummarySnapshot{
		CutoverCandidate: ownershipCutoverCandidate{Eligible: true, Reason: "candidate-ready"},
		Policy:           ownershipRolloutPolicy{Mode: "manual-gate"},
		Progression:      ownershipEffectiveProgression{State: "ready-for-cutover"},
		CutoverDryRun:    ownershipCutoverDryRun{Action: "would-allow"},
		ManualApprovalCheckpoint: ownershipManualApprovalCheckpoint{
			State:  "awaiting-approval",
			Reason: "checkpoint-active",
		},
		OperatorHandoff: ownershipOperatorHandoff{State: "operator-review", Reason: "handoff-active"},
		ApprovalWorkItem: ownershipApprovalWorkItem{
			Status:       "open",
			Owner:        "platform-team/manual-approver",
			ReviewFields: "field-a,field-b",
			Reason:       "needs-review",
		},
		ApprovalChecklist: ownershipApprovalChecklist{State: "ready", Reason: "checklist-ready"},
		GuardedCutoverHook: ownershipGuardedCutoverHook{
			Action: "noop-allow",
			Reason: "hook-allow",
		},
		GuardedCutoverHookPolicy: ownershipGuardedCutoverHookPolicy{
			Mode:   "enforce-ready",
			Action: "enforce-would-allow",
		},
		GuardedCutoverWouldEnforce: ownershipGuardedCutoverWouldEnforce{
			Action: "would-allow",
			Reason: "future-allow",
		},
		GuardedCutoverEnforceHint: ownershipGuardedCutoverEnforceHint{
			State:  "safe-to-observe",
			Reason: "hint-safe",
		},
		GuardedCutoverOverview: ownershipGuardedCutoverOverview{
			State:  "observe",
			Reason: "overview-observe",
		},
	}

	logger := core.NewTestLoggerWithCapture()
	logOwnershipRolloutSummary(logger, "startup", snapshot)
	messages := logger.GetMessages()
	if len(messages) != len(descriptors) {
		t.Fatalf("expected %d captured log messages, got %d", len(descriptors), len(messages))
	}
	if messages[0] != descriptors[0].message {
		t.Fatalf("expected first captured message %q, got %q", descriptors[0].message, messages[0])
	}
	if messages[len(messages)-1] != descriptors[len(descriptors)-1].message {
		t.Fatalf("expected last captured message %q, got %q", descriptors[len(descriptors)-1].message, messages[len(messages)-1])
	}
}

func TestOwnershipRolloutValueAccessors(t *testing.T) {
	snapshot := ownershipRolloutSummarySnapshot{
		Summary: ownershipSummary{ShadowOwnedEvents: 4, LegacyOwnedEvents: 2, Chains: 1},
		Mode:    "shadow",
		CutoverCandidate: ownershipCutoverCandidate{
			Eligible: true,
			Reason:   "candidate-ready",
		},
		Progression: ownershipEffectiveProgression{
			State:  "ready-for-cutover",
			Reason: "progression-ready",
		},
		GuardedCutoverOverview: ownershipGuardedCutoverOverview{
			State:  "observe",
			Reason: "overview-observe",
		},
	}

	accessors := map[string]ownershipRolloutValueAccessor{
		"shadow_owned":  ownershipRolloutShadowOwnedEventsValue,
		"legacy_owned":  ownershipRolloutLegacyOwnedEventsValue,
		"chains":        ownershipRolloutOwnershipChainsValue,
		"mode":          ownershipRolloutModeValue,
		"candidate":     ownershipRolloutCutoverCandidateEligibleValue,
		"progression":   ownershipRolloutProgressionStateValue,
		"progression_r": ownershipRolloutProgressionReasonValue,
		"overview":      ownershipRolloutGuardedCutoverOverviewStateValue,
		"overview_r":    ownershipRolloutGuardedCutoverOverviewReasonValue,
	}
	want := map[string]string{
		"shadow_owned":  "4",
		"legacy_owned":  "2",
		"chains":        "1",
		"mode":          "shadow",
		"candidate":     "true",
		"progression":   "ready-for-cutover",
		"progression_r": "progression-ready",
		"overview":      "observe",
		"overview_r":    "overview-observe",
	}

	for name, accessor := range accessors {
		if got := accessor(snapshot); got != want[name] {
			t.Fatalf("%s expected %q, got %q", name, want[name], got)
		}
	}
}

func TestOwnershipRolloutPresenterSections(t *testing.T) {
	ownershipLines := ownershipRolloutOwnershipPresenterLines()
	approvalLines := ownershipRolloutApprovalPresenterLines()
	guardedLines := ownershipRolloutGuardedCutoverPresenterLines()
	if len(ownershipLines) != 10 {
		t.Fatalf("expected 10 ownership presenter lines, got %d", len(ownershipLines))
	}
	if len(approvalLines) != 10 {
		t.Fatalf("expected 10 approval presenter lines, got %d", len(approvalLines))
	}
	if len(guardedLines) != 11 {
		t.Fatalf("expected 11 guarded presenter lines, got %d", len(guardedLines))
	}
	if ownershipLines[0].runningLabel != "Shadow-Owned Events" {
		t.Fatalf("unexpected ownership first label %q", ownershipLines[0].runningLabel)
	}
	if approvalLines[0].runningLabel != "Manual Approval Checkpoint" {
		t.Fatalf("unexpected approval first label %q", approvalLines[0].runningLabel)
	}
	if guardedLines[0].runningLabel != "Guarded Cutover Hook" {
		t.Fatalf("unexpected guarded first label %q", guardedLines[0].runningLabel)
	}

	ownershipLogs := ownershipRolloutOwnershipLogDescriptors()
	approvalLogs := ownershipRolloutApprovalLogDescriptors()
	guardedLogs := ownershipRolloutGuardedCutoverLogDescriptors()
	if len(ownershipLogs) != 1 {
		t.Fatalf("expected 1 ownership log descriptor, got %d", len(ownershipLogs))
	}
	if len(approvalLogs) != 4 {
		t.Fatalf("expected 4 approval log descriptors, got %d", len(approvalLogs))
	}
	if len(guardedLogs) != 4 {
		t.Fatalf("expected 4 guarded log descriptors, got %d", len(guardedLogs))
	}

	presenterSections := buildOwnershipRolloutPresenterSections()
	if len(presenterSections.Ownership)+len(presenterSections.Approval)+len(presenterSections.Guarded) != len(ownershipRolloutPresenterLines()) {
		t.Fatal("expected presenter section assembler to match flattened presenter lines")
	}

	logSections := buildOwnershipRolloutLogSections()
	if len(logSections.Ownership)+len(logSections.Approval)+len(logSections.Guarded) != len(ownershipRolloutLogDescriptors()) {
		t.Fatal("expected log section assembler to match flattened log descriptors")
	}

	accessors := buildOwnershipRolloutPresenterAccessors()
	if accessors.Ownership.Mode == nil || accessors.Approval.ApprovalChecklistReason == nil || accessors.Guarded.GuardedCutoverOverviewReason == nil {
		t.Fatal("expected presenter accessors to be assembled for all sections")
	}
}

func TestNormalizeOwnershipGuardedCutoverHookPolicyMode(t *testing.T) {
	cases := []string{"", "noop-only", "noop_only", "noop", "weird"}
	for _, raw := range cases {
		if got := normalizeOwnershipGuardedCutoverHookPolicyMode(raw); got != "noop-only" {
			t.Fatalf("raw=%q expected noop-only, got %s", raw, got)
		}
	}

	manualCases := []string{"enforce-ready", "enforce_ready", "enforce"}
	for _, raw := range manualCases {
		if got := normalizeOwnershipGuardedCutoverHookPolicyMode(raw); got != "enforce-ready" {
			t.Fatalf("raw=%q expected enforce-ready, got %s", raw, got)
		}
	}
}

func TestResolveOwnershipGuardedCutoverHookPolicyFromEnv(t *testing.T) {
	hook := ownershipGuardedCutoverHook{
		Action: "noop-allow",
		Reason: "guarded cutover hook would allow progression, but remains non-blocking in dry-run mode",
	}

	policy := resolveOwnershipGuardedCutoverHookPolicyFromEnv(hook)
	if policy.Mode != "noop-only" {
		t.Fatalf("expected default mode noop-only, got %s", policy.Mode)
	}
	if policy.Action != "noop-report" {
		t.Fatalf("expected default action noop-report, got %s", policy.Action)
	}

	t.Setenv("CHAINPULSE_OWNERSHIP_GUARDED_CUTOVER_HOOK_POLICY_MODE", "enforce")
	policy = resolveOwnershipGuardedCutoverHookPolicyFromEnv(hook)
	if policy.Mode != "enforce-ready" {
		t.Fatalf("expected mode enforce-ready, got %s", policy.Mode)
	}
	if policy.Action != "enforce-would-allow" {
		t.Fatalf("expected action enforce-would-allow, got %s", policy.Action)
	}
}

func TestClassifyOwnershipGuardedCutoverWouldEnforce(t *testing.T) {
	cases := []struct {
		name   string
		hook   ownershipGuardedCutoverHook
		policy ownershipGuardedCutoverHookPolicy
		action string
		reason string
	}{
		{
			name:   "allow",
			hook:   ownershipGuardedCutoverHook{Action: "noop-allow"},
			policy: ownershipGuardedCutoverHookPolicy{Mode: "enforce-ready", Action: "enforce-would-allow"},
			action: "would-allow",
			reason: "future enforcement posture would allow this instance to proceed",
		},
		{
			name:   "hold",
			hook:   ownershipGuardedCutoverHook{Action: "noop-hold"},
			policy: ownershipGuardedCutoverHookPolicy{Mode: "noop-only", Action: "noop-report"},
			action: "would-hold",
			reason: "future enforcement posture would still hold this instance",
		},
		{
			name:   "investigate",
			hook:   ownershipGuardedCutoverHook{Action: "noop-investigate"},
			policy: ownershipGuardedCutoverHookPolicy{Mode: "enforce-ready", Action: "enforce-would-investigate"},
			action: "would-investigate",
			reason: "future enforcement posture requires investigation before any guarded cutover decision",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipGuardedCutoverWouldEnforce(tc.hook, tc.policy)
			if got.Action != tc.action {
				t.Fatalf("expected action %s, got %s", tc.action, got.Action)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipGuardedCutoverEnforceHint(t *testing.T) {
	cases := []struct {
		name   string
		input  ownershipGuardedCutoverWouldEnforce
		state  string
		reason string
	}{
		{
			name:   "safe_to_observe",
			input:  ownershipGuardedCutoverWouldEnforce{Action: "would-allow"},
			state:  "safe-to-observe",
			reason: "future enforcement posture is healthy enough to keep observing toward eventual enforcement",
		},
		{
			name:   "hold_before_enforce",
			input:  ownershipGuardedCutoverWouldEnforce{Action: "would-hold"},
			state:  "hold-before-enforce",
			reason: "future enforcement posture should remain on hold before any enforce decision is considered",
		},
		{
			name:   "investigate_before_enforce",
			input:  ownershipGuardedCutoverWouldEnforce{Action: "would-investigate"},
			state:  "investigate-before-enforce",
			reason: "future enforcement posture needs investigation before any enforce decision is considered",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipGuardedCutoverEnforceHint(tc.input)
			if got.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, got.State)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipGuardedCutoverOverview(t *testing.T) {
	cases := []struct {
		name         string
		wouldEnforce ownershipGuardedCutoverWouldEnforce
		enforceHint  ownershipGuardedCutoverEnforceHint
		state        string
		reason       string
	}{
		{
			name:         "observe",
			wouldEnforce: ownershipGuardedCutoverWouldEnforce{Action: "would-allow"},
			enforceHint:  ownershipGuardedCutoverEnforceHint{State: "safe-to-observe"},
			state:        "observe",
			reason:       "guarded cutover posture is healthy enough to keep observing toward possible enforcement",
		},
		{
			name:         "hold",
			wouldEnforce: ownershipGuardedCutoverWouldEnforce{Action: "would-hold"},
			enforceHint:  ownershipGuardedCutoverEnforceHint{State: "hold-before-enforce"},
			state:        "hold",
			reason:       "guarded cutover posture should remain on hold before any future enforcement decision",
		},
		{
			name:         "investigate",
			wouldEnforce: ownershipGuardedCutoverWouldEnforce{Action: "would-investigate"},
			enforceHint:  ownershipGuardedCutoverEnforceHint{State: "investigate-before-enforce"},
			state:        "investigate",
			reason:       "guarded cutover posture requires investigation before any future enforcement decision",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipGuardedCutoverOverview(tc.wouldEnforce, tc.enforceHint)
			if got.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, got.State)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestResolveOwnershipRolloutPolicyFromEnv(t *testing.T) {
	t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_POLICY_MODE", "report_only")
	policy := resolveOwnershipRolloutPolicyFromEnv(ownershipRolloutAdvisory{Decision: "allow"})
	if policy.Mode != "report-only" {
		t.Fatalf("expected mode report-only, got %s", policy.Mode)
	}
	if policy.Action != "report-allow" {
		t.Fatalf("expected action report-allow, got %s", policy.Action)
	}
	if policy.Reason != "report-only mode does not block ownership rollout" {
		t.Fatalf("unexpected reason %s", policy.Reason)
	}
	if policy.Acknowledged {
		t.Fatalf("expected acknowledged false")
	}
	if policy.AckState != "pending" {
		t.Fatalf("expected ack state pending, got %s", policy.AckState)
	}
}

func TestResolveOwnershipRolloutPolicyFromEnvManualGate(t *testing.T) {
	cases := []struct {
		decision string
		action   string
	}{
		{decision: "allow", action: "manual-review-allow"},
		{decision: "hold", action: "manual-review-hold"},
		{decision: "unknown", action: "manual-review-unknown"},
	}

	for _, tc := range cases {
		t.Run(tc.decision, func(t *testing.T) {
			t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_POLICY_MODE", "manual_gate")
			policy := resolveOwnershipRolloutPolicyFromEnv(ownershipRolloutAdvisory{Decision: tc.decision})
			if policy.Mode != "manual-gate" {
				t.Fatalf("expected mode manual-gate, got %s", policy.Mode)
			}
			if policy.Action != tc.action {
				t.Fatalf("expected action %s, got %s", tc.action, policy.Action)
			}
			if policy.Reason != "manual-gate mode requires operator review before ownership progression" {
				t.Fatalf("unexpected reason %s", policy.Reason)
			}
			if policy.Acknowledged {
				t.Fatalf("expected acknowledged false")
			}
			if policy.AckState != "pending" {
				t.Fatalf("expected ack state pending, got %s", policy.AckState)
			}
		})
	}
}

func TestResolveOwnershipRolloutPolicyFromEnvManualGateAcknowledged(t *testing.T) {
	t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_POLICY_MODE", "manual-gate")
	t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_ACKNOWLEDGED", "true")
	policy := resolveOwnershipRolloutPolicyFromEnv(ownershipRolloutAdvisory{Decision: "allow"})

	if policy.Mode != "manual-gate" {
		t.Fatalf("expected mode manual-gate, got %s", policy.Mode)
	}
	if policy.Action != "manual-acknowledged-allow" {
		t.Fatalf("expected action manual-acknowledged-allow, got %s", policy.Action)
	}
	if policy.Reason != "manual-gate mode has recorded operator acknowledgment for ownership progression" {
		t.Fatalf("unexpected reason %s", policy.Reason)
	}
	if !policy.Acknowledged {
		t.Fatalf("expected acknowledged true")
	}
	if policy.AckState != "acknowledged" {
		t.Fatalf("expected ack state acknowledged, got %s", policy.AckState)
	}
}

func TestResolveOwnershipRolloutAcknowledgedFromEnv(t *testing.T) {
	trueCases := []string{"1", "true", "yes", "y", "on"}
	for _, raw := range trueCases {
		t.Run("true_"+raw, func(t *testing.T) {
			t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_ACKNOWLEDGED", raw)
			if !resolveOwnershipRolloutAcknowledgedFromEnv() {
				t.Fatalf("expected acknowledged true for %q", raw)
			}
		})
	}

	falseCases := []string{"", "0", "false", "no", "weird"}
	for _, raw := range falseCases {
		t.Run("false_"+raw, func(t *testing.T) {
			t.Setenv("CHAINPULSE_OWNERSHIP_ROLLOUT_ACKNOWLEDGED", raw)
			if resolveOwnershipRolloutAcknowledgedFromEnv() {
				t.Fatalf("expected acknowledged false for %q", raw)
			}
		})
	}
}

func TestClassifyOwnershipEffectiveProgression(t *testing.T) {
	cases := []struct {
		name   string
		adv    ownershipRolloutAdvisory
		policy ownershipRolloutPolicy
		state  string
		reason string
	}{
		{
			name:   "observe_report_only",
			adv:    ownershipRolloutAdvisory{Decision: "hold"},
			policy: ownershipRolloutPolicy{Mode: "report-only"},
			state:  "observe",
			reason: "report-only mode observes rollout state without changing execution",
		},
		{
			name:   "review_required",
			adv:    ownershipRolloutAdvisory{Decision: "hold"},
			policy: ownershipRolloutPolicy{Mode: "manual-gate", Acknowledged: false},
			state:  "review-required",
			reason: "manual gate requires operator review before progression",
		},
		{
			name:   "acknowledged",
			adv:    ownershipRolloutAdvisory{Decision: "hold"},
			policy: ownershipRolloutPolicy{Mode: "manual-gate", Acknowledged: true},
			state:  "acknowledged",
			reason: "manual gate is acknowledged but advisory still holds progression",
		},
		{
			name:   "ready_for_cutover",
			adv:    ownershipRolloutAdvisory{Decision: "allow"},
			policy: ownershipRolloutPolicy{Mode: "manual-gate", Acknowledged: true},
			state:  "ready-for-cutover",
			reason: "manual gate is acknowledged and advisory allows progression",
		},
		{
			name:   "unknown",
			adv:    ownershipRolloutAdvisory{Decision: "unknown"},
			policy: ownershipRolloutPolicy{Mode: "manual-gate"},
			state:  "unknown",
			reason: "ownership rollout decision is unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipEffectiveProgression(tc.adv, tc.policy)
			if got.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, got.State)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipCutoverDryRun(t *testing.T) {
	cases := []struct {
		name        string
		progression ownershipEffectiveProgression
		action      string
		reason      string
	}{
		{
			name:        "allow",
			progression: ownershipEffectiveProgression{State: "ready-for-cutover"},
			action:      "would-allow",
			reason:      "effective progression state indicates the service would allow cutover in dry-run mode",
		},
		{
			name:        "hold",
			progression: ownershipEffectiveProgression{State: "review-required"},
			action:      "would-hold",
			reason:      "effective progression state does not yet satisfy cutover conditions in dry-run mode",
		},
		{
			name:        "unknown",
			progression: ownershipEffectiveProgression{State: "unknown"},
			action:      "would-unknown",
			reason:      "effective progression state is unknown so dry-run cutover decision is also unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipCutoverDryRun(tc.progression)
			if got.Action != tc.action {
				t.Fatalf("expected action %s, got %s", tc.action, got.Action)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipCutoverCandidate(t *testing.T) {
	cases := []struct {
		name        string
		policy      ownershipRolloutPolicy
		progression ownershipEffectiveProgression
		cutover     ownershipCutoverDryRun
		eligible    bool
		reason      string
	}{
		{
			name:        "eligible",
			policy:      ownershipRolloutPolicy{Mode: "manual-gate", Acknowledged: true},
			progression: ownershipEffectiveProgression{State: "ready-for-cutover"},
			cutover:     ownershipCutoverDryRun{Action: "would-allow"},
			eligible:    true,
			reason:      "manual gate is acknowledged and dry-run cutover would allow progression",
		},
		{
			name:        "requires_manual_gate",
			policy:      ownershipRolloutPolicy{Mode: "report-only", Acknowledged: true},
			progression: ownershipEffectiveProgression{State: "ready-for-cutover"},
			cutover:     ownershipCutoverDryRun{Action: "would-allow"},
			eligible:    false,
			reason:      "cutover candidate requires manual-gate policy mode",
		},
		{
			name:        "requires_ack",
			policy:      ownershipRolloutPolicy{Mode: "manual-gate", Acknowledged: false},
			progression: ownershipEffectiveProgression{State: "ready-for-cutover"},
			cutover:     ownershipCutoverDryRun{Action: "would-allow"},
			eligible:    false,
			reason:      "cutover candidate requires recorded operator acknowledgment",
		},
		{
			name:        "requires_progression",
			policy:      ownershipRolloutPolicy{Mode: "manual-gate", Acknowledged: true},
			progression: ownershipEffectiveProgression{State: "acknowledged"},
			cutover:     ownershipCutoverDryRun{Action: "would-hold"},
			eligible:    false,
			reason:      "cutover candidate requires ready-for-cutover effective progression state",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipCutoverCandidate(tc.policy, tc.progression, tc.cutover)
			if got.Eligible != tc.eligible {
				t.Fatalf("expected eligible %t, got %t", tc.eligible, got.Eligible)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipManualApprovalCheckpoint(t *testing.T) {
	cases := []struct {
		name        string
		progression ownershipEffectiveProgression
		candidate   ownershipCutoverCandidate
		state       string
		reason      string
	}{
		{
			name:        "awaiting_approval",
			progression: ownershipEffectiveProgression{State: "ready-for-cutover"},
			candidate:   ownershipCutoverCandidate{Eligible: true},
			state:       "awaiting-approval",
			reason:      "instance is a cutover candidate and is awaiting manual approval checkpoint",
		},
		{
			name:        "inactive",
			progression: ownershipEffectiveProgression{State: "observe"},
			candidate:   ownershipCutoverCandidate{Eligible: false},
			state:       "inactive",
			reason:      "instance has not yet reached cutover candidate posture",
		},
		{
			name:        "unknown",
			progression: ownershipEffectiveProgression{State: "unknown"},
			candidate:   ownershipCutoverCandidate{Eligible: false},
			state:       "unknown",
			reason:      "manual approval checkpoint is unknown because rollout progression is unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipManualApprovalCheckpoint(tc.progression, tc.candidate)
			if got.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, got.State)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipOperatorHandoff(t *testing.T) {
	cases := []struct {
		name       string
		checkpoint ownershipManualApprovalCheckpoint
		state      string
		reason     string
	}{
		{
			name:       "none",
			checkpoint: ownershipManualApprovalCheckpoint{State: "inactive"},
			state:      "none",
			reason:     "operator handoff is not required for the current rollout posture",
		},
		{
			name:       "operator_review",
			checkpoint: ownershipManualApprovalCheckpoint{State: "awaiting-approval"},
			state:      "operator-review",
			reason:     "manual approval checkpoint is active and requires operator review",
		},
		{
			name:       "investigate",
			checkpoint: ownershipManualApprovalCheckpoint{State: "unknown"},
			state:      "investigate",
			reason:     "manual approval checkpoint is unknown and rollout posture requires investigation",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipOperatorHandoff(tc.checkpoint)
			if got.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, got.State)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipApprovalWorkItem(t *testing.T) {
	cases := []struct {
		name   string
		input  ownershipOperatorHandoff
		status string
		owner  string
		reason string
	}{
		{
			name:   "none",
			input:  ownershipOperatorHandoff{State: "none"},
			status: "none",
			owner:  "none",
			reason: "no approval work item is required for the current rollout posture",
		},
		{
			name:   "open",
			input:  ownershipOperatorHandoff{State: "operator-review"},
			status: "open",
			owner:  "platform-team/manual-approver",
			reason: "operator handoff requires approval review before any future cutover action",
		},
		{
			name:   "investigate",
			input:  ownershipOperatorHandoff{State: "investigate"},
			status: "investigate",
			owner:  "platform-team/runtime-owners",
			reason: "operator handoff requires investigation before approval can proceed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipApprovalWorkItem(tc.input)
			if got.Status != tc.status {
				t.Fatalf("expected status %s, got %s", tc.status, got.Status)
			}
			if got.Owner != tc.owner {
				t.Fatalf("expected owner %s, got %s", tc.owner, got.Owner)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
			if got.ReviewFields == "" {
				t.Fatalf("expected non-empty review fields")
			}
		})
	}
}

func TestClassifyOwnershipApprovalChecklist(t *testing.T) {
	reviewFields := "rollout_effective_state,rollout_cutover_candidate,rollout_manual_approval_checkpoint_state,rollout_operator_handoff_state"
	cases := []struct {
		name       string
		candidate  ownershipCutoverCandidate
		checkpoint ownershipManualApprovalCheckpoint
		handoff    ownershipOperatorHandoff
		workItem   ownershipApprovalWorkItem
		state      string
		reason     string
	}{
		{
			name:       "ready",
			candidate:  ownershipCutoverCandidate{Eligible: true},
			checkpoint: ownershipManualApprovalCheckpoint{State: "awaiting-approval"},
			handoff:    ownershipOperatorHandoff{State: "operator-review"},
			workItem: ownershipApprovalWorkItem{
				Status:       "open",
				ReviewFields: reviewFields,
			},
			state:  "ready",
			reason: "approval checklist prerequisites are present for manual review",
		},
		{
			name:       "incomplete",
			candidate:  ownershipCutoverCandidate{Eligible: false},
			checkpoint: ownershipManualApprovalCheckpoint{State: "inactive"},
			handoff:    ownershipOperatorHandoff{State: "none"},
			workItem: ownershipApprovalWorkItem{
				Status:       "none",
				ReviewFields: reviewFields,
			},
			state:  "incomplete",
			reason: "approval checklist prerequisites are not fully satisfied yet",
		},
		{
			name:       "investigate",
			candidate:  ownershipCutoverCandidate{Eligible: false},
			checkpoint: ownershipManualApprovalCheckpoint{State: "unknown"},
			handoff:    ownershipOperatorHandoff{State: "investigate"},
			workItem: ownershipApprovalWorkItem{
				Status:       "investigate",
				ReviewFields: reviewFields,
			},
			state:  "investigate",
			reason: "approval checklist requires investigation before review can proceed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipApprovalChecklist(tc.candidate, tc.checkpoint, tc.handoff, tc.workItem)
			if got.State != tc.state {
				t.Fatalf("expected state %s, got %s", tc.state, got.State)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}

func TestClassifyOwnershipGuardedCutoverHook(t *testing.T) {
	cases := []struct {
		name      string
		cutover   ownershipCutoverDryRun
		candidate ownershipCutoverCandidate
		checklist ownershipApprovalChecklist
		action    string
		reason    string
	}{
		{
			name:      "allow",
			cutover:   ownershipCutoverDryRun{Action: "would-allow"},
			candidate: ownershipCutoverCandidate{Eligible: true},
			checklist: ownershipApprovalChecklist{State: "ready"},
			action:    "noop-allow",
			reason:    "guarded cutover hook would allow progression, but remains non-blocking in dry-run mode",
		},
		{
			name:      "hold",
			cutover:   ownershipCutoverDryRun{Action: "would-hold"},
			candidate: ownershipCutoverCandidate{Eligible: false},
			checklist: ownershipApprovalChecklist{State: "incomplete"},
			action:    "noop-hold",
			reason:    "guarded cutover hook would still hold because approval prerequisites are not fully satisfied",
		},
		{
			name:      "investigate",
			cutover:   ownershipCutoverDryRun{Action: "would-unknown"},
			candidate: ownershipCutoverCandidate{Eligible: false},
			checklist: ownershipApprovalChecklist{State: "investigate"},
			action:    "noop-investigate",
			reason:    "guarded cutover hook requires investigation before any future control action",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyOwnershipGuardedCutoverHook(tc.cutover, tc.candidate, tc.checklist)
			if got.Action != tc.action {
				t.Fatalf("expected action %s, got %s", tc.action, got.Action)
			}
			if got.Reason != tc.reason {
				t.Fatalf("expected reason %q, got %q", tc.reason, got.Reason)
			}
		})
	}
}
