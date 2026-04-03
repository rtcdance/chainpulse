package main

import (
	"time"

	"chainpulse/pkg/plugins/api"
)

func buildPullerPollProgressSnapshot(
	now time.Time,
	pollIntervalSeconds int,
	checkpointInterval int,
	checkpointSnapshot pullerCheckpointSourceSnapshot,
	progress *pullerLoopRuntimeProgress,
) api.RolloutPollProgressSnapshot {
	if progress == nil {
		checkpointState, blocksUntilCheckpoint := classifyPullerCheckpointProgress(0, checkpointInterval)
		persistedCheckpointState, blocksSinceCheckpoint := classifyPullerPersistedCheckpoint(0, checkpointSnapshot)
		reorgCheckpointState, reorgCheckpointBlock := classifyPullerCheckpointReorgRisk(checkpointSnapshot)
		return api.RolloutPollProgressSnapshot{
			CheckpointState:          checkpointState,
			BlocksUntilCheckpoint:    blocksUntilCheckpoint,
			PersistedCheckpointBlock: checkpointSnapshot.HighestCheckpointBlock,
			BlocksSinceCheckpoint:    blocksSinceCheckpoint,
			PersistedCheckpointState: persistedCheckpointState,
			ReorgCheckpointState:     reorgCheckpointState,
			ReorgCheckpointBlock:     reorgCheckpointBlock,
			ActivityState:            classifyPullerPollActivityState(now, pollIntervalSeconds, pullerLoopRuntimeProgressSnapshot{}),
		}
	}

	snapshot := progress.snapshot()
	blockGap := snapshot.ObservedBlock - snapshot.ProcessedBlock
	if blockGap < 0 {
		blockGap = 0
	}
	checkpointState, blocksUntilCheckpoint := classifyPullerCheckpointProgress(snapshot.ProcessedBlock, checkpointInterval)
	persistedCheckpointState, blocksSinceCheckpoint := classifyPullerPersistedCheckpoint(snapshot.ProcessedBlock, checkpointSnapshot)
	reorgCheckpointState, reorgCheckpointBlock := classifyPullerCheckpointReorgRisk(checkpointSnapshot)
	return api.RolloutPollProgressSnapshot{
		PollCount:                snapshot.PollCount,
		LastPollUnix:             snapshot.LastPollUnix,
		ObservedBlock:            snapshot.ObservedBlock,
		ProcessedBlock:           snapshot.ProcessedBlock,
		BlockGap:                 blockGap,
		CheckpointState:          checkpointState,
		BlocksUntilCheckpoint:    blocksUntilCheckpoint,
		PersistedCheckpointBlock: checkpointSnapshot.HighestCheckpointBlock,
		BlocksSinceCheckpoint:    blocksSinceCheckpoint,
		PersistedCheckpointState: persistedCheckpointState,
		ReorgCheckpointState:     reorgCheckpointState,
		ReorgCheckpointBlock:     reorgCheckpointBlock,
		ActivityState:            classifyPullerPollActivityState(now, pollIntervalSeconds, snapshot),
	}
}

func classifyPullerCheckpointProgress(processedBlock int64, checkpointInterval int) (string, int64) {
	if checkpointInterval <= 0 {
		return "", 0
	}
	if processedBlock <= 0 {
		return "checkpoint-uninitialized", 0
	}
	remainder := processedBlock % int64(checkpointInterval)
	if remainder == 0 {
		return "checkpoint-due", 0
	}
	return "checkpoint-pending", int64(checkpointInterval) - remainder
}
