package core

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// EventStatus represents the processing status of an event
type EventStatus string

const (
	EventStatusPending   EventStatus = "pending"
	EventStatusConfirmed EventStatus = "confirmed"
	EventStatusFailed    EventStatus = "failed"
	EventStatusReorged   EventStatus = "reorged"
)

// BlockchainEvent represents a blockchain event with full details
type BlockchainEvent struct {
	// Event identification
	ID             string    `json:"id"`
	EventHash      string    `json:"event_hash"`
	EventSignature common.Hash `json:"event_signature"`

	// Block information
	BlockNumber    uint64      `json:"block_number"`
	BlockHash      common.Hash `json:"block_hash"`
	BlockTimestamp int64       `json:"block_timestamp"`

	// Transaction information
	TransactionHash  common.Hash `json:"transaction_hash"`
	TransactionIndex uint        `json:"transaction_index"`
	GasUsed          uint64      `json:"gas_used"`
	GasPrice         *big.Int    `json:"gas_price"`

	// Log information
	LogIndex uint `json:"log_index"`
	Removed  bool `json:"removed"`

	// Contract information
	ContractAddress common.Address `json:"contract_address"`
	EventName       string         `json:"event_name"`
	EventTopic      []common.Hash  `json:"event_topic"`

	// Event data
	EventData   []byte                 `json:"event_data"`
	DecodedData map[string]interface{} `json:"decoded_data"`

	// Indexing metadata
	ChainID     string      `json:"chain_id"`
	Network     string      `json:"network"`
	Status      EventStatus `json:"status"`

	// Timestamps
	CreatedAt   time.Time `json:"created_at"`
	ProcessedAt time.Time `json:"processed_at"`
	IndexedAt   time.Time `json:"indexed_at"`
}

// Transaction represents a blockchain transaction
type Transaction struct {
	Hash             common.Hash    `json:"hash"`
	From             common.Address `json:"from"`
	To               *common.Address `json:"to"`
	Value            *big.Int       `json:"value"`
	Gas              uint64         `json:"gas"`
	GasPrice         *big.Int       `json:"gas_price"`
	Input            []byte         `json:"input"`
	Nonce            uint64         `json:"nonce"`
	BlockNumber      uint64         `json:"block_number"`
	BlockHash        common.Hash    `json:"block_hash"`
	TransactionIndex uint           `json:"transaction_index"`
	Status           uint64         `json:"status"` // 1 = success, 0 = failed
	ContractAddress  *common.Address `json:"contract_address"`
	CumulativeGasUsed uint64         `json:"cumulative_gas_used"`
	Logs             []*types.Log   `json:"logs"`
}

// Block represents a blockchain block
type Block struct {
	Number       uint64          `json:"number"`
	Hash         common.Hash     `json:"hash"`
	ParentHash   common.Hash     `json:"parent_hash"`
	Timestamp    int64           `json:"timestamp"`
	Miner        common.Address  `json:"miner"`
	Difficulty   *big.Int        `json:"difficulty"`
	GasLimit     uint64          `json:"gas_limit"`
	GasUsed      uint64          `json:"gas_used"`
	Transactions []common.Hash   `json:"transactions"`
	LogsBloom    types.Bloom     `json:"logs_bloom"`
}

// Validate validates the blockchain event
func (be *BlockchainEvent) Validate() error {
	if be.BlockNumber == 0 {
		return ErrInvalidBlockNumber
	}
	if be.TransactionHash == (common.Hash{}) {
		return ErrInvalidTransactionHash
	}
	if be.ContractAddress == (common.Address{}) {
		return ErrInvalidContractAddress
	}
	if be.EventName == "" {
		return ErrInvalidEventName
	}
	return nil
}

// IsConfirmed returns whether the event is confirmed
func (be *BlockchainEvent) IsConfirmed() bool {
	return be.Status == EventStatusConfirmed
}

// IsPending returns whether the event is pending
func (be *BlockchainEvent) IsPending() bool {
	return be.Status == EventStatusPending
}

// IsFailed returns whether the event failed
func (be *BlockchainEvent) IsFailed() bool {
	return be.Status == EventStatusFailed
}

// IsReorged returns whether the event was reorged
func (be *BlockchainEvent) IsReorged() bool {
	return be.Status == EventStatusReorged
}

// Validate validates the transaction
func (t *Transaction) Validate() error {
	if t.Hash == (common.Hash{}) {
		return ErrInvalidTransactionHash
	}
	if t.From == (common.Address{}) {
		return ErrInvalidAddress
	}
	if t.BlockNumber == 0 {
		return ErrInvalidBlockNumber
	}
	return nil
}

// IsSuccessful returns whether the transaction was successful
func (t *Transaction) IsSuccessful() bool {
	return t.Status == 1
}

// IsFailed returns whether the transaction failed
func (t *Transaction) IsFailed() bool {
	return t.Status == 0
}

// Validate validates the block
func (b *Block) Validate() error {
	if b.Number == 0 {
		return ErrInvalidBlockNumber
	}
	if b.Hash == (common.Hash{}) {
		return ErrInvalidBlockHash
	}
	if b.Timestamp == 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

// GetTimestamp returns the block timestamp as time.Time
func (b *Block) GetTimestamp() time.Time {
	return time.Unix(b.Timestamp, 0)
}

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash   common.Hash    `json:"transaction_hash"`
	BlockNumber       uint64         `json:"block_number"`
	BlockHash         common.Hash    `json:"block_hash"`
	From              common.Address `json:"from"`
	To                *common.Address `json:"to"`
	GasUsed           uint64         `json:"gas_used"`
	CumulativeGasUsed uint64         `json:"cumulative_gas_used"`
	ContractAddress   *common.Address `json:"contract_address"`
	Logs              []*types.Log   `json:"logs"`
	Status            uint64         `json:"status"` // 1 = success, 0 = failed
	LogsBloom         types.Bloom    `json:"logs_bloom"`
}

// IsSuccessful returns whether the receipt indicates success
func (tr *TransactionReceipt) IsSuccessful() bool {
	return tr.Status == 1
}

// IsFailed returns whether the receipt indicates failure
func (tr *TransactionReceipt) IsFailed() bool {
	return tr.Status == 0
}
