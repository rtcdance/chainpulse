package rollout

import (
	"reflect"
	"testing"
)

func TestAppendRolloutPollProgressReason(t *testing.T) {
	t.Parallel()
	parts := []string{}

	AppendRolloutPollProgressReason(&parts, RolloutPollProgressSnapshot{
		PollCount:                2,
		LastPollUnix:             1712345678,
		ObservedBlock:            120,
		ProcessedBlock:           118,
		BlockGap:                 2,
		CheckpointState:          "checkpoint-pending",
		BlocksUntilCheckpoint:    82,
		PersistedCheckpointBlock: 100,
		BlocksSinceCheckpoint:    18,
		PersistedCheckpointState: "persisted-checkpoint-behind",
		ReorgCheckpointState:     "reorg-reconciled",
		ReorgCheckpointBlock:     200,
		ActivityState:            "active",
	})

	want := []string{
		"poll_count: 2",
		"last_poll_unix: 1712345678",
		"observed_block: 120",
		"processed_block: 118",
		"block_gap: 2",
		"checkpoint_progress_state: checkpoint-pending",
		"blocks_until_checkpoint: 82",
		"persisted_checkpoint_block: 100",
		"blocks_since_checkpoint: 18",
		"persisted_checkpoint_state: persisted-checkpoint-behind",
		"reorg_checkpoint_state: reorg-reconciled",
		"reorg_checkpoint_block: 200",
		"poll_activity_state: active",
	}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("expected %v, got %v", want, parts)
	}
}

func TestAppendRolloutConsumerProgressReason(t *testing.T) {
	t.Parallel()
	parts := []string{}

	AppendRolloutConsumerProgressReason(&parts, RolloutConsumerProgressSnapshot{
		ActiveConsumers: 3,
		Lag:             8,
		CurrentOffset:   144,
		ProgressState:   "lagging",
	})

	want := []string{
		"active_consumers: 3",
		"consumer_lag: 8",
		"consumer_offset: 144",
		"consumer_progress_state: lagging",
	}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("expected %v, got %v", want, parts)
	}
}

func TestAppendRolloutExecutionProgressReason(t *testing.T) {
	t.Parallel()
	parts := []string{}

	AppendRolloutExecutionProgressReason(&parts, RolloutExecutionProgress{
		Poll: &RolloutPollProgressSnapshot{
			PollCount:                2,
			LastPollUnix:             1712345678,
			ObservedBlock:            120,
			ProcessedBlock:           118,
			BlockGap:                 2,
			CheckpointState:          "checkpoint-pending",
			BlocksUntilCheckpoint:    82,
			PersistedCheckpointBlock: 100,
			BlocksSinceCheckpoint:    18,
			PersistedCheckpointState: "persisted-checkpoint-behind",
			ReorgCheckpointState:     "reorg-reconciled",
			ReorgCheckpointBlock:     200,
			ActivityState:            "active",
		},
		Consumer: &RolloutConsumerProgressSnapshot{
			ActiveConsumers: 3,
			Lag:             8,
			CurrentOffset:   144,
			ProgressState:   "lagging",
		},
	})

	want := []string{
		"poll_count: 2",
		"last_poll_unix: 1712345678",
		"observed_block: 120",
		"processed_block: 118",
		"block_gap: 2",
		"checkpoint_progress_state: checkpoint-pending",
		"blocks_until_checkpoint: 82",
		"persisted_checkpoint_block: 100",
		"blocks_since_checkpoint: 18",
		"persisted_checkpoint_state: persisted-checkpoint-behind",
		"reorg_checkpoint_state: reorg-reconciled",
		"reorg_checkpoint_block: 200",
		"poll_activity_state: active",
		"active_consumers: 3",
		"consumer_lag: 8",
		"consumer_offset: 144",
		"consumer_progress_state: lagging",
	}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("expected %v, got %v", want, parts)
	}
}

func TestAppendRolloutExecutionProgressPostureReason(t *testing.T) {
	t.Parallel()
	parts := []string{}

	AppendRolloutExecutionProgressPostureReason(&parts, RolloutExecutionProgress{
		Poll: &RolloutPollProgressSnapshot{
			PollCount:                2,
			ProcessedBlock:           118,
			PersistedCheckpointState: "persisted-checkpoint-behind",
			ActivityState:            "active",
		},
		Consumer: &RolloutConsumerProgressSnapshot{
			ActiveConsumers: 3,
			Lag:             8,
			CurrentOffset:   144,
			ProgressState:   "lagging",
		},
	})

	want := []string{
		"poll_progress_posture: poll-catchup",
		"consumer_progress_posture: consumer-backlog",
	}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("expected %v, got %v", want, parts)
	}
}

func TestAppendRolloutExecutionOperatorHintReason(t *testing.T) {
	t.Parallel()
	parts := []string{}

	AppendRolloutExecutionOperatorHintReason(&parts, RolloutExecutionOperatorHint{
		Poll:     "continue observing checkpoint catch-up",
		Consumer: "prioritize backlog drain",
	})

	want := []string{
		"poll_operator_hint: continue observing checkpoint catch-up",
		"consumer_operator_hint: prioritize backlog drain",
	}
	if !reflect.DeepEqual(parts, want) {
		t.Fatalf("expected %v, got %v", want, parts)
	}
}
