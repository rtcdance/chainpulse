package rollout

import "testing"

func TestValidateRolloutExecutionProgressReasonCoverage(t *testing.T) {
	t.Parallel()
	progress := BuildRolloutExecutionProgress(RolloutExecutionProgressInput{
		Poll: RolloutPollProgressSnapshot{
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
		Consumer: RolloutConsumerProgressSnapshot{
			ActiveConsumers: 3,
			Lag:             8,
			CurrentOffset:   144,
			ProgressState:   "lagging",
		},
	})

	reason := "enabled: kafka_ready; poll_count: 2; last_poll_unix: 1712345678; observed_block: 120; processed_block: 118; block_gap: 2; checkpoint_progress_state: checkpoint-pending; blocks_until_checkpoint: 82; persisted_checkpoint_block: 100; blocks_since_checkpoint: 18; persisted_checkpoint_state: persisted-checkpoint-behind; reorg_checkpoint_state: reorg-reconciled; reorg_checkpoint_block: 200; poll_activity_state: active; active_consumers: 3; consumer_lag: 8; consumer_offset: 144; consumer_progress_state: lagging"

	if err := ValidateRolloutExecutionProgressReasonCoverage(reason, progress); err != nil {
		t.Fatalf("expected reason coverage validation to succeed: %v", err)
	}
}

func TestValidateRolloutExecutionProgressPostureReasonCoverage(t *testing.T) {
	t.Parallel()
	reason := "poll_progress_posture: poll-catchup; consumer_progress_posture: consumer-backlog"
	progress := BuildRolloutExecutionProgress(RolloutExecutionProgressInput{
		Poll: RolloutPollProgressSnapshot{
			PollCount:                2,
			ProcessedBlock:           118,
			PersistedCheckpointState: "persisted-checkpoint-behind",
			ActivityState:            "active",
		},
		Consumer: RolloutConsumerProgressSnapshot{
			ActiveConsumers: 3,
			Lag:             8,
			CurrentOffset:   144,
			ProgressState:   "lagging",
		},
	})

	if err := ValidateRolloutExecutionProgressPostureReasonCoverage(reason, progress); err != nil {
		t.Fatalf("expected posture reason coverage validation to succeed: %v", err)
	}
}

func TestValidateRolloutExecutionOperatorHintReasonCoverage(t *testing.T) {
	t.Parallel()
	reason := "poll_operator_hint: continue observing checkpoint catch-up; consumer_operator_hint: prioritize backlog drain"
	hint := RolloutExecutionOperatorHint{
		Poll:     "continue observing checkpoint catch-up",
		Consumer: "prioritize backlog drain",
	}

	if err := ValidateRolloutExecutionOperatorHintReasonCoverage(reason, hint); err != nil {
		t.Fatalf("expected operator hint reason coverage validation to succeed: %v", err)
	}
}

func TestValidateRolloutExecutionProgressReasonCoverageFailsWhenMissingPart(t *testing.T) {
	t.Parallel()
	progress := BuildRolloutExecutionProgress(RolloutExecutionProgressInput{
		Consumer: RolloutConsumerProgressSnapshot{
			ActiveConsumers: 3,
			Lag:             8,
			CurrentOffset:   144,
			ProgressState:   "lagging",
		},
	})

	reason := "active_consumers: 3; consumer_progress_state: lagging"

	if err := ValidateRolloutExecutionProgressReasonCoverage(reason, progress); err == nil {
		t.Fatal("expected reason coverage validation to fail when a progress detail is missing")
	}
}
