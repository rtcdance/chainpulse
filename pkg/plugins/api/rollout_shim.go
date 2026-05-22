package api

import "github.com/rtcdance/chainpulse/pkg/plugins/api/rollout"

// Backward-compatible type aliases for types moved to api/rollout.
type (
	MonolithOwnershipParityRecommendationBundle = rollout.MonolithOwnershipParityRecommendationBundle
	OwnershipParityApprovalWorkItemInput        = rollout.OwnershipParityApprovalWorkItemInput
	RolloutConsumerProgressSnapshot             = rollout.RolloutConsumerProgressSnapshot
	RolloutExecutionOperatorHint                = rollout.RolloutExecutionOperatorHint
	RolloutExecutionProgress                    = rollout.RolloutExecutionProgress
	RolloutExecutionProgressInput               = rollout.RolloutExecutionProgressInput
	RolloutExecutionProgressPosture             = rollout.RolloutExecutionProgressPosture
	RolloutPollProgressSnapshot                 = rollout.RolloutPollProgressSnapshot
	RolloutReportAction                         = rollout.RolloutReportAction
	RolloutReportAdvisory                       = rollout.RolloutReportAdvisory
	RolloutReportApproval                       = rollout.RolloutReportApproval
	RolloutReportApprovalFlowInput              = rollout.RolloutReportApprovalFlowInput
	RolloutReportApprovalInput                  = rollout.RolloutReportApprovalInput
	RolloutReportApprovalItem                   = rollout.RolloutReportApprovalItem
	RolloutReportApprovalWorkItemInput          = rollout.RolloutReportApprovalWorkItemInput
	RolloutReportCandidate                      = rollout.RolloutReportCandidate
	RolloutReportDetails                        = rollout.RolloutReportDetails
	RolloutReportGuarded                        = rollout.RolloutReportGuarded
	RolloutReportGuardedEnforcementInput        = rollout.RolloutReportGuardedEnforcementInput
	RolloutReportGuardedHookInput               = rollout.RolloutReportGuardedHookInput
	RolloutReportGuardedInput                   = rollout.RolloutReportGuardedInput
	RolloutReportMetadata                       = rollout.RolloutReportMetadata
	RolloutReportModeAction                     = rollout.RolloutReportModeAction
	RolloutReportPolicy                         = rollout.RolloutReportPolicy
	RolloutReportProducer                       = rollout.RolloutReportProducer
	RolloutReportProducerFunc                   = rollout.RolloutReportProducerFunc
	RolloutReportResponse                       = rollout.RolloutReportResponse
	RolloutReportSections                       = rollout.RolloutReportSections
	RolloutReportSectionsInput                  = rollout.RolloutReportSectionsInput
	RolloutReportStateReason                    = rollout.RolloutReportStateReason
	RolloutReportSummary                        = rollout.RolloutReportSummary
	RolloutReportSurfaceCoreInput               = rollout.RolloutReportSurfaceCoreInput
	RolloutReportSurfaceCutoverInput            = rollout.RolloutReportSurfaceCutoverInput
	RolloutReportSurfaceInput                   = rollout.RolloutReportSurfaceInput
	RolloutReportSurfaceSection                 = rollout.RolloutReportSurfaceSection
	RouteOwnershipParitySource                  = rollout.RouteOwnershipParitySource
	RouteOwnershipParitySourceFunc              = rollout.RouteOwnershipParitySourceFunc
	RouteOwnershipParitySourceSnapshot          = rollout.RouteOwnershipParitySourceSnapshot
	RouteOwnershipParityState                   = rollout.RouteOwnershipParityState
)

// Backward-compatible const re-exports.
const (
	OwnershipRolloutSchemaFamily      = rollout.OwnershipRolloutSchemaFamily
	OwnershipRolloutReportVersion     = rollout.OwnershipRolloutReportVersion
	OwnershipRolloutReportScope       = rollout.OwnershipRolloutReportScope
	OwnershipRolloutReportMode        = rollout.OwnershipRolloutReportMode
	OwnershipRuntimeParityReviewField = rollout.OwnershipRuntimeParityReviewField
)

// Backward-compatible function re-exports.
var (
	AppendMonolithOwnershipParityReason                         = rollout.AppendMonolithOwnershipParityReason
	AppendOwnershipParityHintReason                             = rollout.AppendOwnershipParityHintReason
	AppendRolloutConsumerProgressReason                         = rollout.AppendRolloutConsumerProgressReason
	AppendRolloutExecutionOperatorHintReason                    = rollout.AppendRolloutExecutionOperatorHintReason
	AppendRolloutExecutionProgressPostureReason                 = rollout.AppendRolloutExecutionProgressPostureReason
	AppendRolloutExecutionProgressReason                        = rollout.AppendRolloutExecutionProgressReason
	AppendRolloutPollProgressReason                             = rollout.AppendRolloutPollProgressReason
	AppendRouteOwnershipParityStateReason                       = rollout.AppendRouteOwnershipParityStateReason
	ApplyRolloutReportApprovalSection                           = rollout.ApplyRolloutReportApprovalSection
	ApplyRolloutReportGuardedSection                            = rollout.ApplyRolloutReportGuardedSection
	ApplyRolloutReportSurfaceSection                            = rollout.ApplyRolloutReportSurfaceSection
	BuildMonolithOwnershipParityActionGuidance                  = rollout.BuildMonolithOwnershipParityActionGuidance
	BuildMonolithOwnershipParityHint                            = rollout.BuildMonolithOwnershipParityHint
	BuildMonolithOwnershipParityRecommendationBundle            = rollout.BuildMonolithOwnershipParityRecommendationBundle
	BuildMonolithOwnershipParityTargetDecision                  = rollout.BuildMonolithOwnershipParityTargetDecision
	BuildOwnershipParityApprovalWorkItem                        = rollout.BuildOwnershipParityApprovalWorkItem
	BuildOwnershipParityReviewFields                            = rollout.BuildOwnershipParityReviewFields
	BuildRolloutExecutionProgress                               = rollout.BuildRolloutExecutionProgress
	BuildRolloutExecutionProgressPosture                        = rollout.BuildRolloutExecutionProgressPosture
	BuildRolloutReportApprovalInput                             = rollout.BuildRolloutReportApprovalInput
	BuildRolloutReportApprovalSection                           = rollout.BuildRolloutReportApprovalSection
	BuildRolloutReportGuardedInput                              = rollout.BuildRolloutReportGuardedInput
	BuildRolloutReportGuardedSection                            = rollout.BuildRolloutReportGuardedSection
	BuildRolloutReportSections                                  = rollout.BuildRolloutReportSections
	BuildRolloutReportSurfaceInput                              = rollout.BuildRolloutReportSurfaceInput
	BuildRolloutReportSurfaceSection                            = rollout.BuildRolloutReportSurfaceSection
	BuildRouteOwnershipParityHint                               = rollout.BuildRouteOwnershipParityHint
	BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails = rollout.BuildRouteOwnershipParitySourceSnapshotFromReadinessDetails
	BuildRouteOwnershipParityState                              = rollout.BuildRouteOwnershipParityState
	BuildRouteOwnershipParityStateFromSource                    = rollout.BuildRouteOwnershipParityStateFromSource
	ClassifyMonolithOwnershipParityPosture                      = rollout.ClassifyMonolithOwnershipParityPosture
	NewOwnershipRolloutReportMetadata                           = rollout.NewOwnershipRolloutReportMetadata
	NewRolloutReportDetailsFromMetadata                         = rollout.NewRolloutReportDetailsFromMetadata
	ValidateMicroserviceOwnershipParityMarker                   = rollout.ValidateMicroserviceOwnershipParityMarker
	ValidateMicroserviceRolloutMetadataParity                   = rollout.ValidateMicroserviceRolloutMetadataParity
	ValidateMicroserviceRuntimeDerivedRolloutParity             = rollout.ValidateMicroserviceRuntimeDerivedRolloutParity
	ValidateRolloutExecutionOperatorHintReasonCoverage          = rollout.ValidateRolloutExecutionOperatorHintReasonCoverage
	ValidateRolloutExecutionProgressPostureReasonCoverage       = rollout.ValidateRolloutExecutionProgressPostureReasonCoverage
	ValidateRolloutExecutionProgressReasonCoverage              = rollout.ValidateRolloutExecutionProgressReasonCoverage
	ValidateRouteMonolithOwnershipParityReason                  = rollout.ValidateRouteMonolithOwnershipParityReason
	ValidateRouteMonolithOwnershipParityRecommendationBundle    = rollout.ValidateRouteMonolithOwnershipParityRecommendationBundle
)
