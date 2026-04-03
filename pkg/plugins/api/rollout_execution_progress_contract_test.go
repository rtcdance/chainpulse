package api

import "testing"

func TestBuildRolloutExecutionProgress(t *testing.T) {
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
	progress := BuildRolloutExecutionProgress(RolloutExecutionProgressInput{})

	if progress.Poll != nil {
		t.Fatalf("expected empty poll progress to be omitted, got %+v", *progress.Poll)
	}
	if progress.Consumer != nil {
		t.Fatalf("expected empty consumer progress to be omitted, got %+v", *progress.Consumer)
	}
}

func TestBuildRolloutExecutionProgressPosture(t *testing.T) {
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
