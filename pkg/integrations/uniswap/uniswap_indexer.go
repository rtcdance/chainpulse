package uniswap

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/rtcdance/chainpulse/pkg/blockchain"
	"github.com/rtcdance/chainpulse/pkg/core"
	"github.com/rtcdance/chainpulse/pkg/core/batch"
	"github.com/rtcdance/chainpulse/pkg/logkeys"

	"github.com/rtcdance/chainpulse/pkg/services/decoder"
)

// SwapEvent represents a decoded Uniswap swap event
type SwapEvent struct {
	TransactionHash common.Hash
	BlockNumber     uint64
	BlockTimestamp  int64
	LogIndex        uint64
	Pool            common.Address
	Sender          common.Address
	Recipient       common.Address
	Amount0In       *big.Int
	Amount1In       *big.Int
	Amount0Out      *big.Int
	Amount1Out      *big.Int
	SqrtPriceX96    *big.Int
	Liquidity       *big.Int
	Tick            int32
	Token0          common.Address
	Token1          common.Address
	Fee             uint32
	DecodedData     map[string]any
}

// SwapHistory represents historical swap data
type SwapHistory struct {
	Swaps         []*SwapEvent
	TotalVolume0  *big.Int
	TotalVolume1  *big.Int
	AveragePrice  *big.Float
	SwapCount     int64
	FirstSwapTime int64
	LastSwapTime  int64
}

// UniswapIndexer indexes Uniswap V3 events
type UniswapIndexer struct {
	database        core.DatabasePlugin
	cache           core.CachePlugin
	logger          core.Logger
	eventDecoder    *decoder.EventDecoder
	contractManager *decoder.ContractManager
	mu              sync.RWMutex
	poolMetadata    map[common.Address]*PoolMetadata
	swapEventCache  map[string]*SwapEvent
}

// PoolMetadata stores metadata about a Uniswap pool
type PoolMetadata struct {
	Address      common.Address
	Token0       common.Address
	Token1       common.Address
	Fee          uint32
	Liquidity    *big.Int
	SqrtPriceX96 *big.Int
	Tick         int32
	LastUpdated  time.Time
}

// NewUniswapIndexer creates a new Uniswap indexer
func NewUniswapIndexer(
	database core.DatabasePlugin,
	cache core.CachePlugin,
	logger core.Logger,
	eventDecoder *decoder.EventDecoder,
	contractManager *decoder.ContractManager,
) *UniswapIndexer {
	return &UniswapIndexer{
		database:        database,
		cache:           cache,
		logger:          logger,
		eventDecoder:    eventDecoder,
		contractManager: contractManager,
		poolMetadata:    make(map[common.Address]*PoolMetadata),
		swapEventCache:  make(map[string]*SwapEvent),
	}
}

// IndexSwapEvents indexes Uniswap swap events
func (ui *UniswapIndexer) IndexSwapEvents(
	ctx context.Context,
	events []*blockchain.BlockchainEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	ui.logger.Debug("indexing swap events", logkeys.LogKeyCount, len(events))

	return batch.Index(ctx, events, ui.indexSwapEvent)
}

// indexSwapEvent indexes a single swap event
func (ui *UniswapIndexer) indexSwapEvent(
	_ context.Context,
	event *blockchain.BlockchainEvent,
) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// Decode event data
	swapEvent, err := ui.decodeSwapEvent(event)
	if err != nil {
		return err
	}

	// Validate swap event
	if err := ui.validateSwapEvent(swapEvent); err != nil {
		return err
	}

	// Store in cache
	cacheKey := fmt.Sprintf("swap:%s:%d:%d", swapEvent.TransactionHash.Hex(), swapEvent.BlockNumber, swapEvent.LogIndex)
	ui.mu.Lock()
	ui.swapEventCache[cacheKey] = swapEvent
	ui.mu.Unlock()

	// Update pool metadata
	ui.updatePoolMetadata(swapEvent)

	ui.logger.Debug("indexed swap event",
		logkeys.LogKeyPool, swapEvent.Pool.Hex(),
		logkeys.LogKeySender, swapEvent.Sender.Hex(),
		logkeys.LogKeyRecipient, swapEvent.Recipient.Hex(),
		logkeys.LogKeySwapAmount0In, swapEvent.Amount0In.String(),
		logkeys.LogKeySwapAmount1In, swapEvent.Amount1In.String())

	return nil
}

// decodeSwapEvent decodes a raw event into a SwapEvent
func (ui *UniswapIndexer) decodeSwapEvent(event *blockchain.BlockchainEvent) (*SwapEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	swapEvent := &SwapEvent{
		TransactionHash: event.TransactionHash,
		BlockNumber:     event.BlockNumber,
		BlockTimestamp:  event.BlockTimestamp,
		LogIndex:        event.LogIndex,
		Pool:            event.ContractAddress,
		DecodedData:     event.DecodedData,
	}

	// Extract decoded data
	if event.DecodedData != nil {
		// Sender
		if sender, ok := event.DecodedData["sender"].(common.Address); ok {
			swapEvent.Sender = sender
		}

		// Recipient
		if recipient, ok := event.DecodedData["recipient"].(common.Address); ok {
			swapEvent.Recipient = recipient
		}

		// Amount0In
		if amount0In, ok := event.DecodedData["amount0In"].(*big.Int); ok {
			swapEvent.Amount0In = amount0In
		}

		// Amount1In
		if amount1In, ok := event.DecodedData["amount1In"].(*big.Int); ok {
			swapEvent.Amount1In = amount1In
		}

		// Amount0Out
		if amount0Out, ok := event.DecodedData["amount0Out"].(*big.Int); ok {
			swapEvent.Amount0Out = amount0Out
		}

		// Amount1Out
		if amount1Out, ok := event.DecodedData["amount1Out"].(*big.Int); ok {
			swapEvent.Amount1Out = amount1Out
		}

		// SqrtPriceX96
		if sqrtPrice, ok := event.DecodedData["sqrtPriceX96"].(*big.Int); ok {
			swapEvent.SqrtPriceX96 = sqrtPrice
		}

		// Liquidity
		if liquidity, ok := event.DecodedData["liquidity"].(*big.Int); ok {
			swapEvent.Liquidity = liquidity
		}

		// Tick
		if tick, ok := event.DecodedData["tick"].(int32); ok {
			swapEvent.Tick = tick
		}
	}

	return swapEvent, nil
}

// validateSwapEvent validates a swap event
func (ui *UniswapIndexer) validateSwapEvent(swapEvent *SwapEvent) error {
	if swapEvent == nil {
		return fmt.Errorf("swap event is nil")
	}

	if swapEvent.Pool == (common.Address{}) {
		return fmt.Errorf("pool address is empty")
	}

	if swapEvent.Amount0In == nil || swapEvent.Amount1In == nil {
		return fmt.Errorf("amount fields are nil")
	}

	if swapEvent.Amount0Out == nil || swapEvent.Amount1Out == nil {
		return fmt.Errorf("amount out fields are nil")
	}

	// Validate that at least one input and one output exists
	if (swapEvent.Amount0In.Sign() == 0 && swapEvent.Amount1In.Sign() == 0) ||
		(swapEvent.Amount0Out.Sign() == 0 && swapEvent.Amount1Out.Sign() == 0) {
		return fmt.Errorf("invalid swap amounts")
	}

	return nil
}

// updatePoolMetadata updates pool metadata from swap event
func (ui *UniswapIndexer) updatePoolMetadata(swapEvent *SwapEvent) {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	metadata, exists := ui.poolMetadata[swapEvent.Pool]
	if !exists {
		metadata = &PoolMetadata{
			Address: swapEvent.Pool,
		}
	}

	metadata.SqrtPriceX96 = swapEvent.SqrtPriceX96
	metadata.Liquidity = swapEvent.Liquidity
	metadata.Tick = swapEvent.Tick
	metadata.LastUpdated = time.Now()

	ui.poolMetadata[swapEvent.Pool] = metadata
}

// GetSwapHistory retrieves swap history for a pool
func (ui *UniswapIndexer) GetSwapHistory(
	ctx context.Context,
	pool common.Address,
	fromBlock, toBlock uint64,
) (*SwapHistory, error) {
	if pool == (common.Address{}) {
		return nil, fmt.Errorf("pool address is empty")
	}

	if fromBlock > toBlock && toBlock != 0 {
		return nil, fmt.Errorf("from_block must be <= to_block")
	}

	ui.logger.Debug("getting swap history",
		logkeys.LogKeyPool, pool.Hex(),
		logkeys.LogKeyFromBlock, fromBlock,
		logkeys.LogKeyToBlock, toBlock)

	// Create filter for swap events
	filter := &core.EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{pool},
		FromBlock:       fromBlock,
		ToBlock:         toBlock,
		Limit:           10000,
	}

	// Query events from database
	events, err := ui.database.QueryEvents(ctx, filter)
	if err != nil {
		ui.logger.Error("failed to query swap events",
			logkeys.LogKeyError, err.Error(),
			logkeys.LogKeyPool, pool.Hex())
		return nil, err
	}

	// Decode events
	swaps := make([]*SwapEvent, 0, len(events))
	totalVolume0 := big.NewInt(0)
	totalVolume1 := big.NewInt(0)
	var firstSwapTime, lastSwapTime int64

	for i, eventInterface := range events {
		event, ok := eventInterface.(*blockchain.BlockchainEvent)
		if !ok {
			ui.logger.Warn("failed to cast event to BlockchainEvent", "index", i)
			continue
		}

		swapEvent, err := ui.decodeSwapEvent(event)
		if err != nil {
			ui.logger.Warn("failed to decode swap event",
				logkeys.LogKeyError, err.Error(),
				logkeys.LogKeyEventID, event.ID)
			continue
		}

		swaps = append(swaps, swapEvent)

		// Accumulate volumes
		if swapEvent.Amount0In != nil {
			totalVolume0.Add(totalVolume0, swapEvent.Amount0In)
		}
		if swapEvent.Amount1In != nil {
			totalVolume1.Add(totalVolume1, swapEvent.Amount1In)
		}

		// Track time range
		if i == 0 {
			firstSwapTime = swapEvent.BlockTimestamp
		}
		lastSwapTime = swapEvent.BlockTimestamp
	}

	// Calculate average price
	avgPrice := big.NewFloat(0)
	if totalVolume0.Sign() > 0 && totalVolume1.Sign() > 0 {
		volume0Float := new(big.Float).SetInt(totalVolume0)
		volume1Float := new(big.Float).SetInt(totalVolume1)
		avgPrice.Quo(volume1Float, volume0Float)
	}

	history := &SwapHistory{
		Swaps:         swaps,
		TotalVolume0:  totalVolume0,
		TotalVolume1:  totalVolume1,
		AveragePrice:  avgPrice,
		SwapCount:     int64(len(swaps)),
		FirstSwapTime: firstSwapTime,
		LastSwapTime:  lastSwapTime,
	}

	ui.logger.Debug("retrieved swap history",
		logkeys.LogKeyPool, pool.Hex(),
		"swap_count", len(swaps),
		"volume0", totalVolume0.String(),
		"volume1", totalVolume1.String())

	return history, nil
}

// GetPoolMetadata retrieves metadata for a pool
func (ui *UniswapIndexer) GetPoolMetadata(pool common.Address) *PoolMetadata {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	return ui.poolMetadata[pool]
}

// GetAllPoolMetadata retrieves metadata for all pools
func (ui *UniswapIndexer) GetAllPoolMetadata() map[common.Address]*PoolMetadata {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	result := make(map[common.Address]*PoolMetadata)
	for addr, metadata := range ui.poolMetadata {
		result[addr] = metadata
	}

	return result
}

// ClearCache clears the swap event cache
func (ui *UniswapIndexer) ClearCache() {
	ui.mu.Lock()
	defer ui.mu.Unlock()

	ui.swapEventCache = make(map[string]*SwapEvent)
}

// GetCacheStats returns cache statistics
func (ui *UniswapIndexer) GetCacheStats() map[string]any {
	ui.mu.RLock()
	defer ui.mu.RUnlock()

	return map[string]any{
		"cached_swaps": len(ui.swapEventCache),
		"pools":        len(ui.poolMetadata),
	}
}
