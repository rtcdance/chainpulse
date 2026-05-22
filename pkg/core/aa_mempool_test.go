package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rtcdance/chainpulse/pkg/mempool"
)

// --- AAMempool tests ---

func TestAAMempoolAddAndRemove(t *testing.T) {
	t.Parallel()
	pool := mempool.NewAAMempool(100, 5*time.Minute)

	op := &UserOperation{
		Sender:               common.HexToAddress("0x1234"),
		Nonce:                big.NewInt(1),
		CallGasLimit:         50000,
		VerificationGasLimit: 100000,
		MaxFeePerGas:         big.NewInt(2e9),
		MaxPriorityFeePerGas: big.NewInt(1e9),
	}

	entry := &mempool.AAMempoolEntry{
		UserOp:         op,
		EntryPointAddr: EntryPointAddresses[EntryPointV06],
		SubmittedAt:    time.Now(),
		PriorityFee:    big.NewInt(1e9),
		Sender:         op.Sender,
		Hash:           "test-hash-1",
		PreValidation:  &mempool.PreValidationResult{Success: true},
	}

	if !pool.AddEntry(entry) {
		t.Error("expected entry to be added")
	}
	if pool.Size() != 1 {
		t.Errorf("expected size 1, got %d", pool.Size())
	}
	if !pool.Contains("test-hash-1") {
		t.Error("expected entry to exist")
	}

	// Dedup
	dup := &mempool.AAMempoolEntry{
		UserOp:      op,
		SubmittedAt: time.Now(),
		PriorityFee: big.NewInt(2e9),
		Hash:        "test-hash-1",
	}
	if pool.AddEntry(dup) {
		t.Error("expected duplicate to be rejected")
	}

	// Remove
	pool.RemoveEntry("test-hash-1")
	if pool.Size() != 0 {
		t.Errorf("expected size 0 after removal, got %d", pool.Size())
	}
}

func TestAAMempoolPriorityOrdering(t *testing.T) {
	t.Parallel()
	pool := mempool.NewAAMempool(100, 5*time.Minute)

	for i := 0; i < 5; i++ {
		entry := &mempool.AAMempoolEntry{
			UserOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234"),
				MaxPriorityFeePerGas: big.NewInt(int64((i + 1) * 1e9)),
			},
			SubmittedAt:   time.Now(),
			PriorityFee:   big.NewInt(int64((i + 1) * 1e9)),
			Hash:          string(rune('a' + i)),
			PreValidation: &mempool.PreValidationResult{Success: true},
		}
		pool.AddEntry(entry)
	}

	pending := pool.GetPendingOps(3)
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending ops, got %d", len(pending))
	}

	// Should be ordered by priority fee descending
	for i := 1; i < len(pending); i++ {
		if pending[i].PriorityFee.Cmp(pending[i-1].PriorityFee) > 0 {
			t.Errorf("pending ops not ordered by priority fee desc: %v > %v",
				pending[i].PriorityFee, pending[i-1].PriorityFee)
		}
	}
}

func TestAAMempoolCapacityEviction(t *testing.T) {
	t.Parallel()
	pool := mempool.NewAAMempool(3, 5*time.Minute)

	for i := 0; i < 5; i++ {
		entry := &mempool.AAMempoolEntry{
			UserOp: &UserOperation{
				Sender:               common.HexToAddress("0x1234"),
				MaxPriorityFeePerGas: big.NewInt(int64((i + 1) * 1e9)),
			},
			SubmittedAt:   time.Now(),
			PriorityFee:   big.NewInt(int64((i + 1) * 1e9)),
			Hash:          string(rune('a' + i)),
			PreValidation: &mempool.PreValidationResult{Success: true},
		}
		pool.AddEntry(entry)
	}

	// Should have evicted lowest-priority entries to stay at capacity
	if pool.Size() > 3 {
		t.Errorf("expected size <= 3, got %d", pool.Size())
	}
}

func TestAAMempoolNilEntry(t *testing.T) {
	t.Parallel()
	pool := mempool.NewAAMempool(100, 5*time.Minute)
	if pool.AddEntry(nil) {
		t.Error("nil entry should not be added")
	}
}

func TestAAMempoolExpiredEntry(t *testing.T) {
	t.Parallel()
	pool := mempool.NewAAMempool(100, 100*time.Millisecond)

	entry := &mempool.AAMempoolEntry{
		UserOp:        &UserOperation{Sender: common.HexToAddress("0x1234")},
		SubmittedAt:   time.Now().Add(-200 * time.Millisecond), // already expired
		PriorityFee:   big.NewInt(1e9),
		Hash:          "expired",
		PreValidation: &mempool.PreValidationResult{Success: true},
	}
	pool.AddEntry(entry)

	// Trigger eviction
	pool.EvictStale()
	if pool.Size() != 0 {
		t.Errorf("expected expired entry to be evicted, got size %d", pool.Size())
	}
}

// --- EntryPoint Versioning tests ---

func TestEntryPointVersionForAddress(t *testing.T) {
	t.Parallel()
	v06 := EntryPointVersionForAddress(EntryPointAddresses[EntryPointV06])
	if v06 != EntryPointV06 {
		t.Errorf("expected v0.6, got %s", v06)
	}

	v07 := EntryPointVersionForAddress(EntryPointAddresses[EntryPointV07])
	if v07 != EntryPointV07 {
		t.Errorf("expected v0.7, got %s", v07)
	}

	unknown := EntryPointVersionForAddress(common.HexToAddress("0x0000"))
	if unknown != "" {
		t.Errorf("expected empty string for unknown address, got %s", unknown)
	}
}

func TestUserOperationV07DecodeGasLimits(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{
		AccountGasLimits: make([]byte, 32),
	}

	// verificationGasLimit = 100000 in first 16 bytes
	verificationGas := big.NewInt(100000)
	copy(op.AccountGasLimits[16-len(verificationGas.Bytes()):], verificationGas.Bytes())

	// callGasLimit = 50000 in second 16 bytes
	callGas := big.NewInt(50000)
	copy(op.AccountGasLimits[32-len(callGas.Bytes()):], callGas.Bytes())

	vgl, cgl := op.DecodeV07GasLimits()
	if vgl != 100000 {
		t.Errorf("verificationGasLimit = %d, want 100000", vgl)
	}
	if cgl != 50000 {
		t.Errorf("callGasLimit = %d, want 50000", cgl)
	}
}

func TestUserOperationV07DecodeFeePerGas(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{
		MaxFeePerGas: make([]byte, 32),
	}

	priorityFee := big.NewInt(1e9) // 1 Gwei
	copy(op.MaxFeePerGas[16-len(priorityFee.Bytes()):], priorityFee.Bytes())

	maxFee := big.NewInt(2e9) // 2 Gwei
	copy(op.MaxFeePerGas[32-len(maxFee.Bytes()):], maxFee.Bytes())

	maxPriorityFee, maxFeePerGas := op.DecodeV07FeePerGas()
	if maxPriorityFee.Cmp(big.NewInt(1e9)) != 0 {
		t.Errorf("maxPriorityFeePerGas = %d, want 1e9", maxPriorityFee)
	}
	if maxFeePerGas.Cmp(big.NewInt(2e9)) != 0 {
		t.Errorf("maxFeePerGas = %d, want 2e9", maxFeePerGas)
	}
}

func TestUserOperationV07ToUserOperation(t *testing.T) {
	t.Parallel()
	op := &UserOperationV07{
		Sender:             common.HexToAddress("0x1234"),
		Nonce:              big.NewInt(1),
		AccountGasLimits:   make([]byte, 32),
		MaxFeePerGas:       make([]byte, 32),
		PreVerificationGas: 21000,
		PaymasterAndData:   make([]byte, 20),
		Signature:          []byte{1, 2, 3},
	}

	// Set gas limits
	verificationGas := big.NewInt(100000)
	copy(op.AccountGasLimits[16-len(verificationGas.Bytes()):], verificationGas.Bytes())
	callGas := big.NewInt(50000)
	copy(op.AccountGasLimits[32-len(callGas.Bytes()):], callGas.Bytes())

	// Set fees
	priorityFee := big.NewInt(1e9)
	copy(op.MaxFeePerGas[16-len(priorityFee.Bytes()):], priorityFee.Bytes())
	maxFee := big.NewInt(2e9)
	copy(op.MaxFeePerGas[32-len(maxFee.Bytes()):], maxFee.Bytes())

	result := op.ToUserOperation()
	if result.CallGasLimit != 50000 {
		t.Errorf("CallGasLimit = %d, want 50000", result.CallGasLimit)
	}
	if result.VerificationGasLimit != 100000 {
		t.Errorf("VerificationGasLimit = %d, want 100000", result.VerificationGasLimit)
	}
	if result.MaxPriorityFeePerGas.Cmp(big.NewInt(1e9)) != 0 {
		t.Errorf("MaxPriorityFeePerGas = %d, want 1e9", result.MaxPriorityFeePerGas)
	}
	if result.MaxFeePerGas.Cmp(big.NewInt(2e9)) != 0 {
		t.Errorf("MaxFeePerGas = %d, want 2e9", result.MaxFeePerGas)
	}
	if result.PreVerificationGas != 21000 {
		t.Errorf("PreVerificationGas = %d, want 21000", result.PreVerificationGas)
	}
}
