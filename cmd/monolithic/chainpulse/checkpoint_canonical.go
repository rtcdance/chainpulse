package main

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	appindexing "github.com/rtcdance/chainpulse/pkg/application/indexing"
)

// canonicalDivergence checks durable history against RPC, including after a
// restart or a same-height branch replacement. A rollback requires a known
// matching ancestor, and never exceeds the configured recovery window.
func canonicalDivergence(ctx context.Context, tip uint64, tipHash common.Hash, maxDepth uint64, stored func(context.Context, uint64) (common.Hash, error), header func(context.Context, uint64) (*types.Header, error)) (uint64, error) {
	return appindexing.CanonicalDivergence(ctx, tip, tipHash, maxDepth, stored, header)
}
