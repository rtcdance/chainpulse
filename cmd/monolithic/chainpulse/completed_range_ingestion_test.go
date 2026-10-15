package main

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

type completedRangeRecorder struct {
	calls []uint64
	sizes []int
	fail  uint64
}

func (r *completedRangeRecorder) ProcessCompleteBlock(_ context.Context, block appindexing.CompletedBlock) error {
	if err := block.Normalize(); err != nil {
		return err
	}
	return r.commit(block.Header.Number.Uint64(), len(block.Events))
}

func (r *completedRangeRecorder) commit(height uint64, size int) error {
	if height == r.fail {
		return errors.New("database commit failed")
	}
	r.calls = append(r.calls, height)
	r.sizes = append(r.sizes, size)
	return nil
}

func TestCompletedRangeCommitsWholeLogGroupsAndEmptyBlocks(t *testing.T) {
	t.Parallel()
	h1 := &types.Header{Number: big.NewInt(1), Time: 1700000001}
	h2 := &types.Header{Number: big.NewInt(2), ParentHash: h1.Hash(), Time: 1700000002}
	h3 := &types.Header{Number: big.NewInt(3), ParentHash: h2.Hash(), Time: 1700000003}
	for _, fail := range []uint64{0, 2} {
		t.Run(big.NewInt(int64(fail)).String(), func(t *testing.T) {
			t.Parallel()
			r := &completedRangeRecorder{fail: fail}
			headers := map[uint64]*types.Header{1: h1, 2: h2, 3: h3}
			events := []blockchain.BlockchainEvent{
				{ChainID: "ethereum", BlockNumber: 2, BlockHash: h2.Hash(), LogIndex: 0},
				{ChainID: "ethereum", BlockNumber: 2, BlockHash: h2.Hash(), LogIndex: 1},
			}
			err := ingestCompleteRange(context.Background(), "ethereum", 1, 3, events, func(_ context.Context, height uint64) (*types.Header, error) { return headers[height], nil }, r, nil)
			if fail == 0 {
				require.NoError(t, err)
				require.Equal(t, []uint64{1, 2, 3}, r.calls)
				require.Equal(t, []int{0, 2, 0}, r.sizes)
			} else {
				require.Error(t, err)
				require.Equal(t, []uint64{1}, r.calls, "nothing after a failed block may advance")
			}
		})
	}
}

func TestCompletedRangeRejectsWrongBranchBeforeWriting(t *testing.T) {
	t.Parallel()
	h := &types.Header{Number: big.NewInt(1), Time: 1700000001}
	r := &completedRangeRecorder{}
	err := ingestCompleteRange(context.Background(), "ethereum", 1, 1, []blockchain.BlockchainEvent{{ChainID: "polygon", BlockNumber: 1, BlockHash: h.Hash()}}, func(context.Context, uint64) (*types.Header, error) { return h, nil }, r, nil)
	require.Error(t, err)
	require.Empty(t, r.calls)
}
