package service

import (
	"context"
	"math/big"

	"chainpulse/shared/types"

	"github.com/ethereum/go-ethereum/common"
)

// IndexerService interface defines the methods that the indexer service should implement
type IndexerService interface {
	StartIndexing(ctx context.Context, contractAddresses []common.Address) error
	ProcessHistoricalEvents(ctx context.Context, contractAddresses []common.Address, fromBlock, toBlock *big.Int) error
	GetEvents(filter *types.EventFilter) ([]types.IndexedEvent, error)
	GetEventByID(id uint) (*types.IndexedEvent, error)
	GetEventsByBlockRange(fromBlock, toBlock *big.Int) ([]types.IndexedEvent, error)
	GetLastProcessedBlock() (*big.Int, error)
	ResumeEvents(ctx context.Context, fromBlock, toBlock *big.Int) error
}