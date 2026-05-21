package rollout

import (
	"fmt"
	"strings"
)

// ValidateMicroserviceRolloutMetadataParity validates the shared metadata
// boundary expected from microservice rollout producers.
func ValidateMicroserviceRolloutMetadataParity(details *RolloutReportDetails, service, reportID, reportSource string) error {
	if details == nil {
		return fmt.Errorf("rollout report details are required")
	}
	if details.SchemaFamily != OwnershipRolloutSchemaFamily {
		return fmt.Errorf("expected schema_family %q, got %q", OwnershipRolloutSchemaFamily, details.SchemaFamily)
	}
	if details.ReportVersion != OwnershipRolloutReportVersion {
		return fmt.Errorf("expected report_version %q, got %q", OwnershipRolloutReportVersion, details.ReportVersion)
	}
	if details.ReportScope != OwnershipRolloutReportScope {
		return fmt.Errorf("expected report_scope %q, got %q", OwnershipRolloutReportScope, details.ReportScope)
	}
	if details.ReportMode != OwnershipRolloutReportMode {
		return fmt.Errorf("expected report_mode %q, got %q", OwnershipRolloutReportMode, details.ReportMode)
	}
	if details.DeploymentMode != "microservice" {
		return fmt.Errorf("expected deployment_mode %q, got %q", "microservice", details.DeploymentMode)
	}
	if details.Service != service {
		return fmt.Errorf("expected service %q, got %q", service, details.Service)
	}
	if details.ReportID != reportID {
		return fmt.Errorf("expected report_id %q, got %q", reportID, details.ReportID)
	}
	if details.ReportSource != reportSource {
		return fmt.Errorf("expected report_source %q, got %q", reportSource, details.ReportSource)
	}
	return nil
}

// ValidateMicroserviceRuntimeDerivedRolloutParity validates the shared runtime
// body posture boundary expected from runtime-derived microservice rollout
// producers. Service-specific advisory status and reason text intentionally
// remain outside this check.
func ValidateMicroserviceRuntimeDerivedRolloutParity(details *RolloutReportDetails) error {
	if details == nil {
		return fmt.Errorf("rollout report details are required")
	}
	if details.Summary.ShadowOwnedEvents != 0 {
		return fmt.Errorf("expected shadow_owned_events 0, got %d", details.Summary.ShadowOwnedEvents)
	}
	if details.Summary.LegacyOwnedEvents != 0 {
		return fmt.Errorf("expected legacy_owned_events 0, got %d", details.Summary.LegacyOwnedEvents)
	}
	if details.Summary.OwnershipChains != 0 {
		return fmt.Errorf("expected ownership_chains 0, got %d", details.Summary.OwnershipChains)
	}
	if details.Advisory.Decision != "hold" {
		return fmt.Errorf("expected advisory decision %q, got %q", "hold", details.Advisory.Decision)
	}
	if details.Policy.Mode != "report-only" {
		return fmt.Errorf("expected policy mode %q, got %q", "report-only", details.Policy.Mode)
	}
	if details.Policy.Action != "report-hold" {
		return fmt.Errorf("expected policy action %q, got %q", "report-hold", details.Policy.Action)
	}
	if details.Policy.Acknowledged {
		return fmt.Errorf("expected policy acknowledged false, got true")
	}
	if details.Policy.AckState != "pending" {
		return fmt.Errorf("expected ack_state %q, got %q", "pending", details.Policy.AckState)
	}
	if details.Progression.State != "observe" {
		return fmt.Errorf("expected progression state %q, got %q", "observe", details.Progression.State)
	}
	if details.CutoverDryRun.Action != "would-hold" {
		return fmt.Errorf("expected cutover_dry_run action %q, got %q", "would-hold", details.CutoverDryRun.Action)
	}
	if details.CutoverCandidate.Eligible {
		return fmt.Errorf("expected cutover candidate false, got true")
	}
	if details.Approval.ManualApprovalCheckpoint.State != "inactive" {
		return fmt.Errorf("expected manual approval checkpoint %q, got %q", "inactive", details.Approval.ManualApprovalCheckpoint.State)
	}
	if details.Approval.OperatorHandoff.State != "none" {
		return fmt.Errorf("expected operator handoff %q, got %q", "none", details.Approval.OperatorHandoff.State)
	}
	if details.Approval.Checklist.State != "incomplete" {
		return fmt.Errorf("expected checklist state %q, got %q", "incomplete", details.Approval.Checklist.State)
	}
	if details.Approval.WorkItem.Status != "none" {
		return fmt.Errorf("expected work item status %q, got %q", "none", details.Approval.WorkItem.Status)
	}
	if details.GuardedCutover.Hook.Action != "noop-hold" {
		return fmt.Errorf("expected guarded hook action %q, got %q", "noop-hold", details.GuardedCutover.Hook.Action)
	}
	if details.GuardedCutover.HookPolicy.Mode != "noop-only" {
		return fmt.Errorf("expected guarded hook policy mode %q, got %q", "noop-only", details.GuardedCutover.HookPolicy.Mode)
	}
	if details.GuardedCutover.HookPolicy.Action != "noop-hold" {
		return fmt.Errorf("expected guarded hook policy action %q, got %q", "noop-hold", details.GuardedCutover.HookPolicy.Action)
	}
	if details.GuardedCutover.WouldEnforce.Action != "would-hold" {
		return fmt.Errorf("expected guarded would_enforce action %q, got %q", "would-hold", details.GuardedCutover.WouldEnforce.Action)
	}
	if details.GuardedCutover.EnforceHint.State != "hold-before-enforce" {
		return fmt.Errorf("expected enforce hint state %q, got %q", "hold-before-enforce", details.GuardedCutover.EnforceHint.State)
	}
	if details.GuardedCutover.Overview.State != "hold" {
		return fmt.Errorf("expected guarded overview state %q, got %q", "hold", details.GuardedCutover.Overview.State)
	}
	return nil
}

// ValidateMicroserviceOwnershipParityMarker validates the shared ownership
// parity marker boundary now expected from route-oriented microservice rollout
// producers that explicitly expose the current ownership/runtime gap.
func ValidateMicroserviceOwnershipParityMarker(details *RolloutReportDetails) error {
	if details == nil {
		return fmt.Errorf("rollout report details are required")
	}
	if !strings.Contains(details.Advisory.Reason, "ownership_parity_hint: ") {
		return fmt.Errorf("expected advisory reason to include ownership_parity_hint")
	}
	if !strings.Contains(details.Advisory.Reason, "ownership runtime parity with monolith") {
		return fmt.Errorf("expected advisory reason to mention ownership runtime parity with monolith")
	}
	if !strings.Contains(details.Approval.WorkItem.ReviewFields, "ownership_runtime_parity") {
		return fmt.Errorf("expected work item review fields to include %q, got %q", "ownership_runtime_parity", details.Approval.WorkItem.ReviewFields)
	}
	if !strings.Contains(details.Approval.WorkItem.Reason, "ownership runtime parity with monolith") {
		return fmt.Errorf("expected work item reason to mention ownership runtime parity with monolith")
	}
	return nil
}

// ValidateRouteMonolithOwnershipParityReason validates the shared
// route-oriented monolith-backed parity reason surface now expected from
// route-facing microservice rollout producers.
func ValidateRouteMonolithOwnershipParityReason(details *RolloutReportDetails, posture, hint, targetDecision, actionGuidance string) error {
	if details == nil {
		return fmt.Errorf("rollout report details are required")
	}
	if posture == "" {
		return fmt.Errorf("monolith parity posture is required")
	}
	if hint == "" {
		return fmt.Errorf("monolith parity hint is required")
	}
	if !strings.Contains(details.Advisory.Reason, "monolith_parity_posture: "+posture) {
		return fmt.Errorf("expected advisory reason to include monolith_parity_posture %q", posture)
	}
	if !strings.Contains(details.Advisory.Reason, "monolith_parity_hint: "+hint) {
		return fmt.Errorf("expected advisory reason to include monolith_parity_hint %q", hint)
	}
	if targetDecision == "" {
		return fmt.Errorf("monolith parity target decision is required")
	}
	if !strings.Contains(details.Advisory.Reason, "monolith_parity_target_decision: "+targetDecision) {
		return fmt.Errorf("expected advisory reason to include monolith_parity_target_decision %q", targetDecision)
	}
	if actionGuidance == "" {
		return fmt.Errorf("monolith parity action guidance is required")
	}
	if !strings.Contains(details.Advisory.Reason, "monolith_parity_action_guidance: "+actionGuidance) {
		return fmt.Errorf("expected advisory reason to include monolith_parity_action_guidance %q", actionGuidance)
	}
	return nil
}

func ValidateRouteMonolithOwnershipParityRecommendationBundle(details *RolloutReportDetails, bundle MonolithOwnershipParityRecommendationBundle) error {
	if bundle.Posture == "" {
		return fmt.Errorf("monolith parity recommendation posture is required")
	}
	if bundle.Hint == "" {
		return fmt.Errorf("monolith parity recommendation hint is required")
	}
	if bundle.TargetDecision == "" {
		return fmt.Errorf("monolith parity recommendation target decision is required")
	}
	if bundle.ActionGuidance == "" {
		return fmt.Errorf("monolith parity recommendation action guidance is required")
	}
	return ValidateRouteMonolithOwnershipParityReason(
		details,
		bundle.Posture,
		bundle.Hint,
		bundle.TargetDecision,
		bundle.ActionGuidance,
	)
}
