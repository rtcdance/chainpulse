package pullers

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

// SetCheckpointValidator installs a canonical-history check for every polling
// cycle, even when the head has not advanced. Install before Start.
func (p *HTTPSJSONRPCPuller) SetCheckpointValidator(validate func(context.Context) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.validateCheckpoint = validate
}

func (p *HTTPSJSONRPCPuller) validateDurableCheckpoint(ctx context.Context) error {
	p.mu.RLock()
	validate := p.validateCheckpoint
	p.mu.RUnlock()
	if validate == nil {
		return nil
	}
	return validate(ctx)
}

// SetCompletedRangeHandler transfers durable block ownership before individual
// logs are published. Install it before Start; failure must not advance intake.
func (p *HTTPSJSONRPCPuller) SetCompletedRangeHandler(handler func(context.Context, uint64, uint64, []blockchain.BlockchainEvent) error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completedRange = handler
}

func (p *HTTPSJSONRPCPuller) commitPulledRange(ctx context.Context, from, to uint64, events []blockchain.BlockchainEvent) error {
	p.mu.RLock()
	handler := p.completedRange
	p.mu.RUnlock()
	if handler == nil {
		return nil
	}
	return handler(ctx, from, to, events)
}

func (p *HTTPSJSONRPCPuller) boundedPollTarget(target uint64) uint64 {
	last := p.GetLastBlockNumber()
	batch := p.GetConfig().BatchSize
	if batch <= 0 {
		batch = 100
	}
	if target > last && target-last > uint64(batch) {
		return last + uint64(batch)
	}
	return target
}

func (p *HTTPSJSONRPCPuller) BlockHeader(ctx context.Context, height uint64) (*types.Header, error) {
	p.mu.RLock()
	client := p.ethClient
	p.mu.RUnlock()
	if client == nil {
		return nil, fmt.Errorf("RPC client is not initialized")
	}
	return client.HeaderByNumber(ctx, new(big.Int).SetUint64(height))
}

func (p *HTTPSJSONRPCPuller) FinalizedHead(ctx context.Context) (uint64, error) {
	p.mu.RLock()
	client := p.ethClient
	p.mu.RUnlock()
	if client == nil {
		return 0, fmt.Errorf("RPC client is not initialized")
	}
	header, err := client.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return 0, fmt.Errorf("read finalized head: %w", err)
	}
	if header == nil || header.Number == nil {
		return 0, fmt.Errorf("finalized header is missing")
	}
	return header.Number.Uint64(), nil
}
