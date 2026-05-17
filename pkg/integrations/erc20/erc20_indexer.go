package erc20

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"chainpulse/pkg/core"
	"chainpulse/pkg/services/decoder"
)

// TransferEvent represents a decoded ERC20 transfer event
type TransferEvent struct {
	TransactionHash common.Hash
	BlockNumber     uint64
	BlockTimestamp  int64
	LogIndex        uint64
	Token           common.Address
	From            common.Address
	To              common.Address
	Value           *big.Int
	DecodedData     map[string]any
}

// TokenBalance represents a token balance at a specific block
type TokenBalance struct {
	Token       common.Address
	Account     common.Address
	Balance     *big.Int
	BlockNumber uint64
	BlockTime   int64
}

// TransferHistory represents historical transfer data
type TransferHistory struct {
	Transfers     []*TransferEvent
	TotalIncoming *big.Int
	TotalOutgoing *big.Int
	NetChange     *big.Int
	TransferCount int64
	FirstTransfer int64
	LastTransfer  int64
}

// ERC20Indexer indexes ERC20 token events
type ERC20Indexer struct {
	database        core.DatabasePlugin
	cache           core.CachePlugin
	logger          core.Logger
	eventDecoder    *decoder.EventDecoder
	contractManager *decoder.ContractManager
	mu              sync.RWMutex
	tokenMetadata   map[common.Address]*TokenMetadata
	balances        map[string]*TokenBalance
	transferCache   map[string]*TransferEvent
}

// TokenMetadata stores metadata about an ERC20 token
type TokenMetadata struct {
	Address     common.Address
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	LastUpdated time.Time
}

// NewERC20Indexer creates a new ERC20 indexer
func NewERC20Indexer(
	database core.DatabasePlugin,
	cache core.CachePlugin,
	logger core.Logger,
	eventDecoder *decoder.EventDecoder,
	contractManager *decoder.ContractManager,
) *ERC20Indexer {
	return &ERC20Indexer{
		database:        database,
		cache:           cache,
		logger:          logger,
		eventDecoder:    eventDecoder,
		contractManager: contractManager,
		tokenMetadata:   make(map[common.Address]*TokenMetadata),
		balances:        make(map[string]*TokenBalance),
		transferCache:   make(map[string]*TransferEvent),
	}
}

// IndexTransfers indexes ERC20 transfer events
func (ei *ERC20Indexer) IndexTransfers(
	ctx context.Context,
	events []*core.BlockchainEvent,
) error {
	if len(events) == 0 {
		return nil
	}

	ei.logger.Debug("indexing transfer events", core.LogKeyCount, len(events))

	for _, event := range events {
		if err := ei.indexTransferEvent(ctx, event); err != nil {
			ei.logger.Error("failed to index transfer event",
				core.LogKeyError, err.Error(),
				core.LogKeyEventID, event.ID,
				core.LogKeyBlockNumber, event.BlockNumber,
				"tx_hash", event.TransactionHash.Hex())
			continue
		}
	}

	return nil
}

// indexTransferEvent indexes a single transfer event
func (ei *ERC20Indexer) indexTransferEvent(
	ctx context.Context,
	event *core.BlockchainEvent,
) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	// Decode event data
	transferEvent, err := ei.decodeTransferEvent(event)
	if err != nil {
		return err
	}

	// Validate transfer event
	if err := ei.validateTransferEvent(transferEvent); err != nil {
		return err
	}

	// Store in cache
	cacheKey := fmt.Sprintf("transfer:%s:%d:%d", transferEvent.TransactionHash.Hex(), transferEvent.BlockNumber, transferEvent.LogIndex)
	ei.mu.Lock()
	ei.transferCache[cacheKey] = transferEvent
	ei.mu.Unlock()

	// Update balances
	ei.updateBalance(transferEvent)

	ei.logger.Debug("indexed transfer event",
		"token", transferEvent.Token.Hex(),
		core.LogKeySender, transferEvent.From.Hex(),
		core.LogKeyRecipient, transferEvent.To.Hex(),
		"value", transferEvent.Value.String())

	return nil
}

// decodeTransferEvent decodes a raw event into a TransferEvent
func (ei *ERC20Indexer) decodeTransferEvent(event *core.BlockchainEvent) (*TransferEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("event is nil")
	}

	transferEvent := &TransferEvent{
		TransactionHash: event.TransactionHash,
		BlockNumber:     event.BlockNumber,
		BlockTimestamp:  event.BlockTimestamp,
		LogIndex:        event.LogIndex,
		Token:           event.ContractAddress,
		DecodedData:     event.DecodedData,
	}

	// Extract decoded data
	if event.DecodedData != nil {
		// From
		if from, ok := event.DecodedData["from"].(common.Address); ok {
			transferEvent.From = from
		}

		// To
		if to, ok := event.DecodedData["to"].(common.Address); ok {
			transferEvent.To = to
		}

		// Value
		if value, ok := event.DecodedData["value"].(*big.Int); ok {
			transferEvent.Value = value
		}
	}

	return transferEvent, nil
}

// validateTransferEvent validates a transfer event
func (ei *ERC20Indexer) validateTransferEvent(transferEvent *TransferEvent) error {
	if transferEvent == nil {
		return fmt.Errorf("transfer event is nil")
	}

	if transferEvent.Token == (common.Address{}) {
		return fmt.Errorf("token address is empty")
	}

	if transferEvent.Value == nil {
		return fmt.Errorf("value is nil")
	}

	if transferEvent.Value.Sign() < 0 {
		return fmt.Errorf("value cannot be negative")
	}

	return nil
}

// updateBalance updates account balance from transfer event
func (ei *ERC20Indexer) updateBalance(transferEvent *TransferEvent) {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	// Update sender balance (decrease)
	if transferEvent.From != (common.Address{}) {
		fromKey := fmt.Sprintf("%s:%s", transferEvent.Token.Hex(), transferEvent.From.Hex())
		fromBalance, exists := ei.balances[fromKey]
		if !exists {
			fromBalance = &TokenBalance{
				Token:       transferEvent.Token,
				Account:     transferEvent.From,
				Balance:     big.NewInt(0),
				BlockNumber: transferEvent.BlockNumber,
				BlockTime:   transferEvent.BlockTimestamp,
			}
		}
		fromBalance.Balance.Sub(fromBalance.Balance, transferEvent.Value)
		fromBalance.BlockNumber = transferEvent.BlockNumber
		fromBalance.BlockTime = transferEvent.BlockTimestamp
		ei.balances[fromKey] = fromBalance
	}

	// Update recipient balance (increase)
	if transferEvent.To != (common.Address{}) {
		toKey := fmt.Sprintf("%s:%s", transferEvent.Token.Hex(), transferEvent.To.Hex())
		toBalance, exists := ei.balances[toKey]
		if !exists {
			toBalance = &TokenBalance{
				Token:       transferEvent.Token,
				Account:     transferEvent.To,
				Balance:     big.NewInt(0),
				BlockNumber: transferEvent.BlockNumber,
				BlockTime:   transferEvent.BlockTimestamp,
			}
		}
		toBalance.Balance.Add(toBalance.Balance, transferEvent.Value)
		toBalance.BlockNumber = transferEvent.BlockNumber
		toBalance.BlockTime = transferEvent.BlockTimestamp
		ei.balances[toKey] = toBalance
	}
}

// GetBalance retrieves the balance of an account for a token
func (ei *ERC20Indexer) GetBalance(token, account common.Address) *big.Int {
	if token == (common.Address{}) || account == (common.Address{}) {
		return big.NewInt(0)
	}

	ei.mu.RLock()
	defer ei.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", token.Hex(), account.Hex())
	if balance, exists := ei.balances[key]; exists {
		return new(big.Int).Set(balance.Balance)
	}

	return big.NewInt(0)
}

// GetTransferHistory retrieves transfer history for an account
func (ei *ERC20Indexer) GetTransferHistory(
	ctx context.Context,
	token, account common.Address,
	fromBlock, toBlock uint64,
) (*TransferHistory, error) {
	if token == (common.Address{}) {
		return nil, fmt.Errorf("token address is empty")
	}

	if account == (common.Address{}) {
		return nil, fmt.Errorf("account address is empty")
	}

	if fromBlock > toBlock && toBlock != 0 {
		return nil, fmt.Errorf("from_block must be <= to_block")
	}

	ei.logger.Debug("getting transfer history",
		"token", token.Hex(),
		"account", account.Hex(),
		core.LogKeyFromBlock, fromBlock,
		core.LogKeyToBlock, toBlock)

	// Create filter for transfer events
	filter := &core.EventFilter{
		Network:         "ethereum",
		ContractAddress: []common.Address{token},
		FromBlock:       fromBlock,
		ToBlock:         toBlock,
		Limit:           10000,
	}

	// Query events from database
	events, err := ei.database.QueryEvents(ctx, filter)
	if err != nil {
		ei.logger.Error("failed to query transfer events",
			core.LogKeyError, err.Error(),
			"token", token.Hex())
		return nil, err
	}

	// Decode and filter events
	transfers := make([]*TransferEvent, 0)
	totalIncoming := big.NewInt(0)
	totalOutgoing := big.NewInt(0)
	var firstTransfer, lastTransfer int64

	for i, eventInterface := range events {
		event, ok := eventInterface.(*core.BlockchainEvent)
		if !ok {
			ei.logger.Warn("failed to cast event to BlockchainEvent", "index", i)
			continue
		}

		transferEvent, err := ei.decodeTransferEvent(event)
		if err != nil {
			ei.logger.Warn("failed to decode transfer event",
				core.LogKeyError, err.Error(),
				core.LogKeyEventID, event.ID)
			continue
		}

		// Filter for account (either sender or recipient)
		if transferEvent.From != account && transferEvent.To != account {
			continue
		}

		transfers = append(transfers, transferEvent)

		// Accumulate volumes
		if transferEvent.To == account {
			totalIncoming.Add(totalIncoming, transferEvent.Value)
		}
		if transferEvent.From == account {
			totalOutgoing.Add(totalOutgoing, transferEvent.Value)
		}

		// Track time range
		if i == 0 {
			firstTransfer = transferEvent.BlockTimestamp
		}
		lastTransfer = transferEvent.BlockTimestamp
	}

	// Calculate net change
	netChange := new(big.Int).Sub(totalIncoming, totalOutgoing)

	history := &TransferHistory{
		Transfers:     transfers,
		TotalIncoming: totalIncoming,
		TotalOutgoing: totalOutgoing,
		NetChange:     netChange,
		TransferCount: int64(len(transfers)),
		FirstTransfer: firstTransfer,
		LastTransfer:  lastTransfer,
	}

	ei.logger.Debug("retrieved transfer history",
		"token", token.Hex(),
		"account", account.Hex(),
		"transfer_count", len(transfers),
		"total_incoming", totalIncoming.String(),
		"total_outgoing", totalOutgoing.String())

	return history, nil
}

// GetTokenMetadata retrieves metadata for a token
func (ei *ERC20Indexer) GetTokenMetadata(token common.Address) *TokenMetadata {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	return ei.tokenMetadata[token]
}

// SetTokenMetadata sets metadata for a token
func (ei *ERC20Indexer) SetTokenMetadata(metadata *TokenMetadata) error {
	if metadata == nil {
		return fmt.Errorf("metadata is nil")
	}

	if metadata.Address == (common.Address{}) {
		return fmt.Errorf("token address is empty")
	}

	ei.mu.Lock()
	defer ei.mu.Unlock()

	metadata.LastUpdated = time.Now()
	ei.tokenMetadata[metadata.Address] = metadata

	return nil
}

// GetAllTokenMetadata retrieves metadata for all tokens
func (ei *ERC20Indexer) GetAllTokenMetadata() map[common.Address]*TokenMetadata {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	result := make(map[common.Address]*TokenMetadata)
	for addr, metadata := range ei.tokenMetadata {
		result[addr] = metadata
	}

	return result
}

// ClearCache clears the transfer event cache
func (ei *ERC20Indexer) ClearCache() {
	ei.mu.Lock()
	defer ei.mu.Unlock()

	ei.transferCache = make(map[string]*TransferEvent)
}

// GetCacheStats returns cache statistics
func (ei *ERC20Indexer) GetCacheStats() map[string]any {
	ei.mu.RLock()
	defer ei.mu.RUnlock()

	return map[string]any{
		"cached_transfers": len(ei.transferCache),
		"tracked_balances": len(ei.balances),
		"tokens":           len(ei.tokenMetadata),
	}
}
