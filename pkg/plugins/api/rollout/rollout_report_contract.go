package rollout

import "context"

const (
	OwnershipRolloutSchemaFamily  = "ownership-rollout-report"
	OwnershipRolloutReportVersion = "v1"
	OwnershipRolloutReportScope   = "ownership-rollout"
	OwnershipRolloutReportMode    = "runtime"
)

// RolloutReportMetadata captures the stable identity envelope for a rollout report.
type RolloutReportMetadata struct {
	ReportID       string
	SchemaFamily   string
	ReportVersion  string
	Service        string
	ReportScope    string
	ReportSource   string
	ReportMode     string
	DeploymentMode string
	GeneratedAt    int64
}

// RolloutReportResponse represents a rollout status report response.
type RolloutReportResponse struct {
	Status    string                `json:"status"`
	Timestamp int64                 `json:"timestamp"`
	Available bool                  `json:"available"`
	Message   string                `json:"message"`
	Details   *RolloutReportDetails `json:"details,omitempty"`
}

// RolloutReportDetails represents the typed contract returned by /health/rollout.
type RolloutReportDetails struct {
	ReportID         string                   `json:"report_id"`
	SchemaFamily     string                   `json:"schema_family"`
	ReportVersion    string                   `json:"report_version"`
	Service          string                   `json:"service"`
	ReportScope      string                   `json:"report_scope"`
	ReportSource     string                   `json:"report_source"`
	ReportMode       string                   `json:"report_mode"`
	DeploymentMode   string                   `json:"deployment_mode"`
	GeneratedAt      int64                    `json:"generated_at"`
	Summary          RolloutReportSummary     `json:"summary"`
	Mode             string                   `json:"mode"`
	Advisory         RolloutReportAdvisory    `json:"advisory"`
	Policy           RolloutReportPolicy      `json:"policy"`
	Progression      RolloutReportStateReason `json:"progression"`
	CutoverDryRun    RolloutReportAction      `json:"cutover_dry_run"`
	CutoverCandidate RolloutReportCandidate   `json:"cutover_candidate"`
	Approval         RolloutReportApproval    `json:"approval"`
	GuardedCutover   RolloutReportGuarded     `json:"guarded_cutover"`
}

type RolloutReportSections struct {
	Surface        RolloutReportSurfaceSection `json:"surface"`
	Approval       RolloutReportApproval       `json:"approval"`
	GuardedCutover RolloutReportGuarded        `json:"guarded_cutover"`
}

type RolloutReportSectionsInput struct {
	Surface        RolloutReportSurfaceInput
	Approval       RolloutReportApprovalInput
	GuardedCutover RolloutReportGuardedInput
}

type RolloutReportSurfaceSection struct {
	Summary          RolloutReportSummary     `json:"summary"`
	Mode             string                   `json:"mode"`
	Advisory         RolloutReportAdvisory    `json:"advisory"`
	Policy           RolloutReportPolicy      `json:"policy"`
	Progression      RolloutReportStateReason `json:"progression"`
	CutoverDryRun    RolloutReportAction      `json:"cutover_dry_run"`
	CutoverCandidate RolloutReportCandidate   `json:"cutover_candidate"`
}

type RolloutReportSurfaceInput struct {
	Summary          RolloutReportSummary
	Mode             string
	Advisory         RolloutReportAdvisory
	Policy           RolloutReportPolicy
	Progression      RolloutReportStateReason
	CutoverDryRun    RolloutReportAction
	CutoverCandidate RolloutReportCandidate
}

type RolloutReportSurfaceCoreInput struct {
	Summary     RolloutReportSummary
	Mode        string
	Advisory    RolloutReportAdvisory
	Policy      RolloutReportPolicy
	Progression RolloutReportStateReason
}

type RolloutReportSurfaceCutoverInput struct {
	CutoverDryRun    RolloutReportAction
	CutoverCandidate RolloutReportCandidate
}

type RolloutReportSummary struct {
	ShadowOwnedEvents int64 `json:"shadow_owned_events"`
	LegacyOwnedEvents int64 `json:"legacy_owned_events"`
	OwnershipChains   int   `json:"ownership_chains"`
}

type RolloutReportAdvisory struct {
	Decision string `json:"decision"`
	Status   string `json:"status"`
	Ready    bool   `json:"ready"`
	Reason   string `json:"reason"`
}

type RolloutReportPolicy struct {
	Mode         string `json:"mode"`
	Action       string `json:"action"`
	Reason       string `json:"reason"`
	Acknowledged bool   `json:"acknowledged"`
	AckState     string `json:"ack_state"`
}

type RolloutReportStateReason struct {
	State  string `json:"state"`
	Reason string `json:"reason"`
}

type RolloutReportAction struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type RolloutReportCandidate struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}

type RolloutReportApproval struct {
	ManualApprovalCheckpoint RolloutReportStateReason  `json:"manual_approval_checkpoint"`
	OperatorHandoff          RolloutReportStateReason  `json:"operator_handoff"`
	WorkItem                 RolloutReportApprovalItem `json:"work_item"`
	Checklist                RolloutReportStateReason  `json:"checklist"`
}

type RolloutReportApprovalInput struct {
	ManualApprovalCheckpoint RolloutReportStateReason
	OperatorHandoff          RolloutReportStateReason
	WorkItem                 RolloutReportApprovalItem
	Checklist                RolloutReportStateReason
}

type RolloutReportApprovalFlowInput struct {
	ManualApprovalCheckpoint RolloutReportStateReason
	OperatorHandoff          RolloutReportStateReason
	Checklist                RolloutReportStateReason
}

type RolloutReportApprovalWorkItemInput struct {
	WorkItem RolloutReportApprovalItem
}

type RolloutReportApprovalItem struct {
	Status       string `json:"status"`
	Owner        string `json:"owner"`
	ReviewFields string `json:"review_fields"`
	Reason       string `json:"reason"`
}

type RolloutReportGuarded struct {
	Hook         RolloutReportAction      `json:"hook"`
	HookPolicy   RolloutReportModeAction  `json:"hook_policy"`
	WouldEnforce RolloutReportAction      `json:"would_enforce"`
	EnforceHint  RolloutReportStateReason `json:"enforce_hint"`
	Overview     RolloutReportStateReason `json:"overview"`
}

type RolloutReportGuardedInput struct {
	Hook         RolloutReportAction
	HookPolicy   RolloutReportModeAction
	WouldEnforce RolloutReportAction
	EnforceHint  RolloutReportStateReason
	Overview     RolloutReportStateReason
}

type RolloutReportGuardedHookInput struct {
	Hook       RolloutReportAction
	HookPolicy RolloutReportModeAction
}

type RolloutReportGuardedEnforcementInput struct {
	WouldEnforce RolloutReportAction
	EnforceHint  RolloutReportStateReason
	Overview     RolloutReportStateReason
}

type RolloutReportModeAction struct {
	Mode   string `json:"mode"`
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// RolloutReportProducer provides a typed rollout report payload for shared
// health/report surfaces.
type RolloutReportProducer interface {
	BuildRolloutReport(ctx context.Context) *RolloutReportDetails
}

// RolloutReportProducerFunc adapts a function to RolloutReportProducer.
type RolloutReportProducerFunc func(ctx context.Context) *RolloutReportDetails

func (f RolloutReportProducerFunc) BuildRolloutReport(ctx context.Context) *RolloutReportDetails {
	if f == nil {
		return nil
	}
	return f(ctx)
}

func NewRolloutReportDetailsFromMetadata(metadata RolloutReportMetadata) *RolloutReportDetails {
	return &RolloutReportDetails{
		ReportID:       metadata.ReportID,
		SchemaFamily:   metadata.SchemaFamily,
		ReportVersion:  metadata.ReportVersion,
		Service:        metadata.Service,
		ReportScope:    metadata.ReportScope,
		ReportSource:   metadata.ReportSource,
		ReportMode:     metadata.ReportMode,
		DeploymentMode: metadata.DeploymentMode,
		GeneratedAt:    metadata.GeneratedAt,
	}
}

func NewOwnershipRolloutReportMetadata(service, reportID, reportSource, deploymentMode string, generatedAt int64) RolloutReportMetadata {
	return RolloutReportMetadata{
		ReportID:       reportID,
		SchemaFamily:   OwnershipRolloutSchemaFamily,
		ReportVersion:  OwnershipRolloutReportVersion,
		Service:        service,
		ReportScope:    OwnershipRolloutReportScope,
		ReportSource:   reportSource,
		ReportMode:     OwnershipRolloutReportMode,
		DeploymentMode: deploymentMode,
		GeneratedAt:    generatedAt,
	}
}

func BuildRolloutReportSurfaceInput(core RolloutReportSurfaceCoreInput, cutover RolloutReportSurfaceCutoverInput) RolloutReportSurfaceInput {
	return RolloutReportSurfaceInput{
		Summary:          core.Summary,
		Mode:             core.Mode,
		Advisory:         core.Advisory,
		Policy:           core.Policy,
		Progression:      core.Progression,
		CutoverDryRun:    cutover.CutoverDryRun,
		CutoverCandidate: cutover.CutoverCandidate,
	}
}

func BuildRolloutReportSurfaceSection(input RolloutReportSurfaceInput) RolloutReportSurfaceSection {
	return RolloutReportSurfaceSection(input)
}

func ApplyRolloutReportSurfaceSection(details *RolloutReportDetails, section RolloutReportSurfaceSection) {
	if details == nil {
		return
	}

	details.Summary = section.Summary
	details.Mode = section.Mode
	details.Advisory = section.Advisory
	details.Policy = section.Policy
	details.Progression = section.Progression
	details.CutoverDryRun = section.CutoverDryRun
	details.CutoverCandidate = section.CutoverCandidate
}

func BuildRolloutReportApprovalInput(flow RolloutReportApprovalFlowInput, workItem RolloutReportApprovalWorkItemInput) RolloutReportApprovalInput {
	return RolloutReportApprovalInput{
		ManualApprovalCheckpoint: flow.ManualApprovalCheckpoint,
		OperatorHandoff:          flow.OperatorHandoff,
		WorkItem:                 workItem.WorkItem,
		Checklist:                flow.Checklist,
	}
}

func BuildRolloutReportApprovalSection(input RolloutReportApprovalInput) RolloutReportApproval {
	return RolloutReportApproval(input)
}

func BuildRolloutReportGuardedInput(hook RolloutReportGuardedHookInput, enforcement RolloutReportGuardedEnforcementInput) RolloutReportGuardedInput {
	return RolloutReportGuardedInput{
		Hook:         hook.Hook,
		HookPolicy:   hook.HookPolicy,
		WouldEnforce: enforcement.WouldEnforce,
		EnforceHint:  enforcement.EnforceHint,
		Overview:     enforcement.Overview,
	}
}

func ApplyRolloutReportApprovalSection(details *RolloutReportDetails, section RolloutReportApproval) {
	if details == nil {
		return
	}

	details.Approval = section
}

func BuildRolloutReportGuardedSection(input RolloutReportGuardedInput) RolloutReportGuarded {
	return RolloutReportGuarded(input)
}

func BuildRolloutReportSections(input RolloutReportSectionsInput) RolloutReportSections {
	return RolloutReportSections{
		Surface:        BuildRolloutReportSurfaceSection(input.Surface),
		Approval:       BuildRolloutReportApprovalSection(input.Approval),
		GuardedCutover: BuildRolloutReportGuardedSection(input.GuardedCutover),
	}
}

func ApplyRolloutReportGuardedSection(details *RolloutReportDetails, section RolloutReportGuarded) {
	if details == nil {
		return
	}

	details.GuardedCutover = section
}

func (d *RolloutReportDetails) IsEmpty() bool {
	return d == nil || d.ReportID == ""
}
