package main

import (
	"context"
	"math"

	"chainpulse/pkg/core"
	"chainpulse/pkg/plugins/pullers"
)

func capturePullerBlockProgress(
	ctx context.Context,
	puller *pullers.MultiChainDataPuller,
	checkpointSource pullerCheckpointSource,
	checkpointInterval int,
	logger core.Logger,
	progress *pullerLoopRuntimeProgress,
) {
	if puller == nil || progress == nil {
		return
	}

	highestLatest, err := puller.GetHighestLatestBlock(ctx)
	if err != nil {
		if logger != nil {
			logger.Debug("Failed to capture latest puller block progress", "error", err.Error())
		}
		return
	}
	if highestLatest > 0 {
		progress.recordObservedBlock(safeUint64ToInt64(highestLatest))
	}

	if highestProcessed := puller.GetHighestProcessedBlock(); highestProcessed > 0 {
		progress.recordProcessedBlock(safeUint64ToInt64(highestProcessed))
	}

	if checkpointSource == nil || checkpointInterval <= 0 {
		return
	}
	for chainID, blockHeight := range puller.GetProcessedBlocksFromAllChains() {
		if err := checkpointSource.ObserveChainProgress(ctx, chainID, blockHeight); err != nil && logger != nil {
			logger.Debug("Failed to observe puller checkpoint progress", "chain_id", chainID, "error", err.Error())
		}
		if blockHeight == 0 || blockHeight%uint64(checkpointInterval) != 0 {
			continue
		}
		if err := checkpointSource.SaveCheckpoint(ctx, chainID, blockHeight); err != nil && logger != nil {
			logger.Debug("Failed to persist puller checkpoint progress", "chain_id", chainID, "error", err.Error())
		}
	}
}

func safeUint64ToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(value)
}
