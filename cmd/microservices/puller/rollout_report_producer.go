package main

import (
	"context"
	"strings"
	"time"

	"chainpulse/pkg/plugins/api"
)

const (
	pullerRolloutSkeletonReason      = "puller rollout producer skeleton is not yet wired to ownership runtime state"
	pullerRolloutRuntimeWiringReason = "puller rollout producer reflects local runtime dependency wiring; ownership runtime state is not yet wired"
)

type pullerRolloutRuntimeState struct {
	BlockchainRPCsConfigured      bool
	DatabaseReady                 bool
	KafkaReady                    bool
	PullerLoopConfigured          bool
	ExecutionRuntimeEnabled       bool
	ConfiguredPullerCount         int
	AttachedPullerCount           int
	DatabaseHealthStatus          string
	DatabaseHealthMessage         string
	KafkaHealthStatus             string
	KafkaHealthMessage            string
	PollCount                     int64
	LastPollUnix                  int64
	ObservedBlock                 int64
	ProcessedBlock                int64
	BlockGap                      int64
	CheckpointProgressState       string
	BlocksUntilCheckpoint         int64
	PersistedCheckpointBlock      int64
	BlocksSinceCheckpoint         int64
	PersistedCheckpointState      string
	ReorgCheckpointState          string
	ReorgCheckpointBlock          int64
	CheckpointChainSummary        string
	CheckpointChainPostureSummary string
	CheckpointCoverageHint        string
	CheckpointCoveragePosture     string
	CheckpointRecoveryHint        string
	PollActivityState             string
	PublishedEvents               int64
	PublishedMessages             int64
	SharedRuntimeCount            int
	SharedRuntimeProcessed        int64
	SharedRuntimeSkipped          int64
	SharedRuntimeFailures         int64
	SharedRuntimeLastChain        string
	SharedRuntimeLastCursor       string
	SharedRuntimeLastBlock        int64
	SharedRuntimeLastError        string
}

type pullerRolloutWiringCompleteness struct {
	Mode           string
	AdvisoryStatus string
	AdvisoryReady  bool
	EnabledSignals []string
	MissingSignals []string
	PostureHint    string
	Reason         string
}

func buildPullerRolloutSummary() api.RolloutReportSummary {
	return api.RolloutReportSummary{
		ShadowOwnedEvents: 0,
		LegacyOwnedEvents: 0,
		OwnershipChains:   0,
	}
}

func newPullerRolloutReportProducer(instanceID string, stateProvider func() pullerRolloutRuntimeState) api.RolloutReportProducer {
	return api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx

		details := api.NewRolloutReportDetailsFromMetadata(api.NewOwnershipRolloutReportMetadata(
			"puller",
			"puller-ownership-rollout-runtime",
			"microservice",
			"microservice",
			time.Now().Unix(),
		))
		if instanceID != "" {
			details.ReportSource = "microservice:" + instanceID
		}

		runtimeState := pullerRolloutRuntimeState{}
		if stateProvider != nil {
			runtimeState = stateProvider()
		}

		if runtimeState.DatabaseReady ||
			runtimeState.KafkaReady ||
			runtimeState.PullerLoopConfigured ||
			runtimeState.BlockchainRPCsConfigured {
			applyPullerRuntimeDerivedRollout(details, runtimeState)
			return details
		}

		applyPullerSkeletonRollout(details)
		return details
	})
}

func applyPullerSkeletonRollout(details *api.RolloutReportDetails) {
	if details == nil {
		return
	}
	applyPullerRolloutReportSections(details, buildPullerSkeletonSections())
}

func applyPullerRuntimeDerivedRollout(details *api.RolloutReportDetails, runtimeState pullerRolloutRuntimeState) {
	if details == nil {
		return
	}
	applyPullerRolloutReportSections(details, buildPullerRuntimeDerivedSections(runtimeState))
}

func buildPullerSkeletonSurfaceInput() api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		api.RolloutReportSurfaceCoreInput{
			Summary: buildPullerRolloutSummary(),
			Mode:    "unavailable",
			Advisory: api.RolloutReportAdvisory{
				Decision: "unknown",
				Status:   "unavailable",
				Ready:    false,
				Reason:   pullerRolloutSkeletonReason,
			},
			Policy: api.RolloutReportPolicy{
				Mode:         "report-only",
				Action:       "report-unknown",
				Reason:       pullerRolloutSkeletonReason,
				Acknowledged: false,
				AckState:     "pending",
			},
			Progression: api.RolloutReportStateReason{
				State:  "unknown",
				Reason: pullerRolloutSkeletonReason,
			},
		},
		api.RolloutReportSurfaceCutoverInput{
			CutoverDryRun: api.RolloutReportAction{
				Action: "would-unknown",
				Reason: pullerRolloutSkeletonReason,
			},
			CutoverCandidate: api.RolloutReportCandidate{
				Eligible: false,
				Reason:   pullerRolloutSkeletonReason,
			},
		},
	)
}

func buildPullerRuntimeDerivedSurfaceInput(completeness pullerRolloutWiringCompleteness) api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		api.RolloutReportSurfaceCoreInput{
			Summary: buildPullerRolloutSummary(),
			Mode:    completeness.Mode,
			Advisory: api.RolloutReportAdvisory{
				Decision: "hold",
				Status:   completeness.AdvisoryStatus,
				Ready:    completeness.AdvisoryReady,
				Reason:   completeness.Reason,
			},
			Policy: api.RolloutReportPolicy{
				Mode:         "report-only",
				Action:       "report-hold",
				Reason:       completeness.Reason,
				Acknowledged: false,
				AckState:     "pending",
			},
			Progression: api.RolloutReportStateReason{
				State:  "observe",
				Reason: completeness.Reason,
			},
		},
		api.RolloutReportSurfaceCutoverInput{
			CutoverDryRun: api.RolloutReportAction{
				Action: "would-hold",
				Reason: completeness.Reason,
			},
			CutoverCandidate: api.RolloutReportCandidate{
				Eligible: false,
				Reason:   completeness.Reason,
			},
		},
	)
}

func buildPullerSkeletonApprovalInput() api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		api.RolloutReportApprovalFlowInput{
			ManualApprovalCheckpoint: api.RolloutReportStateReason{
				State:  "unknown",
				Reason: pullerRolloutSkeletonReason,
			},
			OperatorHandoff: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: pullerRolloutSkeletonReason,
			},
			Checklist: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: pullerRolloutSkeletonReason,
			},
		},
		api.RolloutReportApprovalWorkItemInput{
			WorkItem: api.RolloutReportApprovalItem{
				Status:       "investigate",
				Owner:        "platform-team/runtime-owners",
				ReviewFields: "database_ready,kafka_ready,puller_loop_configured,blockchain_rpcs_configured",
				Reason:       pullerRolloutSkeletonReason,
			},
		},
	)
}

func buildPullerRuntimeDerivedApprovalInput(completeness pullerRolloutWiringCompleteness) api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		api.RolloutReportApprovalFlowInput{
			ManualApprovalCheckpoint: api.RolloutReportStateReason{
				State:  "inactive",
				Reason: completeness.Reason,
			},
			OperatorHandoff: api.RolloutReportStateReason{
				State:  "none",
				Reason: completeness.Reason,
			},
			Checklist: api.RolloutReportStateReason{
				State:  "incomplete",
				Reason: completeness.Reason,
			},
		},
		api.RolloutReportApprovalWorkItemInput{
			WorkItem: api.RolloutReportApprovalItem{
				Status:       "none",
				Owner:        "none",
				ReviewFields: "database_ready,kafka_ready,puller_loop_configured,blockchain_rpcs_configured",
				Reason:       completeness.Reason,
			},
		},
	)
}

func buildPullerSkeletonGuardedInput() api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		api.RolloutReportGuardedHookInput{
			Hook: api.RolloutReportAction{
				Action: "noop-investigate",
				Reason: pullerRolloutSkeletonReason,
			},
			HookPolicy: api.RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-investigate",
				Reason: pullerRolloutSkeletonReason,
			},
		},
		api.RolloutReportGuardedEnforcementInput{
			WouldEnforce: api.RolloutReportAction{
				Action: "would-investigate",
				Reason: pullerRolloutSkeletonReason,
			},
			EnforceHint: api.RolloutReportStateReason{
				State:  "investigate-before-enforce",
				Reason: pullerRolloutSkeletonReason,
			},
			Overview: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: pullerRolloutSkeletonReason,
			},
		},
	)
}

func buildPullerRuntimeDerivedGuardedInput(completeness pullerRolloutWiringCompleteness) api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		api.RolloutReportGuardedHookInput{
			Hook: api.RolloutReportAction{
				Action: "noop-hold",
				Reason: completeness.Reason,
			},
			HookPolicy: api.RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-hold",
				Reason: completeness.Reason,
			},
		},
		api.RolloutReportGuardedEnforcementInput{
			WouldEnforce: api.RolloutReportAction{
				Action: "would-hold",
				Reason: completeness.Reason,
			},
			EnforceHint: api.RolloutReportStateReason{
				State:  "hold-before-enforce",
				Reason: completeness.Reason,
			},
			Overview: api.RolloutReportStateReason{
				State:  "hold",
				Reason: completeness.Reason,
			},
		},
	)
}

func classifyPullerRolloutWiringCompleteness(runtimeState pullerRolloutRuntimeState) pullerRolloutWiringCompleteness {
	//nolint:funlen // Classification function checks many signals.
	enabled := make([]string, 0, 4)
	missing := make([]string, 0, 4)

	collectPullerRuntimeSignal(&enabled, &missing, "database_ready", runtimeState.DatabaseReady)
	collectPullerRuntimeSignal(&enabled, &missing, "kafka_ready", runtimeState.KafkaReady)
	collectPullerRuntimeSignal(&enabled, &missing, "puller_loop_configured", runtimeState.PullerLoopConfigured)
	collectPullerRuntimeSignal(&enabled, &missing, "blockchain_rpcs_configured", runtimeState.BlockchainRPCsConfigured)

	mode := "partially-wired"
	advisoryStatus := "partial-runtime-wiring"
	advisoryReady := false
	if len(missing) == 0 && len(enabled) > 0 {
		mode = "runtime-wired"
		advisoryStatus = classifyPullerRuntimeHealthAdvisoryStatus(runtimeState)
		advisoryReady = advisoryStatus == "runtime-wired"
	}

	parts := []string{pullerRolloutRuntimeWiringReason}
	if len(enabled) > 0 {
		parts = append(parts, "enabled: "+strings.Join(enabled, ","))
	}
	if len(missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(missing, ","))
	}
	appendPullerRuntimeHealthReason(&parts, "database", runtimeState.DatabaseHealthStatus, runtimeState.DatabaseHealthMessage)
	appendPullerRuntimeHealthReason(&parts, "kafka", runtimeState.KafkaHealthStatus, runtimeState.KafkaHealthMessage)
	progress := api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
		Poll: api.RolloutPollProgressSnapshot{
			PollCount:                runtimeState.PollCount,
			LastPollUnix:             runtimeState.LastPollUnix,
			ObservedBlock:            runtimeState.ObservedBlock,
			ProcessedBlock:           runtimeState.ProcessedBlock,
			BlockGap:                 runtimeState.BlockGap,
			CheckpointState:          runtimeState.CheckpointProgressState,
			BlocksUntilCheckpoint:    runtimeState.BlocksUntilCheckpoint,
			PersistedCheckpointBlock: runtimeState.PersistedCheckpointBlock,
			BlocksSinceCheckpoint:    runtimeState.BlocksSinceCheckpoint,
			PersistedCheckpointState: runtimeState.PersistedCheckpointState,
			ReorgCheckpointState:     runtimeState.ReorgCheckpointState,
			ReorgCheckpointBlock:     runtimeState.ReorgCheckpointBlock,
			ActivityState:            runtimeState.PollActivityState,
		},
	})
	api.AppendRolloutExecutionProgressReason(&parts, progress)
	api.AppendRolloutExecutionProgressPostureReason(&parts, progress)
	if runtimeState.CheckpointChainSummary != "" {
		parts = append(parts, "checkpoint_chain_summary: "+runtimeState.CheckpointChainSummary)
	}
	if runtimeState.CheckpointChainPostureSummary != "" {
		parts = append(parts, "checkpoint_chain_posture_summary: "+runtimeState.CheckpointChainPostureSummary)
	}
	if runtimeState.CheckpointCoverageHint != "" {
		parts = append(parts, "checkpoint_coverage: "+runtimeState.CheckpointCoverageHint)
	}
	if runtimeState.CheckpointCoveragePosture != "" {
		parts = append(parts, "checkpoint_coverage_posture: "+runtimeState.CheckpointCoveragePosture)
	}
	checkpointRecoveryHint := strings.TrimSpace(runtimeState.CheckpointRecoveryHint)
	if checkpointRecoveryHint == "" {
		pollProgressPosture := api.BuildRolloutExecutionProgressPosture(progress).Poll
		checkpointRecoveryHint = classifyPullerCheckpointRecoveryHint(
			runtimeState.ReorgCheckpointState,
			runtimeState.PersistedCheckpointState,
			runtimeState.CheckpointCoveragePosture,
			pollProgressPosture,
		)
	}
	if checkpointRecoveryHint != "" {
		api.AppendRolloutExecutionOperatorHintReason(&parts, api.RolloutExecutionOperatorHint{
			Poll: checkpointRecoveryHint,
		})
	}
	postureHint := classifyPullerRolloutPostureHint(mode, advisoryStatus, len(missing))
	if postureHint != "" {
		parts = append(parts, "rollout_posture_hint: "+postureHint)
	}

	return pullerRolloutWiringCompleteness{
		Mode:           mode,
		AdvisoryStatus: advisoryStatus,
		AdvisoryReady:  advisoryReady,
		EnabledSignals: enabled,
		MissingSignals: missing,
		PostureHint:    postureHint,
		Reason:         strings.Join(parts, "; "),
	}
}

func classifyPullerRolloutPostureHint(mode, advisoryStatus string, missingSignals int) string {
	switch {
	case mode == "partially-wired" && missingSignals > 0:
		return "finish wiring the missing puller runtime dependencies before treating this rollout as ready"
	case advisoryStatus == "runtime-wired":
		return "observe the runtime-wired puller rollout while local ingestion dependencies remain healthy"
	case advisoryStatus == "runtime-wired-degraded":
		return "hold the runtime-wired puller rollout and investigate degraded ingestion dependency health before treating it as ready"
	case advisoryStatus == "runtime-wired-unhealthy":
		return "restore unhealthy puller ingestion dependencies before relying on this runtime-wired rollout"
	case advisoryStatus == "runtime-wired-health-unknown":
		return "verify puller ingestion dependency health before treating this runtime-wired rollout as ready"
	default:
		return ""
	}
}

func classifyPullerRuntimeHealthAdvisoryStatus(runtimeState pullerRolloutRuntimeState) string {
	statuses := []string{
		normalizePullerHealthStatus(runtimeState.DatabaseHealthStatus),
		normalizePullerHealthStatus(runtimeState.KafkaHealthStatus),
	}

	hasUnknown := false
	for _, status := range statuses {
		switch status {
		case "unhealthy":
			return "runtime-wired-unhealthy"
		case "degraded":
			return "runtime-wired-degraded"
		case "unknown":
			hasUnknown = true
		}
	}
	if hasUnknown {
		return "runtime-wired-health-unknown"
	}
	return "runtime-wired"
}

func normalizePullerHealthStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy":
		return "healthy"
	case "degraded":
		return "degraded"
	case "unhealthy":
		return "unhealthy"
	default:
		return "unknown"
	}
}

func appendPullerRuntimeHealthReason(parts *[]string, component, status, message string) {
	normalizedStatus := normalizePullerHealthStatus(status)
	if normalizedStatus == "unknown" && strings.TrimSpace(message) == "" {
		return
	}

	*parts = append(*parts, component+"_status: "+normalizedStatus)
	if trimmedMessage := strings.TrimSpace(message); trimmedMessage != "" {
		*parts = append(*parts, component+"_message: "+trimmedMessage)
	}
}

func classifyPullerCheckpointRecoveryHint(reorgState, persistedState, coveragePosture, pollPosture string) string {
	switch {
	case reorgState == "reorg-risk":
		return "checkpoint reorg risk is active; prioritize reconciliation before relying on persisted progress"
	case persistedState == "persisted-checkpoint-behind" && pollPosture == "poll-catchup":
		return "persisted checkpoint is behind live progress; continue observing catch-up and checkpoint advancement"
	case coveragePosture == "coverage-reconciled":
		return "checkpoint recovery has reconciled recent risk; continue observing fresh checkpoint coverage"
	case coveragePosture == "coverage-healthy":
		return "checkpoint coverage looks healthy; continue observing steady checkpoint advancement"
	case persistedState == "persisted-checkpoint-missing":
		return "no persisted checkpoint has been recorded yet; verify checkpoint creation before relying on recovery posture"
	default:
		return ""
	}
}

func collectPullerRuntimeSignal(enabled, missing *[]string, label string, active bool) {
	if active {
		*enabled = append(*enabled, label)
		return
	}
	*missing = append(*missing, label)
}
