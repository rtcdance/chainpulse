package ports

import "context"

// IdempotencyInvalidator clears idempotency entries for a block range.
type IdempotencyInvalidator interface {
	InvalidateRange(fromBlock, toBlock uint64) int
}

// CheckpointStore persists indexing progress across restarts
type CheckpointStore interface {
	GetLastIndexedBlock(ctx context.Context, chainID string) (uint64, string, error)
	SaveLastIndexedBlock(ctx context.Context, chainID string, blockNumber uint64, blockHash string) error
	GetBlockHash(ctx context.Context, chainID string, blockNumber uint64) (string, error)
}
