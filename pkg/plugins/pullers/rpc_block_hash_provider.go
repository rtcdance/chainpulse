package pullers

import (
	"context"
	"fmt"

	"chainpulse/pkg/core"
	"github.com/ethereum/go-ethereum/common"
)

// RPCBlockHashProvider fetches canonical chain block hashes via eth_getBlockByNumber.
// This is the production provider for reorg detection — it compares locally-indexed
// hashes against the live canonical chain, which is the only reliable way to detect
// that a reorg has occurred.
type RPCBlockHashProvider struct {
	puller *HTTPSJSONRPCPuller
}

// NewRPCBlockHashProvider creates a new RPC-backed block hash provider.
func NewRPCBlockHashProvider(puller *HTTPSJSONRPCPuller) *RPCBlockHashProvider {
	return &RPCBlockHashProvider{puller: puller}
}

// GetBlockHash fetches the block hash for the given block number from the RPC node.
func (p *RPCBlockHashProvider) GetBlockHash(ctx context.Context, blockNumber uint64) (common.Hash, error) {
	header, err := p.puller.getBlockHeader(ctx, blockNumber)
	if err != nil {
		return common.Hash{}, fmt.Errorf("RPC getBlockHeader(%d): %w", blockNumber, err)
	}
	if header == nil {
		return common.Hash{}, nil
	}
	return common.HexToHash(header.Hash), nil
}

// Compile-time check
var _ core.BlockHashProvider = (*RPCBlockHashProvider)(nil)
