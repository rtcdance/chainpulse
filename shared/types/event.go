package types

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type IndexedEvent struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BlockNumber *big.Int  `json:"block_number" gorm:"index"`
	TxHash      string    `json:"tx_hash" gorm:"index"`
	EventName   string    `json:"event_name" gorm:"index"`
	Contract    string    `json:"contract" gorm:"index"`
	From        string    `json:"from,omitempty"`
	To          string    `json:"to,omitempty"`
	TokenID     string    `json:"token_id,omitempty"`
	Value       string    `json:"value,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NFTTransferEvent struct {
	BlockNumber *big.Int    `json:"block_number"`
	TxHash      common.Hash `json:"tx_hash"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	TokenID     *big.Int    `json:"token_id"`
	Contract    common.Address `json:"contract"`
	Timestamp   time.Time   `json:"timestamp"`
}

type TokenTransferEvent struct {
	BlockNumber *big.Int    `json:"block_number"`
	TxHash      common.Hash `json:"tx_hash"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	Value       *big.Int    `json:"value"`
	Contract    common.Address `json:"contract"`
	Timestamp   time.Time   `json:"timestamp"`
}

type EventFilter struct {
	EventType   string `json:"event_type"`
	Contract    string `json:"contract"`
	FromBlock   *big.Int `json:"from_block"`
	ToBlock     *big.Int `json:"to_block"`
	Limit       int    `json:"limit"`
}

type RawEvent struct {
	BlockNumber *big.Int    `json:"block_number"`
	BlockHash   string      `json:"block_hash"`
	TxHash      string      `json:"tx_hash"`
	EventName   string      `json:"event_name"`
	ContractAddr string     `json:"contract_addr"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
}

type Database struct {
	// This would be an interface to the actual database implementation
	// For now, we'll define the methods that would be implemented
}

// The actual implementation would be in the database package
// This is just a placeholder for the interface
	Offset      int    `json:"offset"`
}

type LastProcessedBlock struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	BlockNumber *big.Int  `json:"block_number"`
	BlockHash   string    `json:"block_hash"` // Add block hash for reorg detection
	ChainID     string    `json:"chain_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProcessedEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	EventKey  string    `json:"event_key" gorm:"index;unique"`
	Processed bool      `json:"processed"`
	Timestamp time.Time `json:"timestamp"`
}

type ReorgConfig struct {
	Enabled       bool          `json:"enabled"`
	CheckInterval time.Duration `json:"check_interval"`
	Depth         int           `json:"depth"`     // 重组检测深度
	MaxDepth      int           `json:"max_depth"` // 最大重组深度
}

type Contract struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Address   string    `json:"address" gorm:"index;unique"`
	Name      string    `json:"name,omitempty"`
	Symbol    string    `json:"symbol,omitempty"`
	Type      string    `json:"type,omitempty"` // ERC20, ERC721, ERC1155, etc.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Stats struct {
	TotalEvents    int64 `json:"total_events"`
	TotalContracts int64 `json:"total_contracts"`
	LatestBlock    int64 `json:"latest_block"`
}