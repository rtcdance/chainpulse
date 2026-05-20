package blockchain

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestBlockchainTypesSmoke(t *testing.T) {
	// BlockchainEvent construction and method calls
	evt := BlockchainEvent{
		ID:             "evt-1",
		BlockNumber:    100,
		TransactionHash: common.HexToHash("0xabc"),
		Status:         EventStatusPending,
		CreatedAt:      time.Now(),
	}
	if !evt.IsPending() {
		t.Error("expected pending")
	}
	if evt.IsConfirmed() {
		t.Error("expected not confirmed")
	}

	// Transaction
	tx := Transaction{
		Hash: common.HexToHash("0xdef"),
		Type: TxEIP1559,
	}
	if !tx.IsEIP1559() {
		t.Error("expected EIP-1559")
	}
	if tx.IsSuccessful() {
		t.Error("expected not successful (status=0)")
	}

	// Block
	block := Block{Number: 100}
	ts := block.GetTimestamp()
	if ts.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	// EventStatus constants
	if EventStatusConfirmed != "confirmed" {
		t.Error("bad confirmed constant")
	}

	// TxType constants
	if TxLegacy != 0 || TxEIP1559 != 2 || TxBlob != 3 {
		t.Error("bad tx type constants")
	}

	// TxStatus constants
	if TxStatusUnknown != 0 || TxStatusSuccess != 2 {
		t.Error("bad tx status constants")
	}

	// UserOperation
	uo := UserOperation{Sender: common.HexToAddress("0x1234")}
	if uo.HasPaymaster() {
		t.Error("expected no paymaster")
	}

	// BlockchainEvent.EffectiveGasPrice
	evt.GasPrice = big.NewInt(50)
	evt.TransactionType = TxLegacy
	price := evt.EffectiveGasPrice(big.NewInt(10))
	if price == nil || price.Cmp(big.NewInt(50)) != 0 {
		t.Error("expected effective gas price 50 for legacy")
	}

	// BlobSidecar size
	var blob Blob
	if len(blob) != 131072 {
		t.Error("blob should be 131072 bytes")
	}

	// EntryPoint
	if EntryPointV06 != "0.6" {
		t.Error("bad entrypoint version")
	}
	addr := EntryPointAddresses[EntryPointV06]
	if addr == (common.Address{}) {
		t.Error("expected non-zero entrypoint address")
	}
	ver := EntryPointVersionForAddress(addr)
	if ver != EntryPointV06 {
		t.Error("expected v0.6")
	}

	// UserOperationV07
	uo07 := UserOperationV07{
		AccountGasLimits: make([]byte, 32),
		MaxFeePerGas:     make([]byte, 32),
	}
	verGas, callGas := uo07.DecodeV07GasLimits()
	if verGas != 0 || callGas != 0 {
		t.Error("expected zero gas limits")
	}
	uoConv := uo07.ToUserOperation()
	if uoConv == nil {
		t.Error("expected non-nil user operation")
	}

	// Receipt
	rec := TransactionReceipt{Status: 1}
	if !rec.IsSuccessful() {
		t.Error("expected successful receipt")
	}

	// PaymasterReputation
	pr := PaymasterReputation{Address: common.HexToAddress("0xbeef")}
	pr.UpdateProposed()
	pr.UpdateIncluded()
	if pr.OpsSeen != 1 || pr.OpsIncluded != 1 {
		t.Error("expected 1/1 ops")
	}
	rate := pr.CalculateInclusionRate()
	if rate != 1.0 {
		t.Error("expected 100% inclusion rate")
	}
}
