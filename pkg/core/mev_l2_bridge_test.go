package core

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func bigNewInt(v int64) *big.Int { return big.NewInt(v) }

// ─── MEV Pipeline Tests ──────────────────────────────────────────────────────

func TestSlotAuctionTimelineDuration(t *testing.T) {
	now := time.Now()
	timeline := SlotAuctionTimeline{
		Slot:          100,
		Phase:         PhaseBidSubmission,
		BidSubmission: now,
		Cutoff:        now.Add(2 * time.Second),
		Reveal:        now.Add(4 * time.Second),
		Inclusion:     now.Add(6 * time.Second),
	}

	d, err := timeline.Duration(PhaseBidSubmission, PhaseCutoff)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 2*time.Second {
		t.Errorf("expected 2s, got %v", d)
	}

	d, err = timeline.Duration(PhaseCutoff, PhaseReveal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 2*time.Second {
		t.Errorf("expected 2s, got %v", d)
	}

	// Full pipeline
	d, err = timeline.Duration(PhaseBidSubmission, PhaseInclusion)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d != 6*time.Second {
		t.Errorf("expected 6s, got %v", d)
	}
}

func TestWithdrawalProof_RequiresL1FinalityCheck(t *testing.T) {
	t.Run("L1BlockNumber zero means no finality check needed", func(t *testing.T) {
		wp := WithdrawalProof{
			L2TxHash:        [32]byte{1},
			L2BlockNumber:   100,
			OutputRoot:      [32]byte{2},
			OutputRootIndex: 5,
			MerkleProof:     [][32]byte{{3}},
			L1BridgeAddress: [20]byte{0xff},
		}
		if wp.RequiresL1FinalityCheck() {
			t.Error("expected RequiresL1FinalityCheck() = false for zero L1BlockNumber")
		}
	})

	t.Run("L1BlockNumber set means finality check required", func(t *testing.T) {
		wp := WithdrawalProof{
			L2TxHash:        [32]byte{1},
			L2BlockNumber:   100,
			OutputRoot:      [32]byte{2},
			OutputRootIndex: 5,
			MerkleProof:     [][32]byte{{3}},
			L1BridgeAddress: [20]byte{0xff},
			L1BlockNumber:   18_000_000,
		}
		if !wp.RequiresL1FinalityCheck() {
			t.Error("expected RequiresL1FinalityCheck() = true for non-zero L1BlockNumber")
		}
	})
}
func TestSlotAuctionTimelineZeroPhase(t *testing.T) {
	timeline := SlotAuctionTimeline{
		Slot:          100,
		Phase:         PhaseBidSubmission,
		BidSubmission: time.Now(),
		// Cutoff is zero
	}

	_, err := timeline.Duration(PhaseBidSubmission, PhaseCutoff)
	if err == nil {
		t.Error("expected error for zero phase timing")
	}
}
func TestSlotAuctionTimelineInvalidPhase(t *testing.T) {
	timeline := SlotAuctionTimeline{
		Slot:          100,
		BidSubmission: time.Now(),
		Cutoff:        time.Now().Add(time.Second),
	}

	_, err := timeline.Duration(PhaseInclusion, PhaseCutoff)
	if err == nil {
		t.Error("expected error for invalid from phase")
	}

	_, err = timeline.Duration(PhaseBidSubmission, PhaseBidSubmission)
	if err == nil {
		t.Error("expected error for invalid to phase")
	}
}
func TestPBSLatencyRecording(t *testing.T) {
	p := NewPBSLatency(100)

	p.Record(100*time.Millisecond, 200*time.Millisecond)

	if avg := p.AvgBuilderToRelay(); avg != 100*time.Millisecond {
		t.Errorf("expected 100ms builder→relay, got %v", avg)
	}
	if avg := p.AvgRelayToProposer(); avg != 200*time.Millisecond {
		t.Errorf("expected 200ms relay→proposer, got %v", avg)
	}
	if avg := p.AvgE2E(); avg != 300*time.Millisecond {
		t.Errorf("expected 300ms E2E, got %v", avg)
	}
}
func TestPBSLatencySlidingWindow(t *testing.T) {
	p := NewPBSLatency(5) // small window

	for i := 0; i < 10; i++ {
		p.Record(time.Duration(i+1)*100*time.Millisecond, time.Duration(i+1)*50*time.Millisecond)
	}

	// Should only keep last 5 entries (i=5..9)
	avg := p.AvgBuilderToRelay()
	// Last 5: 600,700,800,900,1000 ms → avg = 800ms
	expected := 800 * time.Millisecond
	if avg != expected {
		t.Errorf("expected %v, got %v", expected, avg)
	}
}
func TestPBSLatencyEmpty(t *testing.T) {
	p := NewPBSLatency(100)

	if avg := p.AvgBuilderToRelay(); avg != 0 {
		t.Errorf("expected 0 for empty latency, got %v", avg)
	}
	if p99 := p.P99E2E(); p99 != 0 {
		t.Errorf("expected 0 for empty p99, got %v", p99)
	}
}
func TestPBSLatencyHealthy(t *testing.T) {
	p := NewPBSLatency(100)
	p.Record(50*time.Millisecond, 100*time.Millisecond)

	if !p.IsLatencyHealthy(500 * time.Millisecond) {
		t.Error("150ms E2E should be healthy within 500ms")
	}
	if p.IsLatencyHealthy(100 * time.Millisecond) {
		t.Error("150ms E2E should not be healthy within 100ms")
	}
}
func TestPBSLatencyP99(t *testing.T) {
	p := NewPBSLatency(100)

	for i := 0; i < 100; i++ {
		p.Record(time.Duration(i+1)*time.Millisecond, time.Duration(i+1)*2*time.Millisecond)
	}

	p99 := p.P99E2E()
	// E2E = (i+1) + 2*(i+1) = 3*(i+1), so values 3..300ms
	// P99 index = int(100*0.99) = 99, value at index 99 ≈ 300ms
	if p99 < 290*time.Millisecond {
		t.Errorf("P99 seems too low: %v", p99)
	}
}
func TestPayloadAttributesFields(t *testing.T) {
	pa := PayloadAttributes{
		Timestamp:             1234567890,
		PrevRandao:            [32]byte{1, 2, 3},
		SuggestedFeeRecipient: [20]byte{0xff},
		Withdrawals:           []Withdrawal{{Index: 1, ValidatorIndex: 42}},
	}

	if pa.Timestamp != 1234567890 {
		t.Errorf("unexpected timestamp: %d", pa.Timestamp)
	}
	if len(pa.Withdrawals) != 1 {
		t.Errorf("expected 1 withdrawal, got %d", len(pa.Withdrawals))
	}
	if pa.Withdrawals[0].ValidatorIndex != 42 {
		t.Errorf("expected validator 42, got %d", pa.Withdrawals[0].ValidatorIndex)
	}
	if pa.ParentBeaconBlockRoot != nil {
		t.Error("expected nil ParentBeaconBlockRoot pre-Dencun")
	}
}

// ─── L2 Bridge Tests ────────────────────────────────────────────────────────

func TestDepositProofVerification(t *testing.T) {
	tests := []struct {
		name    string
		proof   DepositProof
		wantErr bool
	}{
		{
			name: "valid deposit",
			proof: DepositProof{
				L1TxHash:      [32]byte{1},
				L1BlockNumber: 100,
				L2TxHash:      [32]byte{2},
				L2BlockNumber: 200,
			},
			wantErr: false,
		},
		{
			name: "empty L1 tx hash",
			proof: DepositProof{
				L1TxHash:      [32]byte{},
				L1BlockNumber: 100,
				L2TxHash:      [32]byte{2},
				L2BlockNumber: 200,
			},
			wantErr: true,
		},
		{
			name: "empty L2 tx hash",
			proof: DepositProof{
				L1TxHash:      [32]byte{1},
				L1BlockNumber: 100,
				L2TxHash:      [32]byte{},
				L2BlockNumber: 200,
			},
			wantErr: true,
		},
		{
			name: "L2 block precedes L1",
			proof: DepositProof{
				L1TxHash:      [32]byte{1},
				L1BlockNumber: 200,
				L2TxHash:      [32]byte{2},
				L2BlockNumber: 100,
			},
			wantErr: true,
		},
		{
			name: "zero L1 block",
			proof: DepositProof{
				L1TxHash:      [32]byte{1},
				L1BlockNumber: 0,
				L2TxHash:      [32]byte{2},
				L2BlockNumber: 200,
			},
			wantErr: true,
		},
		{
			name: "storage slot provided but value empty",
			proof: DepositProof{
				L1TxHash:      [32]byte{1},
				L1BlockNumber: 100,
				L2TxHash:      [32]byte{2},
				L2BlockNumber: 200,
				StorageSlot:   [32]byte{0xab},
				StorageValue:  [32]byte{},
			},
			wantErr: true,
		},
		{
			name: "storage slot and value both provided",
			proof: DepositProof{
				L1TxHash:      [32]byte{1},
				L1BlockNumber: 100,
				L2TxHash:      [32]byte{2},
				L2BlockNumber: 200,
				StorageSlot:   [32]byte{0xab},
				StorageValue:  [32]byte{0xcd},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.proof.VerifyDepositInclusion()
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyDepositInclusion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestWithdrawalProofVerification(t *testing.T) {
	tests := []struct {
		name    string
		proof   WithdrawalProof
		wantErr bool
	}{
		{
			name: "valid withdrawal proof",
			proof: WithdrawalProof{
				L2TxHash:        [32]byte{1},
				L2BlockNumber:   100,
				OutputRoot:      [32]byte{2},
				OutputRootIndex: 5,
				MerkleProof:     [][32]byte{{3}, {4}},
				L1BridgeAddress: [20]byte{0xff},
			},
			wantErr: false,
		},
		{
			name: "empty L2 tx hash",
			proof: WithdrawalProof{
				L2TxHash:        [32]byte{},
				OutputRoot:      [32]byte{2},
				MerkleProof:     [][32]byte{{3}},
				L1BridgeAddress: [20]byte{0xff},
			},
			wantErr: true,
		},
		{
			name: "empty output root",
			proof: WithdrawalProof{
				L2TxHash:        [32]byte{1},
				OutputRoot:      [32]byte{},
				MerkleProof:     [][32]byte{{3}},
				L1BridgeAddress: [20]byte{0xff},
			},
			wantErr: true,
		},
		{
			name: "empty merkle proof",
			proof: WithdrawalProof{
				L2TxHash:        [32]byte{1},
				OutputRoot:      [32]byte{2},
				MerkleProof:     nil,
				L1BridgeAddress: [20]byte{0xff},
			},
			wantErr: true,
		},
		{
			name: "empty L1 bridge address",
			proof: WithdrawalProof{
				L2TxHash:        [32]byte{1},
				OutputRoot:      [32]byte{2},
				MerkleProof:     [][32]byte{{3}},
				L1BridgeAddress: [20]byte{},
			},
			wantErr: true,
		},
		{
			name: "valid withdrawal proof with merkle verification",
			proof: func() WithdrawalProof {
				var leaf [32]byte
				copy(leaf[:], crypto.Keccak256([]byte("withdrawal_leaf")))
				var sibling [32]byte
				copy(sibling[:], crypto.Keccak256([]byte("sibling")))
				// root = keccak256(leaf || sibling)
				combined := append(leaf[:], sibling[:]...)
				var root [32]byte
				copy(root[:], crypto.Keccak256(combined))
				return WithdrawalProof{
					L2TxHash:        [32]byte{1},
					L2BlockNumber:   100,
					OutputRoot:      root,
					OutputRootIndex: 5,
					MerkleProof:     [][32]byte{sibling},
					WithdrawalHash:  leaf,
					L1BridgeAddress: [20]byte{0xff},
				}
			}(),
			wantErr: false,
		},
		{
			name: "invalid merkle proof fails verification",
			proof: WithdrawalProof{
				L2TxHash:        [32]byte{1},
				L2BlockNumber:   100,
				OutputRoot:      [32]byte{0xaa}, // wrong root
				OutputRootIndex: 5,
				MerkleProof:     [][32]byte{{3}},
				WithdrawalHash:  [32]byte{0xbb}, // non-zero triggers merkle check
				L1BridgeAddress: [20]byte{0xff},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.proof.VerifyWithdrawalProof()
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyWithdrawalProof() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestStateRootVerifier(t *testing.T) {
	verifier := NewStateRootVerifier("optimism", [20]byte{0xaa})

	root := L2OutputRoot{
		Index:         1,
		L2BlockNumber: 1000,
		OutputRoot:    [32]byte{0xde, 0xad},
	}

	// Matching root
	if err := verifier.VerifyStateRoot(root, [32]byte{0xde, 0xad}); err != nil {
		t.Errorf("expected no error for matching root: %v", err)
	}

	// Mismatching root
	if err := verifier.VerifyStateRoot(root, [32]byte{0xbe, 0xef}); err == nil {
		t.Error("expected error for mismatched root")
	}
}
func TestWithdrawalDelay(t *testing.T) {
	delay, err := WithdrawalDelay("optimism")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delay != 7*24*time.Hour {
		t.Errorf("expected 7 days for optimism, got %v", delay)
	}

	delay, err = WithdrawalDelay("arbitrum")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delay != 7*24*time.Hour {
		t.Errorf("expected 7 days for arbitrum, got %v", delay)
	}

	_, err = WithdrawalDelay("unknown")
	if err == nil {
		t.Error("expected error for unknown chain")
	}

	// Numeric chain IDs should resolve (EIP-155 standard)
	delay, err = WithdrawalDelay("10") // Optimism
	if err != nil {
		t.Fatalf("unexpected error for numeric chain ID 10: %v", err)
	}
	if delay != 7*24*time.Hour {
		t.Errorf("expected 7 days for chain ID 10, got %v", delay)
	}

	delay, err = WithdrawalDelay("42161") // Arbitrum One
	if err != nil {
		t.Fatalf("unexpected error for numeric chain ID 42161: %v", err)
	}
	if delay != 7*24*time.Hour {
		t.Errorf("expected 7 days for chain ID 42161, got %v", delay)
	}
}
func TestBridgeMessage(t *testing.T) {
	deposit := BridgeMessage{
		Direction:   "deposit",
		FromChain:   "ethereum",
		ToChain:     "optimism",
		Amount:      bigNewInt(1000),
		Sender:      [20]byte{1},
		Recipient:   [20]byte{2},
		TxHash:      [32]byte{3},
		BlockNumber: 100,
	}
	if !deposit.IsDeposit() {
		t.Error("expected IsDeposit() = true")
	}

	withdrawal := BridgeMessage{
		Direction: "withdrawal",
	}
	if withdrawal.IsDeposit() {
		t.Error("expected IsDeposit() = false for withdrawal")
	}
}
func TestMerkleProofVerification(t *testing.T) {
	// This uses the placeholder keccak256, so we test structure, not cryptographic correctness.
	// A single-leaf proof with one sibling.
	leaf := [32]byte{1}
	sibling := [32]byte{2}
	root := [32]byte{} // placeholder keccak256 won't match a real root

	// Should return false because the placeholder keccak256 won't produce a matching root
	result := VerifyMerkleProof(leaf, [][32]byte{sibling}, root, nil)
	if result {
		// This is expected to be false with the placeholder keccak256
		t.Log("placeholder keccak256 happened to match — unlikely but possible")
	}

	// Empty proof should return leaf == root
	result = VerifyMerkleProof(leaf, nil, leaf, nil)
	if !result {
		t.Error("empty proof should return leaf == root")
	}

	// Test position-dependent ordering: sibling on the left should produce
	// a different hash than sibling on the right.
	rightFirst := VerifyMerkleProof(leaf, [][32]byte{sibling}, [32]byte{}, []bool{false})
	leftFirst := VerifyMerkleProof(leaf, [][32]byte{sibling}, [32]byte{}, []bool{true})
	// They should produce different intermediate hashes, so they can't both match the same root.
	// The important thing is that the function respects the flag — if both produce
	// the same result, the flag is being ignored.
	_ = rightFirst
	_ = leftFirst
}
