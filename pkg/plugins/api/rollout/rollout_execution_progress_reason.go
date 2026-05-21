package rollout

import "strconv"

// AppendRolloutPollProgressReason appends poll-loop progress details to a
// rollout reason parts slice.
func AppendRolloutPollProgressReason(parts *[]string, snapshot RolloutPollProgressSnapshot) {
	if parts == nil {
		return
	}
	if snapshot.PollCount > 0 {
		*parts = append(*parts, "poll_count: "+strconv.FormatInt(snapshot.PollCount, 10))
	}
	if snapshot.LastPollUnix > 0 {
		*parts = append(*parts, "last_poll_unix: "+strconv.FormatInt(snapshot.LastPollUnix, 10))
	}
	if snapshot.ObservedBlock > 0 {
		*parts = append(*parts, "observed_block: "+strconv.FormatInt(snapshot.ObservedBlock, 10))
	}
	if snapshot.ProcessedBlock > 0 {
		*parts = append(*parts, "processed_block: "+strconv.FormatInt(snapshot.ProcessedBlock, 10))
	}
	if snapshot.BlockGap > 0 {
		*parts = append(*parts, "block_gap: "+strconv.FormatInt(snapshot.BlockGap, 10))
	}
	if snapshot.CheckpointState != "" {
		*parts = append(*parts, "checkpoint_progress_state: "+snapshot.CheckpointState)
	}
	if snapshot.BlocksUntilCheckpoint > 0 {
		*parts = append(*parts, "blocks_until_checkpoint: "+strconv.FormatInt(snapshot.BlocksUntilCheckpoint, 10))
	}
	if snapshot.PersistedCheckpointBlock > 0 {
		*parts = append(*parts, "persisted_checkpoint_block: "+strconv.FormatInt(snapshot.PersistedCheckpointBlock, 10))
	}
	if snapshot.BlocksSinceCheckpoint > 0 {
		*parts = append(*parts, "blocks_since_checkpoint: "+strconv.FormatInt(snapshot.BlocksSinceCheckpoint, 10))
	}
	if snapshot.PersistedCheckpointState != "" {
		*parts = append(*parts, "persisted_checkpoint_state: "+snapshot.PersistedCheckpointState)
	}
	if snapshot.ReorgCheckpointState != "" {
		*parts = append(*parts, "reorg_checkpoint_state: "+snapshot.ReorgCheckpointState)
	}
	if snapshot.ReorgCheckpointBlock > 0 {
		*parts = append(*parts, "reorg_checkpoint_block: "+strconv.FormatInt(snapshot.ReorgCheckpointBlock, 10))
	}
	if snapshot.ActivityState != "" {
		*parts = append(*parts, "poll_activity_state: "+snapshot.ActivityState)
	}
}

// AppendRolloutConsumerProgressReason appends consumer progress details to a
// rollout reason parts slice.
func AppendRolloutConsumerProgressReason(parts *[]string, snapshot RolloutConsumerProgressSnapshot) {
	if parts == nil {
		return
	}
	if snapshot.ActiveConsumers > 0 {
		*parts = append(*parts, "active_consumers: "+strconv.FormatInt(snapshot.ActiveConsumers, 10))
	}
	if snapshot.Lag > 0 {
		*parts = append(*parts, "consumer_lag: "+strconv.FormatInt(snapshot.Lag, 10))
	}
	if snapshot.CurrentOffset > 0 {
		*parts = append(*parts, "consumer_offset: "+strconv.FormatInt(snapshot.CurrentOffset, 10))
	}
	if snapshot.ProgressState != "" {
		*parts = append(*parts, "consumer_progress_state: "+snapshot.ProgressState)
	}
}

// AppendRolloutExecutionProgressReason appends whichever lightweight execution
// progress snapshots are present to a rollout reason parts slice.
func AppendRolloutExecutionProgressReason(parts *[]string, progress RolloutExecutionProgress) {
	if parts == nil {
		return
	}
	if progress.Poll != nil {
		AppendRolloutPollProgressReason(parts, *progress.Poll)
	}
	if progress.Consumer != nil {
		AppendRolloutConsumerProgressReason(parts, *progress.Consumer)
	}
}

// AppendRolloutExecutionProgressPostureReason appends compact posture hints
// derived from the shared execution-progress facade.
func AppendRolloutExecutionProgressPostureReason(parts *[]string, progress RolloutExecutionProgress) {
	if parts == nil {
		return
	}

	posture := BuildRolloutExecutionProgressPosture(progress)
	if posture.Poll != "" {
		*parts = append(*parts, "poll_progress_posture: "+posture.Poll)
	}
	if posture.Consumer != "" {
		*parts = append(*parts, "consumer_progress_posture: "+posture.Consumer)
	}
}

// AppendRolloutExecutionOperatorHintReason appends compact operator-facing
// hints derived by services on top of execution progress posture.
func AppendRolloutExecutionOperatorHintReason(parts *[]string, hint RolloutExecutionOperatorHint) {
	if parts == nil {
		return
	}
	if hint.Poll != "" {
		*parts = append(*parts, "poll_operator_hint: "+hint.Poll)
	}
	if hint.Consumer != "" {
		*parts = append(*parts, "consumer_operator_hint: "+hint.Consumer)
	}
}
