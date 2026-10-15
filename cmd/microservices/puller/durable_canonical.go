package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/plugins/pullers"
)

// reconcileDurableIntake runs before the no-new-block fast path. The database
// owns restart position; the published tip only tracks uncommitted Kafka work.
func (r *pullerExecutionRuntime) reconcileDurableIntake(ctx context.Context, source *pullers.MultiChainDataPuller, chain string, maxDepth uint64) error {
	if r.durableStore == nil {
		return fmt.Errorf("durable puller requires canonical PostgreSQL supervision")
	}
	height, hash, err := r.durableStore.GetCheckpointForChain(ctx, chain)
	if err != nil {
		return fmt.Errorf("read durable intake checkpoint: %w", err)
	}
	readHeader := func(ctx context.Context, height uint64) (*types.Header, error) {
		return source.BlockHeaderFromChain(ctx, chain, height)
	}
	rewound := false
	if height > 0 {
		from, err := appindexing.CanonicalDivergence(ctx, height, common.HexToHash(hash), maxDepth,
			func(ctx context.Context, height uint64) (common.Hash, error) {
				block, err := r.durableStore.ChainView(chain).GetBlock(ctx, height)
				if err != nil || block == nil {
					return common.Hash{}, err
				}
				return block.Hash, nil
			}, readHeader)
		if err != nil {
			return fmt.Errorf("inspect durable chain %s: %w", chain, err)
		}
		if from > 0 {
			if _, err := r.durableStore.CommitReorgAtCheckpoint(ctx, chain, from, height, hash); err != nil {
				return err
			}
			r.logger.Warn("Rewound native durable chain", "chain_id", chain, "from_block", from, "to_block", height)
			r.metrics.RecordCounter("puller_canonical_reorgs", 1, map[string]string{"chain_id": chain})
			height, rewound = from-1, true
		}
	}
	r.mu.RLock()
	initialized, tip := r.durableInitialized[chain], r.publishedTips[chain]
	r.mu.RUnlock()
	reset := !initialized || rewound
	if !reset && tip != nil {
		canonical, err := readHeader(ctx, tip.Number.Uint64())
		if err != nil {
			return fmt.Errorf("inspect published intake tip: %w", err)
		}
		reset = canonical.Hash() != tip.Hash()
	}
	if reset {
		if err := source.SetLastProcessedBlock(chain, height); err != nil {
			return err
		}
		r.mu.Lock()
		r.durableInitialized[chain] = true
		delete(r.publishedTips, chain)
		r.mu.Unlock()
	}
	return nil
}
