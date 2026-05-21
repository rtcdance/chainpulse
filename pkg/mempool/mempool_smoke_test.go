package mempool

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestAAMempoolSmoke(t *testing.T) {
	m := NewAAMempool(100, time.Minute)
	if m.Size() != 0 {
		t.Error("expected empty mempool")
	}

	entry := &AAMempoolEntry{
		UserOp: &blockchain.UserOperation{
			Sender: common.HexToAddress("0x1234"),
			Nonce:  big.NewInt(1),
		},
		EntryPointAddr: common.HexToAddress("0x5678"),
		SubmittedAt:    time.Now(),
		PriorityFee:    big.NewInt(100),
		Sender:         common.HexToAddress("0x1234"),
		Hash:           "hash-1",
	}

	ok := m.AddEntry(entry)
	if !ok {
		t.Error("expected add to succeed")
	}
	if m.Size() != 1 {
		t.Errorf("expected size 1, got %d", m.Size())
	}
	if !m.Contains("hash-1") {
		t.Error("expected to contain hash-1")
	}

	m.RemoveEntry("hash-1")
	if m.Size() != 0 {
		t.Error("expected empty after remove")
	}
}

func TestAAMempoolGetPendingOps(t *testing.T) {
	m := NewAAMempool(10, time.Minute)
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		PriorityFee: big.NewInt(50),
		Hash:        "hash-low",
		SubmittedAt: time.Now(),
	})
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		PriorityFee: big.NewInt(200),
		Hash:        "hash-high",
		SubmittedAt: time.Now(),
	})

	ops := m.GetPendingOps(10)
	if len(ops) == 0 {
		t.Fatal("expected pending ops")
	}
	if ops[0].Hash != "hash-high" {
		t.Error("expected highest priority fee first")
	}
}
