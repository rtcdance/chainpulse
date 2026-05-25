package mempool

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/blockchain"
)

func TestNewAAMempool_Defaults(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(0, 0)
	if m.maxSize != 10000 {
		t.Errorf("expected maxSize 10000, got %d", m.maxSize)
	}
	if m.entryTTL != 5*time.Minute {
		t.Errorf("expected entryTTL 5m, got %v", m.entryTTL)
	}
}

func TestNewAAMempool_Custom(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(500, 10*time.Minute)
	if m.maxSize != 500 {
		t.Errorf("expected maxSize 500, got %d", m.maxSize)
	}
}

func TestAAMempool_AddEntry_Nil(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	if m.AddEntry(nil) {
		t.Error("expected add nil to fail")
	}
}

func TestAAMempool_AddEntry_NilUserOp(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	if m.AddEntry(&AAMempoolEntry{Hash: "h1"}) {
		t.Error("expected add nil userOp to fail")
	}
}

func TestAAMempool_AddEntry_Duplicate(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	e := &AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "dup",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(10),
	}
	if !m.AddEntry(e) {
		t.Error("expected first add to succeed")
	}
	if m.AddEntry(e) {
		t.Error("expected duplicate add to fail")
	}
}

func TestAAMempool_AddEntry_EvictOnFull(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(2, time.Minute)
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "low",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(10),
	})
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "high",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(100),
	})
	if !m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "extra",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(50),
	}) {
		t.Error("expected add to succeed with eviction")
	}
	if m.Size() != 2 {
		t.Errorf("expected size 2 after eviction, got %d", m.Size())
	}
	if m.Contains("low") {
		t.Error("expected lowest priority entry to be evicted")
	}
}

func TestAAMempool_EvictStale(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, 50*time.Millisecond)
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "stale",
		SubmittedAt: time.Now().Add(-100 * time.Millisecond),
		PriorityFee: big.NewInt(10),
	})
	m.EvictStale()
	if m.Size() != 0 {
		t.Errorf("expected stale entry evicted, got size %d", m.Size())
	}
}

func TestAAMempool_EvictStale_NoneStale(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Hour)
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "fresh",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(10),
	})
	m.EvictStale()
	if m.Size() != 1 {
		t.Errorf("expected fresh entry to remain, got size %d", m.Size())
	}
}

func TestPriorityFeeHeap_Less(t *testing.T) {
	t.Parallel()
	now := time.Now()
	h := priorityFeeHeap{
		{Hash: "a", PriorityFee: big.NewInt(100), SubmittedAt: now},
		{Hash: "b", PriorityFee: big.NewInt(50), SubmittedAt: now},
	}
	if !h.Less(0, 1) {
		t.Error("expected higher fee to be less")
	}
	if h.Less(1, 0) {
		t.Error("expected lower fee to not be less")
	}
}

func TestPriorityFeeHeap_Less_SameFee(t *testing.T) {
	t.Parallel()
	early := time.Now()
	late := early.Add(time.Second)
	h := priorityFeeHeap{
		{Hash: "early", PriorityFee: big.NewInt(100), SubmittedAt: early},
		{Hash: "late", PriorityFee: big.NewInt(100), SubmittedAt: late},
	}
	if !h.Less(0, 1) {
		t.Error("expected earlier submission to be less when fees equal")
	}
}

func TestPriorityFeeHeap_Less_NilFee(t *testing.T) {
	t.Parallel()
	early := time.Now()
	late := early.Add(time.Second)
	h := priorityFeeHeap{
		{Hash: "a", PriorityFee: nil, SubmittedAt: early},
		{Hash: "b", PriorityFee: big.NewInt(50), SubmittedAt: late},
	}
	if !h.Less(0, 1) {
		t.Error("expected nil fee to use time comparison (earlier wins)")
	}
}

func TestPriorityFeeHeap_Push_BadType(t *testing.T) {
	t.Parallel()
	var h priorityFeeHeap
	h.Push("not an entry")
	if len(h) != 0 {
		t.Error("expected push to reject bad type")
	}
}

func TestAAMempool_RemoveEntry_NotFound(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	m.RemoveEntry("nonexistent")
	if m.Size() != 0 {
		t.Error("expected noop on remove nonexistent")
	}
}

func TestAAMempool_GetPendingOps_Empty(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	ops := m.GetPendingOps(5)
	if len(ops) != 0 {
		t.Errorf("expected 0 ops, got %d", len(ops))
	}
}

func TestAAMempool_GetPendingOps_LessThanN(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "h1",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(10),
	})
	ops := m.GetPendingOps(5)
	if len(ops) != 1 {
		t.Errorf("expected 1 op, got %d", len(ops))
	}
}

func TestAAMempool_GetPendingOps_EvictsExpiredDuringGet(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, 10*time.Millisecond)
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{},
		Hash:        "expired",
		SubmittedAt: time.Now().Add(-time.Second),
		PriorityFee: big.NewInt(10),
	})
	ops := m.GetPendingOps(5)
	if len(ops) != 0 {
		t.Errorf("expected expired entry to be evicted, got %d", len(ops))
	}
}

func TestAAMempool_Contains_NotFound(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	if m.Contains("nonexistent") {
		t.Error("expected false for nonexistent")
	}
}

func TestAAMempool_SenderAddress(t *testing.T) {
	t.Parallel()
	m := NewAAMempool(10, time.Minute)
	addr := common.HexToAddress("0xABCD")
	m.AddEntry(&AAMempoolEntry{
		UserOp:      &blockchain.UserOperation{Sender: addr},
		Hash:        "h1",
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(10),
		Sender:      addr,
	})
	if !m.Contains("h1") {
		t.Error("expected to contain entry")
	}
}
