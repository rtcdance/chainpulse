package main

import (
	"strings"

	"chainpulse/pkg/plugins/api"
)

func buildEventProcessorKafkaConsumerProgressSnapshot(
	kafkaHealth eventProcessorKafkaHealthProvider,
	activityState string,
) api.RolloutConsumerProgressSnapshot {
	activeConsumers, lag, currentOffset := eventProcessorKafkaConsumerProgressFields(kafkaHealth)

	return api.RolloutConsumerProgressSnapshot{
		ActiveConsumers: activeConsumers,
		Lag:             lag,
		CurrentOffset:   currentOffset,
		ProgressState:   classifyEventProcessorConsumerProgressState(activeConsumers, lag, activityState),
	}
}

func eventProcessorKafkaConsumerProgressFields(kafkaHealth eventProcessorKafkaHealthProvider) (int64, int64, int64) {
	if kafkaHealth == nil {
		return 0, 0, 0
	}

	healthStatus := kafkaHealth.Health()

	var activeConsumers int64
	if healthStatus != nil && healthStatus.Details != nil {
		activeConsumers = eventProcessorInt64FromInterface(healthStatus.Details, "active_consumers")
	}
	if activeConsumers == 0 {
		if statusProvider, ok := kafkaHealth.(eventProcessorKafkaConsumerGroupStatusProvider); ok {
			activeConsumers = eventProcessorInt64FromInterface(statusProvider.GetConsumerGroupStatus(), "active_consumers")
		}
	}

	var lag int64
	if healthStatus != nil && healthStatus.Details != nil {
		lag = eventProcessorInt64FromInterface(healthStatus.Details, "consumer_group_lag")
	}
	if lag == 0 {
		if statusProvider, ok := kafkaHealth.(eventProcessorKafkaConsumerGroupStatusProvider); ok {
			lag = eventProcessorInt64FromInterface(statusProvider.GetConsumerGroupStatus(), "lag")
		}
	}
	if lag == 0 {
		if metricsProvider, ok := kafkaHealth.(eventProcessorKafkaConsumerGroupMetricsProvider); ok {
			lag = eventProcessorInt64Metric(metricsProvider.GetConsumerGroupMetrics(), "lag")
		}
	}

	var currentOffset int64
	if healthStatus != nil && healthStatus.Details != nil {
		currentOffset = eventProcessorInt64FromInterface(healthStatus.Details, "max_tracked_offset")
	}
	if currentOffset == 0 {
		if statusProvider, ok := kafkaHealth.(eventProcessorKafkaConsumerGroupStatusProvider); ok {
			currentOffset = eventProcessorInt64FromInterface(statusProvider.GetConsumerGroupStatus(), "max_tracked_offset")
		}
	}

	return activeConsumers, lag, currentOffset
}

func classifyEventProcessorConsumerProgressState(activeConsumers, lag int64, activityState string) string {
	switch {
	case activeConsumers <= 0:
		return "idle"
	case lag > 0:
		return "lagging"
	case strings.TrimSpace(activityState) == "active":
		return "active"
	default:
		return "monitoring"
	}
}

func classifyEventProcessorConsumerLagSeverity(lag int64) string {
	switch {
	case lag <= 0:
		return ""
	case lag >= 100:
		return "backlog-high"
	case lag >= 20:
		return "backlog-medium"
	default:
		return "backlog-low"
	}
}
