package indexing

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CanonicalDivergence checks durable history against RPC, including after a
// restart or a same-height branch replacement. A rollback requires a known
// matching ancestor, and never exceeds the configured recovery window.
func CanonicalDivergence(ctx context.Context, tip uint64, tipHash common.Hash, maxDepth uint64, stored func(context.Context, uint64) (common.Hash, error), header func(context.Context, uint64) (*types.Header, error)) (uint64, error) {
	if tipHash == (common.Hash{}) || maxDepth == 0 {
		return 0, fmt.Errorf("checkpoint hash and recovery window are required")
	}
	lower := uint64(0)
	if tip > maxDepth {
		lower = tip - maxDepth
	}
	for height := tip; ; height-- {
		local := tipHash
		if height != tip {
			var err error
			local, err = stored(ctx, height)
			if err != nil {
				return 0, fmt.Errorf("read durable ancestor %d: %w", height, err)
			}
		}
		if local == (common.Hash{}) {
			return 0, fmt.Errorf("unknown durable ancestor at %d", height)
		}
		h, err := header(ctx, height)
		if err != nil {
			return 0, fmt.Errorf("read canonical ancestor %d: %w", height, err)
		}
		if h == nil || h.Number == nil || h.Number.Uint64() != height {
			return 0, fmt.Errorf("invalid canonical header at %d", height)
		}
		if local == h.Hash() {
			if height == tip {
				return 0, nil
			}
			return height + 1, nil
		}
		if height == lower {
			return 0, fmt.Errorf("canonical ancestor not found within %d blocks", maxDepth)
		}
	}
}
