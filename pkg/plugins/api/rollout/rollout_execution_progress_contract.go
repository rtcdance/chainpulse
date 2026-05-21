package rollout

// RolloutPollProgressSnapshot captures lightweight execution progress for
// poll-loop driven services such as the puller.
type RolloutPollProgressSnapshot struct {
	PollCount                int64
	LastPollUnix             int64
	ObservedBlock            int64
	ProcessedBlock           int64
	BlockGap                 int64
	CheckpointState          string
	BlocksUntilCheckpoint    int64
	PersistedCheckpointBlock int64
	BlocksSinceCheckpoint    int64
	PersistedCheckpointState string
	ReorgCheckpointState     string
	ReorgCheckpointBlock     int64
	ActivityState            string
}

// RolloutConsumerProgressSnapshot captures lightweight execution progress for
// consumer-driven services such as the event processor.
type RolloutConsumerProgressSnapshot struct {
	ActiveConsumers int64
	Lag             int64
	CurrentOffset   int64
	ProgressState   string
}

// RolloutExecutionProgress groups the lightweight execution progress snapshots
// that rollout producers may want to append to rollout reason details.
type RolloutExecutionProgress struct {
	Poll     *RolloutPollProgressSnapshot
	Consumer *RolloutConsumerProgressSnapshot
}

// RolloutExecutionProgressInput captures optional execution-progress snapshot
// inputs before they are normalized into the shared facade.
type RolloutExecutionProgressInput struct {
	Poll     RolloutPollProgressSnapshot
	Consumer RolloutConsumerProgressSnapshot
}

// RolloutExecutionProgressPosture captures compact operator-readable posture
// conclusions derived from the shared execution-progress facade.
type RolloutExecutionProgressPosture struct {
	Poll     string
	Consumer string
}

// RolloutExecutionOperatorHint captures compact operator-facing next-step
// hints that rollout producers may derive from execution progress posture.
type RolloutExecutionOperatorHint struct {
	Poll     string
	Consumer string
}

// BuildRolloutExecutionProgress normalizes optional poll and consumer progress
// snapshots into the shared execution-progress facade.
func BuildRolloutExecutionProgress(input RolloutExecutionProgressInput) RolloutExecutionProgress {
	progress := RolloutExecutionProgress{}
	if rolloutPollProgressSnapshotPresent(input.Poll) {
		poll := input.Poll
		progress.Poll = &poll
	}
	if rolloutConsumerProgressSnapshotPresent(input.Consumer) {
		consumer := input.Consumer
		progress.Consumer = &consumer
	}
	return progress
}

// BuildRolloutExecutionProgressPosture derives compact operator-readable
// posture hints from the shared execution-progress facade.
func BuildRolloutExecutionProgressPosture(progress RolloutExecutionProgress) RolloutExecutionProgressPosture {
	posture := RolloutExecutionProgressPosture{}
	if progress.Poll != nil {
		posture.Poll = classifyRolloutPollProgressPosture(*progress.Poll)
	}
	if progress.Consumer != nil {
		posture.Consumer = classifyRolloutConsumerProgressPosture(*progress.Consumer)
	}
	return posture
}

func rolloutPollProgressSnapshotPresent(snapshot RolloutPollProgressSnapshot) bool {
	return snapshot.PollCount > 0 ||
		snapshot.LastPollUnix > 0 ||
		snapshot.ObservedBlock > 0 ||
		snapshot.ProcessedBlock > 0 ||
		snapshot.BlockGap > 0 ||
		snapshot.CheckpointState != "" ||
		snapshot.BlocksUntilCheckpoint > 0 ||
		snapshot.PersistedCheckpointBlock > 0 ||
		snapshot.BlocksSinceCheckpoint > 0 ||
		snapshot.PersistedCheckpointState != "" ||
		snapshot.ReorgCheckpointState != "" ||
		snapshot.ReorgCheckpointBlock > 0 ||
		snapshot.ActivityState != ""
}

func rolloutConsumerProgressSnapshotPresent(snapshot RolloutConsumerProgressSnapshot) bool {
	return snapshot.ActiveConsumers > 0 || snapshot.Lag > 0 || snapshot.CurrentOffset > 0 || snapshot.ProgressState != ""
}

func classifyRolloutPollProgressPosture(snapshot RolloutPollProgressSnapshot) string {
	switch {
	case snapshot.ReorgCheckpointState == "reorg-risk":
		return "poll-risk"
	case snapshot.PersistedCheckpointState == "persisted-checkpoint-behind" || snapshot.CheckpointState == "checkpoint-due":
		return "poll-catchup"
	case snapshot.ActivityState == "active" && (snapshot.ProcessedBlock > 0 || snapshot.PollCount > 0):
		return "poll-advancing"
	case snapshot.ActivityState == "no-polls-yet":
		return "poll-idle"
	case snapshot.ActivityState == "stale":
		return "poll-stalled"
	case snapshot.ActivityState != "":
		return "poll-monitoring"
	default:
		return ""
	}
}

func classifyRolloutConsumerProgressPosture(snapshot RolloutConsumerProgressSnapshot) string {
	switch snapshot.ProgressState {
	case "lagging":
		return "consumer-backlog"
	case "idle":
		return "consumer-idle"
	case "active":
		if snapshot.CurrentOffset > 0 {
			return "consumer-advancing"
		}
		return "consumer-active"
	case "monitoring":
		if snapshot.Lag > 0 {
			return "consumer-watch"
		}
		return "consumer-monitoring"
	default:
		return ""
	}
}
