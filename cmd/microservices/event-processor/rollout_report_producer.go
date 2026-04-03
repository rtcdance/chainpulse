package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"chainpulse/pkg/plugins/api"
)

const (
	eventProcessorRolloutSkeletonReason      = "event-processor rollout producer skeleton is not yet wired to ownership runtime state"
	eventProcessorRolloutRuntimeWiringReason = "event-processor rollout producer reflects local runtime dependency wiring; ownership runtime state is not yet wired"
)

type eventProcessorRolloutRuntimeState struct {
	DatabaseReady              bool
	EventStoreReady            bool
	KafkaReady                 bool
	MetadataStoreReady         bool
	ProcessorRuntimeReady      bool
	ConsumeLoopOwned           bool
	EventStoreHealthStatus     string
	EventStoreHealthMessage    string
	MetadataStoreHealthStatus  string
	MetadataStoreHealthMessage string
	KafkaHealthStatus          string
	KafkaHealthMessage         string
	ProcessorHealthStatus      string
	ProcessorHealthMessage     string
	ConsumeLoopStatus          string
	ConsumeLoopLastError       string
	KafkaMessageCount          int64
	KafkaErrorCount            int64
	KafkaActivityState         string
	ActiveConsumers            int64
	ConsumerLag                int64
	ConsumerLagSeverity        string
	ConsumerOffset             int64
	ConsumerProgressState      string
	ConsumerProgressPosture    string
	ConsumerBacklogHint        string
	ProcessedEventCount        int64
	FailedEventCount           int64
	DuplicateEventCount        int64
	ConfiguredConsumeTopics    int
	ActiveConsumeTopics        int
	SharedRuntimeShadowEnabled bool
	SharedRuntimeChainCount    int
	SharedRuntimeProcessed     int64
	SharedRuntimeSkipped       int64
	SharedRuntimeFailures      int64
	SharedRuntimeLastChain     string
	SharedRuntimeLastCursor    string
	SharedRuntimeLastBlock     uint64
	SharedRuntimeLastError     string
}

type eventProcessorRolloutWiringCompleteness struct {
	Mode           string
	AdvisoryStatus string
	AdvisoryReady  bool
	EnabledSignals []string
	MissingSignals []string
	PostureHint    string
	Reason         string
}

func buildEventProcessorRolloutSummary() api.RolloutReportSummary {
	return api.RolloutReportSummary{
		ShadowOwnedEvents: 0,
		LegacyOwnedEvents: 0,
		OwnershipChains:   0,
	}
}

func newEventProcessorRolloutReportProducer(instanceID string, stateProvider func() eventProcessorRolloutRuntimeState) api.RolloutReportProducer {
	return api.RolloutReportProducerFunc(func(ctx context.Context) *api.RolloutReportDetails {
		_ = ctx

		details := api.NewRolloutReportDetailsFromMetadata(api.NewOwnershipRolloutReportMetadata(
			"event-processor",
			"event-processor-ownership-rollout-runtime",
			"microservice",
			"microservice",
			time.Now().Unix(),
		))
		if instanceID != "" {
			details.ReportSource = "microservice:" + instanceID
		}

		runtimeState := eventProcessorRolloutRuntimeState{}
		if stateProvider != nil {
			runtimeState = stateProvider()
		}

		if runtimeState.DatabaseReady ||
			runtimeState.EventStoreReady ||
			runtimeState.MetadataStoreReady ||
			runtimeState.KafkaReady {
			applyEventProcessorRuntimeDerivedRollout(details, runtimeState)
			return details
		}

		applyEventProcessorSkeletonRollout(details)
		return details
	})
}

func applyEventProcessorSkeletonRollout(details *api.RolloutReportDetails) {
	if details == nil {
		return
	}

	applyEventProcessorRolloutReportSections(details, buildEventProcessorSkeletonSections())
}

func applyEventProcessorRuntimeDerivedRollout(details *api.RolloutReportDetails, runtimeState eventProcessorRolloutRuntimeState) {
	if details == nil {
		return
	}

	applyEventProcessorRolloutReportSections(details, buildEventProcessorRuntimeDerivedSections(runtimeState))
}

func buildEventProcessorSkeletonSurfaceInput() api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		api.RolloutReportSurfaceCoreInput{
			Summary: buildEventProcessorRolloutSummary(),
			Mode:    "unavailable",
			Advisory: api.RolloutReportAdvisory{
				Decision: "unknown",
				Status:   "unavailable",
				Ready:    false,
				Reason:   eventProcessorRolloutSkeletonReason,
			},
			Policy: api.RolloutReportPolicy{
				Mode:         "report-only",
				Action:       "report-unknown",
				Reason:       eventProcessorRolloutSkeletonReason,
				Acknowledged: false,
				AckState:     "pending",
			},
			Progression: api.RolloutReportStateReason{
				State:  "unknown",
				Reason: eventProcessorRolloutSkeletonReason,
			},
		},
		api.RolloutReportSurfaceCutoverInput{
			CutoverDryRun: api.RolloutReportAction{
				Action: "would-unknown",
				Reason: eventProcessorRolloutSkeletonReason,
			},
			CutoverCandidate: api.RolloutReportCandidate{
				Eligible: false,
				Reason:   eventProcessorRolloutSkeletonReason,
			},
		},
	)
}

func buildEventProcessorRuntimeDerivedSurfaceInput(completeness eventProcessorRolloutWiringCompleteness) api.RolloutReportSurfaceInput {
	return api.BuildRolloutReportSurfaceInput(
		api.RolloutReportSurfaceCoreInput{
			Summary: buildEventProcessorRolloutSummary(),
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

func buildEventProcessorSkeletonApprovalInput() api.RolloutReportApprovalInput {
	return api.BuildRolloutReportApprovalInput(
		api.RolloutReportApprovalFlowInput{
			ManualApprovalCheckpoint: api.RolloutReportStateReason{
				State:  "unknown",
				Reason: eventProcessorRolloutSkeletonReason,
			},
			OperatorHandoff: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: eventProcessorRolloutSkeletonReason,
			},
			Checklist: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: eventProcessorRolloutSkeletonReason,
			},
		},
		api.RolloutReportApprovalWorkItemInput{
			WorkItem: api.RolloutReportApprovalItem{
				Status:       "investigate",
				Owner:        "platform-team/runtime-owners",
				ReviewFields: "database_ready,event_store_ready,metadata_store_ready,kafka_ready",
				Reason:       eventProcessorRolloutSkeletonReason,
			},
		},
	)
}

func buildEventProcessorRuntimeDerivedApprovalInput(completeness eventProcessorRolloutWiringCompleteness) api.RolloutReportApprovalInput {
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
				ReviewFields: "database_ready,event_store_ready,metadata_store_ready,kafka_ready",
				Reason:       completeness.Reason,
			},
		},
	)
}

func buildEventProcessorSkeletonGuardedInput() api.RolloutReportGuardedInput {
	return api.BuildRolloutReportGuardedInput(
		api.RolloutReportGuardedHookInput{
			Hook: api.RolloutReportAction{
				Action: "noop-investigate",
				Reason: eventProcessorRolloutSkeletonReason,
			},
			HookPolicy: api.RolloutReportModeAction{
				Mode:   "noop-only",
				Action: "noop-investigate",
				Reason: eventProcessorRolloutSkeletonReason,
			},
		},
		api.RolloutReportGuardedEnforcementInput{
			WouldEnforce: api.RolloutReportAction{
				Action: "would-investigate",
				Reason: eventProcessorRolloutSkeletonReason,
			},
			EnforceHint: api.RolloutReportStateReason{
				State:  "investigate-before-enforce",
				Reason: eventProcessorRolloutSkeletonReason,
			},
			Overview: api.RolloutReportStateReason{
				State:  "investigate",
				Reason: eventProcessorRolloutSkeletonReason,
			},
		},
	)
}

func buildEventProcessorRuntimeDerivedGuardedInput(completeness eventProcessorRolloutWiringCompleteness) api.RolloutReportGuardedInput {
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

func classifyEventProcessorRolloutWiringCompleteness(runtimeState eventProcessorRolloutRuntimeState) eventProcessorRolloutWiringCompleteness {
	enabled := make([]string, 0, 4)
	missing := make([]string, 0, 4)

	collectEventProcessorRuntimeSignal(&enabled, &missing, "database_ready", runtimeState.DatabaseReady)
	collectEventProcessorRuntimeSignal(&enabled, &missing, "event_store_ready", runtimeState.EventStoreReady)
	collectEventProcessorRuntimeSignal(&enabled, &missing, "metadata_store_ready", runtimeState.MetadataStoreReady)
	collectEventProcessorRuntimeSignal(&enabled, &missing, "kafka_ready", runtimeState.KafkaReady)

	mode := "partially-wired"
	advisoryStatus := "partial-runtime-wiring"
	advisoryReady := false
	if len(missing) == 0 && len(enabled) > 0 {
		mode = "runtime-wired"
		advisoryStatus = classifyEventProcessorRuntimeHealthAdvisoryStatus(runtimeState)
		advisoryReady = advisoryStatus == "runtime-wired"
	}

	parts := []string{eventProcessorRolloutRuntimeWiringReason}
	if len(enabled) > 0 {
		parts = append(parts, "enabled: "+strings.Join(enabled, ","))
	}
	if len(missing) > 0 {
		parts = append(parts, "missing: "+strings.Join(missing, ","))
	}
	appendEventProcessorRuntimeHealthReason(&parts, "event_store", runtimeState.EventStoreHealthStatus, runtimeState.EventStoreHealthMessage)
	appendEventProcessorRuntimeHealthReason(&parts, "metadata_store", runtimeState.MetadataStoreHealthStatus, runtimeState.MetadataStoreHealthMessage)
	appendEventProcessorRuntimeHealthReason(&parts, "kafka", runtimeState.KafkaHealthStatus, runtimeState.KafkaHealthMessage)
	appendEventProcessorKafkaActivityReason(&parts, runtimeState.KafkaMessageCount, runtimeState.KafkaErrorCount, runtimeState.KafkaActivityState)
	progress := api.BuildRolloutExecutionProgress(api.RolloutExecutionProgressInput{
		Consumer: api.RolloutConsumerProgressSnapshot{
			ActiveConsumers: runtimeState.ActiveConsumers,
			Lag:             runtimeState.ConsumerLag,
			CurrentOffset:   runtimeState.ConsumerOffset,
			ProgressState:   runtimeState.ConsumerProgressState,
		},
	})
	api.AppendRolloutExecutionProgressReason(&parts, progress)
	api.AppendRolloutExecutionProgressPostureReason(&parts, progress)
	consumerLagSeverity := strings.TrimSpace(runtimeState.ConsumerLagSeverity)
	if consumerLagSeverity == "" {
		consumerLagSeverity = classifyEventProcessorConsumerLagSeverity(runtimeState.ConsumerLag)
	}
	if consumerLagSeverity != "" {
		parts = append(parts, "consumer_lag_severity: "+consumerLagSeverity)
	}
	consumerBacklogHint := strings.TrimSpace(runtimeState.ConsumerBacklogHint)
	if consumerBacklogHint == "" {
		consumerProgressPosture := strings.TrimSpace(runtimeState.ConsumerProgressPosture)
		if consumerProgressPosture == "" {
			consumerProgressPosture = api.BuildRolloutExecutionProgressPosture(progress).Consumer
		}
		consumerBacklogHint = classifyEventProcessorConsumerBacklogHint(
			consumerProgressPosture,
			consumerLagSeverity,
		)
	}
	if consumerBacklogHint != "" {
		api.AppendRolloutExecutionOperatorHintReason(&parts, api.RolloutExecutionOperatorHint{
			Consumer: consumerBacklogHint,
		})
	}
	postureHint := classifyEventProcessorRolloutPostureHint(mode, advisoryStatus, len(missing))
	if postureHint != "" {
		parts = append(parts, "rollout_posture_hint: "+postureHint)
	}

	return eventProcessorRolloutWiringCompleteness{
		Mode:           mode,
		AdvisoryStatus: advisoryStatus,
		AdvisoryReady:  advisoryReady,
		EnabledSignals: enabled,
		MissingSignals: missing,
		PostureHint:    postureHint,
		Reason:         strings.Join(parts, "; "),
	}
}

func classifyEventProcessorRolloutPostureHint(mode, advisoryStatus string, missingSignals int) string {
	switch {
	case mode == "partially-wired" && missingSignals > 0:
		return "finish wiring the missing event-processor runtime dependencies before treating this rollout as ready"
	case advisoryStatus == "runtime-wired":
		return "observe the runtime-wired event-processor rollout while local processing dependencies remain healthy"
	case advisoryStatus == "runtime-wired-degraded":
		return "hold the runtime-wired event-processor rollout and investigate degraded processing dependency health before treating it as ready"
	case advisoryStatus == "runtime-wired-unhealthy":
		return "restore unhealthy event-processor processing dependencies before relying on this runtime-wired rollout"
	case advisoryStatus == "runtime-wired-health-unknown":
		return "verify event-processor processing dependency health before treating this runtime-wired rollout as ready"
	default:
		return ""
	}
}

func classifyEventProcessorRuntimeHealthAdvisoryStatus(runtimeState eventProcessorRolloutRuntimeState) string {
	statuses := []string{
		normalizeEventProcessorHealthStatus(runtimeState.EventStoreHealthStatus),
		normalizeEventProcessorHealthStatus(runtimeState.MetadataStoreHealthStatus),
		normalizeEventProcessorHealthStatus(runtimeState.KafkaHealthStatus),
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

func normalizeEventProcessorHealthStatus(status string) string {
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

func appendEventProcessorRuntimeHealthReason(parts *[]string, component, status, message string) {
	normalizedStatus := normalizeEventProcessorHealthStatus(status)
	if normalizedStatus == "unknown" && strings.TrimSpace(message) == "" {
		return
	}

	*parts = append(*parts, component+"_status: "+normalizedStatus)
	if trimmedMessage := strings.TrimSpace(message); trimmedMessage != "" {
		*parts = append(*parts, component+"_message: "+trimmedMessage)
	}
}

func appendEventProcessorKafkaActivityReason(parts *[]string, messageCount, errorCount int64, activityState string) {
	if messageCount > 0 {
		*parts = append(*parts, "kafka_message_count: "+strconv.FormatInt(messageCount, 10))
	}
	if errorCount > 0 {
		*parts = append(*parts, "kafka_error_count: "+strconv.FormatInt(errorCount, 10))
	}
	if trimmedState := strings.TrimSpace(activityState); trimmedState != "" {
		*parts = append(*parts, "kafka_activity_state: "+trimmedState)
	}
}

func classifyEventProcessorConsumerBacklogHint(progressPosture, lagSeverity string) string {
	switch {
	case progressPosture == "consumer-backlog" && lagSeverity == "backlog-high":
		return "consumer backlog is high; prioritize drain and investigate processor throughput"
	case progressPosture == "consumer-backlog" && lagSeverity == "backlog-medium":
		return "consumer backlog is building; monitor drain rate and investigate if it persists"
	case progressPosture == "consumer-backlog" && lagSeverity == "backlog-low":
		return "consumer backlog is present but small; continue observing drain progression"
	case progressPosture == "consumer-advancing":
		return "consumer progress is advancing; continue observing steady backlog drain"
	case progressPosture == "consumer-idle":
		return "consumer group appears idle; verify whether no work is expected"
	default:
		return ""
	}
}

func collectEventProcessorRuntimeSignal(enabled, missing *[]string, label string, active bool) {
	if active {
		*enabled = append(*enabled, label)
		return
	}
	*missing = append(*missing, label)
}
