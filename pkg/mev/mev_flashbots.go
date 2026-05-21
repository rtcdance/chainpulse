package mev

import (
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type FlashbotsRelay struct {
	mu       sync.RWMutex
	bids     []RelayBid
	sealed   []SealedBlock
	simulate func(blockNum uint64) bool
}

type RelayBid struct {
	BuilderName    string         `json:"builder_name"`
	BlockNumber    uint64         `json:"block_number"`
	FeeRecipient   common.Address `json:"fee_recipient"`
	BlockValue     *big.Int       `json:"block_value"`
	GasUsed        uint64         `json:"gas_used"`
	SubmittedAt    time.Time      `json:"submitted_at"`
	Simulated      bool           `json:"simulated"`
	SimulationPass bool           `json:"simulation_pass"`
}

type SealedBlock struct {
	Slot         uint64         `json:"slot"`
	BuilderName  string         `json:"builder_name"`
	BlockHash    common.Hash    `json:"block_hash"`
	BlockNumber  uint64         `json:"block_number"`
	FeeRecipient common.Address `json:"fee_recipient"`
	GasUsed      uint64         `json:"gas_used"`
	GasLimit     uint64         `json:"gas_limit"`
	Transactions int            `json:"transactions"`
	Timestamp    time.Time      `json:"timestamp"`
}

func NewFlashbotsRelay(simFn func(uint64) bool) *FlashbotsRelay {
	return &FlashbotsRelay{
		bids:     make([]RelayBid, 0, 100),
		sealed:   make([]SealedBlock, 0, 100),
		simulate: simFn,
	}
}

func (r *FlashbotsRelay) SubmitBid(builderName string, blockNumber uint64, feeRecipient common.Address, blockValue *big.Int, gasUsed uint64) *RelayBid {
	bid := &RelayBid{
		BuilderName:  builderName,
		BlockNumber:  blockNumber,
		FeeRecipient: feeRecipient,
		BlockValue:   blockValue,
		GasUsed:      gasUsed,
		SubmittedAt:  time.Now(),
	}

	if r.simulate != nil {
		bid.Simulated = true
		bid.SimulationPass = r.simulate(blockNumber)
	} else {
		bid.Simulated = false
		bid.SimulationPass = true
	}

	r.mu.Lock()
	r.bids = append(r.bids, *bid)
	r.mu.Unlock()

	return bid
}

func (r *FlashbotsRelay) SelectWinner(blockNumber uint64, slot uint64) *SealedBlock {
	r.mu.Lock()
	defer r.mu.Unlock()

	var winner *RelayBid
	for i, bid := range r.bids {
		if bid.BlockNumber != blockNumber || !bid.SimulationPass {
			continue
		}
		if winner == nil || (bid.BlockValue != nil && (winner.BlockValue == nil || bid.BlockValue.Cmp(winner.BlockValue) > 0)) {
			winner = &r.bids[i]
		}
	}

	if winner == nil {
		return nil
	}

	sealed := &SealedBlock{
		Slot:         slot,
		BuilderName:  winner.BuilderName,
		BlockNumber:  winner.BlockNumber,
		FeeRecipient: winner.FeeRecipient,
		GasUsed:      winner.GasUsed,
		GasLimit:     30000000,
		Transactions: 50,
		BlockHash:    randomHash(),
		Timestamp:    time.Now(),
	}

	r.sealed = append(r.sealed, *sealed)
	return sealed
}

type OrderFlow struct {
	mu          sync.Mutex
	privateTxns []PrivateTransaction
}

type PrivateTransaction struct {
	Hash         common.Hash `json:"hash"`
	From         string      `json:"from"`
	To           string      `json:"to"`
	Value        *big.Int    `json:"value"`
	MaxFeePerGas *big.Int    `json:"max_fee_per_gas"`
	SubmittedAt  time.Time   `json:"submitted_at"`
	BundleID     string      `json:"bundle_id,omitempty"`
	Intent       string      `json:"intent"`
}

func NewOrderFlow() *OrderFlow {
	return &OrderFlow{
		privateTxns: make([]PrivateTransaction, 0),
	}
}

func (of *OrderFlow) SubmitPrivateTx(from, to string, value *big.Int, maxFee *big.Int, intent string) common.Hash {
	tx := PrivateTransaction{
		Hash:         randomHash(),
		From:         from,
		To:           to,
		Value:        value,
		MaxFeePerGas: maxFee,
		SubmittedAt:  time.Now(),
		Intent:       intent,
	}

	of.mu.Lock()
	of.privateTxns = append(of.privateTxns, tx)
	of.mu.Unlock()

	return tx.Hash
}

func (of *OrderFlow) GetPendingBundle(intent string) []PrivateTransaction {
	of.mu.Lock()
	defer of.mu.Unlock()

	var result []PrivateTransaction
	var remaining []PrivateTransaction

	for _, tx := range of.privateTxns {
		if tx.Intent == intent {
			result = append(result, tx)
		} else {
			remaining = append(remaining, tx)
		}
	}
	of.privateTxns = remaining

	return result
}

func randomHash() common.Hash {
	return common.BytesToHash([]byte(time.Now().String()))
}
