package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/rtcdance/chainpulse/pkg/application/bootstrap"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

func (m *monolithicPullerRuntime) configureDurablePuller(ctx context.Context, chainID string, puller *pullers.HTTPSJSONRPCPuller, runtime *appindexing.SharedRuntime, maxDepth uint64) error {
	store := bootstrap.BuildReorgStore(m.database)
	if store == nil {
		return fmt.Errorf("durable puller requires a chain-scoped reorg store")
	}
	chainDB := store.ChainDatabase(chainID)
	checkpoint, err := runtime.ChainCheckpoint(ctx, chainID)
	if err != nil {
		return fmt.Errorf("load durable pull position for %s: %w", chainID, err)
	}
	if checkpoint.BlockNumber > 0 {
		puller.SetLastBlockNumber(checkpoint.BlockNumber)
	}
	chainRuntime := m.reorgChains[chainID]
	puller.SetCheckpointValidator(func(ctx context.Context) error {
		checkpoint, err := runtime.ChainCheckpoint(ctx, chainID)
		if err != nil {
			return fmt.Errorf("read durable checkpoint: %w", err)
		}
		if checkpoint.BlockNumber == 0 {
			return nil
		}
		from, err := canonicalDivergence(ctx, checkpoint.BlockNumber, common.HexToHash(checkpoint.BlockHash), maxDepth,
			func(ctx context.Context, height uint64) (common.Hash, error) {
				block, err := chainDB.GetBlock(ctx, height)
				if err != nil || block == nil {
					return common.Hash{}, err
				}
				return block.Hash, nil
			}, puller.BlockHeader)
		if err != nil {
			chainRuntime.recordError(err)
			return err
		}
		if from > 0 {
			chainRuntime.recordDetection(from)
			if err := chainRuntime.handler.HandleReorg(ctx, from); err != nil {
				chainRuntime.recordError(err)
				return err
			}
			chainRuntime.recordHandled(from)
			puller.SetLastBlockNumber(from - 1)
		} else {
			// Partial range commits and delayed outbox notices can leave intake
			// behind durable progress. The database always owns the resume point.
			puller.SetLastBlockNumber(checkpoint.BlockNumber)
		}
		return nil
	})
	puller.SetCompletedRangeHandler(func(ctx context.Context, from, to uint64, events []blockchain.BlockchainEvent) (resultErr error) {
		defer func() {
			if resultErr != nil {
				chainRuntime.recordError(resultErr)
			} else {
				chainRuntime.clearError()
			}
		}()
		checkpoint, err := runtime.ChainCheckpoint(ctx, chainID)
		if err != nil {
			return fmt.Errorf("read range checkpoint: %w", err)
		}
		if checkpoint.BlockNumber+1 != from {
			return fmt.Errorf("range starts at %d instead of durable next block %d", from, checkpoint.BlockNumber+1)
		}
		return ingestCompleteRange(ctx, chainID, from, to, events, func(ctx context.Context, height uint64) (*types.Header, error) {
			header, err := puller.BlockHeader(ctx, height)
			if err == nil && header != nil && height == from && checkpoint.BlockNumber > 0 && header.ParentHash != common.HexToHash(checkpoint.BlockHash) {
				return nil, fmt.Errorf("range parent diverges from durable checkpoint")
			}
			return header, err
		}, runtime, func(header *types.Header) error {
			return store.CompleteReorgsThrough(ctx, chainID, header.Number.Uint64())
		})
	})
	return nil
}
