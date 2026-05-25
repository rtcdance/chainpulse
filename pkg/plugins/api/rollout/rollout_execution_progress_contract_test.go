package rollout

import "testing"

func TestBuildRolloutExecutionProgress(t *testing.T) {
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

	if progress.Poll == nil {
		t.Fatalf("expected poll progress to be present")
	}
	if progress.Poll.PollCount != 2 ||
		progress.Poll.LastPollUnix != 1712345678 ||
		progress.Poll.ObservedBlock != 120 ||
		progress.Poll.ProcessedBlock != 118 ||
		progress.Poll.BlockGap != 2 ||
		progress.Poll.CheckpointState != "checkpoint-pending" ||
		progress.Poll.BlocksUntilCheckpoint != 82 ||
		progress.Poll.PersistedCheckpointBlock != 100 ||
		progress.Poll.BlocksSinceCheckpoint != 18 ||
		progress.Poll.PersistedCheckpointState != "persisted-checkpoint-behind" ||
		progress.Poll.ReorgCheckpointState != "reorg-reconciled" ||
		progress.Poll.ReorgCheckpointBlock != 200 ||
		progress.Poll.ActivityState != "active" {
		t.Fatalf("unexpected poll progress: %+v", *progress.Poll)
	}
	if progress.Consumer == nil {
		t.Fatalf("expected consumer progress to be present")
	}
	if progress.Consumer.ActiveConsumers != 3 || progress.Consumer.Lag != 8 || progress.Consumer.CurrentOffset != 144 || progress.Consumer.ProgressState != "lagging" {
		t.Fatalf("unexpected consumer progress: %+v", *progress.Consumer)
	}
}

func TestBuildRolloutExecutionProgressOmitsEmptySnapshots(t *testing.T) {
	t.Parallel()
	progress := BuildRolloutExecutionProgress(RolloutExecutionProgressInput{})

	if progress.Poll != nil {
		t.Fatalf("expected empty poll progress to be omitted, got %+v", *progress.Poll)
	}
	if progress.Consumer != nil {
		t.Fatalf("expected empty consumer progress to be omitted, got %+v", *progress.Consumer)
	}
}

func TestBuildRolloutExecutionProgressPosture(t *testing.T) {
	t.Parallel()
	posture := BuildRolloutExecutionProgressPosture(BuildRolloutExecutionProgress(RolloutExecutionProgressInput{
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
	}))

	if posture.Poll != "poll-catchup" {
		t.Fatalf("expected poll posture poll-catchup, got %q", posture.Poll)
	}
	if posture.Consumer != "consumer-backlog" {
		t.Fatalf("expected consumer posture consumer-backlog, got %q", posture.Consumer)
	}
}

func TestClassifyRolloutPollProgressPosture_ReorgRisk(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{
		ReorgCheckpointState: "reorg-risk",
	})
	if got != "poll-risk" {
		t.Fatalf("expected poll-risk, got %q", got)
	}
}

func TestClassifyRolloutPollProgressPosture_PollAdvancing(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{
		ActivityState:  "active",
		ProcessedBlock: 100,
	})
	if got != "poll-advancing" {
		t.Fatalf("expected poll-advancing, got %q", got)
	}
}

func TestClassifyRolloutPollProgressPosture_PollIdle(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{
		ActivityState: "no-polls-yet",
	})
	if got != "poll-idle" {
		t.Fatalf("expected poll-idle, got %q", got)
	}
}

func TestClassifyRolloutPollProgressPosture_PollStalled(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{
		ActivityState: "stale",
	})
	if got != "poll-stalled" {
		t.Fatalf("expected poll-stalled, got %q", got)
	}
}

func TestClassifyRolloutPollProgressPosture_PollMonitoring(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{
		ActivityState: "unknown-state",
	})
	if got != "poll-monitoring" {
		t.Fatalf("expected poll-monitoring, got %q", got)
	}
}

func TestClassifyRolloutPollProgressPosture_Empty(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{})
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestClassifyRolloutPollProgressPosture_CheckpointDue(t *testing.T) {
	t.Parallel()
	got := classifyRolloutPollProgressPosture(RolloutPollProgressSnapshot{
		CheckpointState: "checkpoint-due",
	})
	if got != "poll-catchup" {
		t.Fatalf("expected poll-catchup, got %q", got)
	}
}

func TestClassifyRolloutConsumerProgressPosture_Idle(t *testing.T) {
	t.Parallel()
	got := classifyRolloutConsumerProgressPosture(RolloutConsumerProgressSnapshot{
		ProgressState: "idle",
	})
	if got != "consumer-idle" {
		t.Fatalf("expected consumer-idle, got %q", got)
	}
}

func TestClassifyRolloutConsumerProgressPosture_Advancing(t *testing.T) {
	t.Parallel()
	got := classifyRolloutConsumerProgressPosture(RolloutConsumerProgressSnapshot{
		ProgressState: "active",
		CurrentOffset: 10,
	})
	if got != "consumer-advancing" {
		t.Fatalf("expected consumer-advancing, got %q", got)
	}
}

func TestClassifyRolloutConsumerProgressPosture_ActiveNoOffset(t *testing.T) {
	t.Parallel()
	got := classifyRolloutConsumerProgressPosture(RolloutConsumerProgressSnapshot{
		ProgressState: "active",
		CurrentOffset: 0,
	})
	if got != "consumer-active" {
		t.Fatalf("expected consumer-active, got %q", got)
	}
}

func TestClassifyRolloutConsumerProgressPosture_Watch(t *testing.T) {
	t.Parallel()
	got := classifyRolloutConsumerProgressPosture(RolloutConsumerProgressSnapshot{
		ProgressState: "monitoring",
		Lag:           5,
	})
	if got != "consumer-watch" {
		t.Fatalf("expected consumer-watch, got %q", got)
	}
}

func TestClassifyRolloutConsumerProgressPosture_MonitoringNoLag(t *testing.T) {
	t.Parallel()
	got := classifyRolloutConsumerProgressPosture(RolloutConsumerProgressSnapshot{
		ProgressState: "monitoring",
		Lag:           0,
	})
	if got != "consumer-monitoring" {
		t.Fatalf("expected consumer-monitoring, got %q", got)
	}
}

func TestClassifyRolloutConsumerProgressPosture_Empty(t *testing.T) {
	t.Parallel()
	got := classifyRolloutConsumerProgressPosture(RolloutConsumerProgressSnapshot{})
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
