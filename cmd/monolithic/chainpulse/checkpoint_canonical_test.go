package main

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestCheckpointCanonicalRequiresKnownAncestor(t *testing.T) {
	t.Parallel()
	for _, depth := range []uint64{0, 1, 2, 12, 13} {
		t.Run(new(big.Int).SetUint64(depth).String(), func(t *testing.T) {
			t.Parallel()
			const tip = uint64(20)
			headers := make(map[uint64]*types.Header)
			stored := make(map[uint64]common.Hash)
			for height := uint64(0); height <= tip; height++ {
				h := &types.Header{Number: new(big.Int).SetUint64(height), Time: height + 1700000000}
				stored[height] = h.Hash()
				if height > tip-depth {
					h.Time++
				}
				headers[height] = h
			}
			from, err := canonicalDivergence(context.Background(), tip, stored[tip], 12,
				func(_ context.Context, height uint64) (common.Hash, error) { return stored[height], nil },
				func(_ context.Context, height uint64) (*types.Header, error) { return headers[height], nil })
			if depth > 12 {
				require.Error(t, err)
				require.Zero(t, from)
			} else {
				require.NoError(t, err)
				if depth == 0 {
					require.Zero(t, from)
				} else {
					require.Equal(t, tip-depth+1, from)
				}
			}
		})
	}
}

func TestCheckpointCanonicalUnknownAncestorStops(t *testing.T) {
	t.Parallel()
	_, err := canonicalDivergence(context.Background(), 20, common.HexToHash("0x1"), 12,
		func(context.Context, uint64) (common.Hash, error) { return common.Hash{}, nil },
		func(_ context.Context, height uint64) (*types.Header, error) {
			return &types.Header{Number: new(big.Int).SetUint64(height)}, nil
		})
	require.Error(t, err)
}
