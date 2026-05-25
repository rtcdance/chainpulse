package compliance

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestDepositProof_VerifyDepositInclusion(t *testing.T) {
	t.Parallel()

	validHash := [32]byte{1}
	emptyHash := [32]byte{}

	tests := []struct {
		name    string
		proof   DepositProof
		wantErr bool
		errMsg  string
	}{
		{
			"valid",
			DepositProof{
				L1TxHash: validHash, L2TxHash: validHash,
				L1BlockNumber: 10, L2BlockNumber: 20,
			},
			false, "",
		},
		{
			"empty_l1_tx",
			DepositProof{L2TxHash: validHash, L1BlockNumber: 1, L2BlockNumber: 2},
			true, "L1 transaction hash is empty",
		},
		{
			"empty_l2_tx",
			DepositProof{L1TxHash: validHash, L1BlockNumber: 1, L2BlockNumber: 2},
			true, "L2 transaction hash is empty",
		},
		{
			"zero_l1_block",
			DepositProof{L1TxHash: validHash, L2TxHash: validHash, L1BlockNumber: 0, L2BlockNumber: 1},
			true, "L1 block number is zero",
		},
		{
			"zero_l2_block",
			DepositProof{L1TxHash: validHash, L2TxHash: validHash, L1BlockNumber: 1, L2BlockNumber: 0},
			true, "L2 block number is zero",
		},
		{
			"l2_before_l1",
			DepositProof{L1TxHash: validHash, L2TxHash: validHash, L1BlockNumber: 20, L2BlockNumber: 10},
			true, "cannot precede L1 block",
		},
		{
			"storage_slot_no_value",
			DepositProof{
				L1TxHash: validHash, L2TxHash: validHash,
				L1BlockNumber: 10, L2BlockNumber: 20,
				StorageSlot: validHash, StorageValue: emptyHash,
			},
			true, "storage slot provided but storage value is empty",
		},
		{
			"storage_slot_with_value",
			DepositProof{
				L1TxHash: validHash, L2TxHash: validHash,
				L1BlockNumber: 10, L2BlockNumber: 20,
				StorageSlot: validHash, StorageValue: validHash,
			},
			false, "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.proof.VerifyDepositInclusion()
			if (err != nil) != tc.wantErr {
				t.Errorf("VerifyDepositInclusion() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && err != nil && !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error = %q, want containing %q", err.Error(), tc.errMsg)
			}
		})
	}
}

func TestWithdrawalProof_RequiresL1FinalityCheck(t *testing.T) {
	t.Parallel()

	w := WithdrawalProof{L1BlockNumber: 0}
	if w.RequiresL1FinalityCheck() {
		t.Error("should not require L1 finality when block is 0")
	}

	w.L1BlockNumber = 100
	if !w.RequiresL1FinalityCheck() {
		t.Error("should require L1 finality when block > 0")
	}
}

func TestWithdrawalProof_VerifyWithdrawalProof(t *testing.T) {
	t.Parallel()

	validHash := [32]byte{1}
	validAddr := [20]byte{1}

	t.Run("valid_no_merkle", func(t *testing.T) {
		t.Parallel()
		w := WithdrawalProof{
			L2TxHash:        validHash,
			OutputRoot:      validHash,
			MerkleProof:     [][32]byte{validHash},
			L1BridgeAddress: validAddr,
		}
		if err := w.VerifyWithdrawalProof(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty_l2_tx", func(t *testing.T) {
		t.Parallel()
		w := WithdrawalProof{
			OutputRoot:      validHash,
			MerkleProof:     [][32]byte{validHash},
			L1BridgeAddress: validAddr,
		}
		err := w.VerifyWithdrawalProof()
		if err == nil || !strings.Contains(err.Error(), "L2 transaction hash is empty") {
			t.Errorf("expected L2 tx hash error, got: %v", err)
		}
	})

	t.Run("empty_output_root", func(t *testing.T) {
		t.Parallel()
		w := WithdrawalProof{
			L2TxHash:        validHash,
			MerkleProof:     [][32]byte{validHash},
			L1BridgeAddress: validAddr,
		}
		err := w.VerifyWithdrawalProof()
		if err == nil || !strings.Contains(err.Error(), "output root is empty") {
			t.Errorf("expected output root error, got: %v", err)
		}
	})

	t.Run("empty_merkle_proof", func(t *testing.T) {
		t.Parallel()
		w := WithdrawalProof{
			L2TxHash:        validHash,
			OutputRoot:      validHash,
			L1BridgeAddress: validAddr,
		}
		err := w.VerifyWithdrawalProof()
		if err == nil || !strings.Contains(err.Error(), "merkle proof is empty") {
			t.Errorf("expected merkle proof error, got: %v", err)
		}
	})

	t.Run("empty_bridge_address", func(t *testing.T) {
		t.Parallel()
		w := WithdrawalProof{
			L2TxHash:    validHash,
			OutputRoot:  validHash,
			MerkleProof: [][32]byte{validHash},
		}
		err := w.VerifyWithdrawalProof()
		if err == nil || !strings.Contains(err.Error(), "L1 bridge address is empty") {
			t.Errorf("expected bridge address error, got: %v", err)
		}
	})

	t.Run("invalid_merkle", func(t *testing.T) {
		t.Parallel()
		w := WithdrawalProof{
			L2TxHash:        validHash,
			OutputRoot:      validHash,
			MerkleProof:     [][32]byte{validHash},
			L1BridgeAddress: validAddr,
			WithdrawalHash:  validHash,
		}
		err := w.VerifyWithdrawalProof()
		if err == nil || !strings.Contains(err.Error(), "merkle proof verification failed") {
			t.Errorf("expected merkle verification error, got: %v", err)
		}
	})
}

func TestVerifyMerkleProof(t *testing.T) {
	t.Parallel()

	emptyHash := [32]byte{}

	t.Run("empty_proof_matches_leaf_as_root", func(t *testing.T) {
		t.Parallel()
		leaf := [32]byte{1, 2, 3}
		if !VerifyMerkleProof(leaf, nil, leaf, nil) {
			t.Error("empty proof with leaf==root should pass")
		}
	})

	t.Run("empty_proof_mismatch", func(t *testing.T) {
		t.Parallel()
		leaf := [32]byte{1, 2, 3}
		root := [32]byte{4, 5, 6}
		if VerifyMerkleProof(leaf, nil, root, nil) {
			t.Error("empty proof with leaf!=root should fail")
		}
	})

	t.Run("single_level_default_left", func(t *testing.T) {
		t.Parallel()
		leaf := hashBytes([]byte("leaf"))
		sibling := hashBytes([]byte("sibling"))
		combined := append(leaf[:], sibling[:]...)
		root := hashTo32(keccak256(combined))

		proof := [][32]byte{sibling}
		if !VerifyMerkleProof(leaf, proof, root, nil) {
			t.Error("valid merkle proof should pass")
		}
	})

	t.Run("single_level_sibling_left", func(t *testing.T) {
		t.Parallel()
		leaf := hashBytes([]byte("leaf"))
		sibling := hashBytes([]byte("sibling"))
		combined := append(sibling[:], leaf[:]...)
		root := hashTo32(keccak256(combined))

		proof := [][32]byte{sibling}
		if !VerifyMerkleProof(leaf, proof, root, []bool{true}) {
			t.Error("valid merkle proof with sibling_left should pass")
		}
	})

	t.Run("flags_shorter_than_proof", func(t *testing.T) {
		t.Parallel()
		leaf := hashBytes([]byte("leaf"))
		s1 := hashBytes([]byte("s1"))
		s2 := hashBytes([]byte("s2"))

		mid := hashTo32(keccak256(append(leaf[:], s1[:]...)))
		root := hashTo32(keccak256(append(mid[:], s2[:]...)))

		proof := [][32]byte{s1, s2}
		flags := []bool{false}
		if !VerifyMerkleProof(leaf, proof, root, flags) {
			t.Error("should use default false for missing flags")
		}
	})

	t.Run("wrong_root", func(t *testing.T) {
		t.Parallel()
		leaf := [32]byte{1}
		proof := [][32]byte{emptyHash}
		wrongRoot := [32]byte{99}
		if VerifyMerkleProof(leaf, proof, wrongRoot, nil) {
			t.Error("wrong root should fail")
		}
	})
}

func TestWithdrawalDelay(t *testing.T) {
	t.Parallel()

	oneWeek := 7 * 24 * time.Hour

	tests := []struct {
		name    string
		chainID string
		want    time.Duration
		wantErr bool
	}{
		{"optimism_name", "optimism", oneWeek, false},
		{"arbitrum_name", "arbitrum", oneWeek, false},
		{"base_name", "base", oneWeek, false},
		{"optimism_id", "10", oneWeek, false},
		{"arbitrum_id", "42161", oneWeek, false},
		{"base_id", "8453", oneWeek, false},
		{"polygon_zkevm_id", "1101", oneWeek, false},
		{"unknown", "99999", 0, true},
		{"empty", "", 0, true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := WithdrawalDelay(tc.chainID)
			if (err != nil) != tc.wantErr {
				t.Errorf("WithdrawalDelay(%q) error = %v, wantErr %v", tc.chainID, err, tc.wantErr)
			}
			if err == nil && got != tc.want {
				t.Errorf("WithdrawalDelay(%q) = %v, want %v", tc.chainID, got, tc.want)
			}
		})
	}
}

func TestBridgeMessage_IsDeposit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction string
		want      bool
	}{
		{"deposit", "deposit", true},
		{"withdrawal", "withdrawal", false},
		{"empty", "", false},
		{"unknown", "transfer", false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			m := BridgeMessage{Direction: tc.direction}
			if got := m.IsDeposit(); got != tc.want {
				t.Errorf("IsDeposit() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestKeccak256(t *testing.T) {
	t.Parallel()

	t.Run("known_hash", func(t *testing.T) {
		t.Parallel()
		got := keccak256([]byte("hello"))
		if len(got) != 32 {
			t.Errorf("keccak256 length = %d, want 32", len(got))
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got := keccak256([]byte{})
		if len(got) != 32 {
			t.Errorf("keccak256 of empty = %d bytes", len(got))
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		t.Parallel()
		input := []byte("test-data")
		h1 := keccak256(input)
		h2 := keccak256(input)
		if hex.EncodeToString(h1) != hex.EncodeToString(h2) {
			t.Error("keccak256 should be deterministic")
		}
	})
}

func TestStateRootVerifier_VerifyStateRoot(t *testing.T) {
	t.Parallel()

	v := NewStateRootVerifier("10", [20]byte{1})

	t.Run("match", func(t *testing.T) {
		t.Parallel()
		root := [32]byte{1, 2, 3}
		err := v.VerifyStateRoot(L2OutputRoot{OutputRoot: root}, root)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		t.Parallel()
		root := [32]byte{1, 2, 3}
		wrong := [32]byte{4, 5, 6}
		err := v.VerifyStateRoot(L2OutputRoot{OutputRoot: wrong}, root)
		if err == nil || !strings.Contains(err.Error(), "state root mismatch") {
			t.Errorf("expected state root mismatch error, got: %v", err)
		}
	})
}

func TestNewStateRootVerifier(t *testing.T) {
	t.Parallel()

	addr := [20]byte{0x42}
	v := NewStateRootVerifier("optimism", addr)
	if v.chainID != "optimism" {
		t.Errorf("chainID = %q", v.chainID)
	}
	if v.l1Contract != addr {
		t.Errorf("l1Contract mismatch")
	}
}

func TestBridgeEventSignatures(t *testing.T) {
	t.Parallel()

	sigs := BridgeEventSignatures()
	if len(sigs) != 7 {
		t.Errorf("expected 7 signatures, got %d", len(sigs))
	}
	for i, sig := range sigs {
		if !strings.HasPrefix(sig, "0x") {
			t.Errorf("signature %d doesn't start with 0x: %s", i, sig)
		}
	}
}

func TestIsBridgeEvent(t *testing.T) {
	t.Parallel()

	t.Run("known_event", func(t *testing.T) {
		t.Parallel()
		if !IsBridgeEvent(common.HexToHash(OptimismFailedRelayedMessageTopic0)) {
			t.Error("expected bridge event for known topic")
		}
	})

	t.Run("unknown_topic", func(t *testing.T) {
		t.Parallel()
		unknown := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		if IsBridgeEvent(unknown) {
			t.Error("unknown topic should not be a bridge event")
		}
	})

	t.Run("zero_hash", func(t *testing.T) {
		t.Parallel()
		if IsBridgeEvent(common.Hash{}) {
			t.Error("zero hash should not be a bridge event")
		}
	})
}

func TestL2BridgeContractAddresses(t *testing.T) {
	t.Parallel()

	if len(L2BridgeContractAddresses) < 3 {
		t.Error("expected at least 3 chain entries")
	}
}

func TestDepositAmountField(t *testing.T) {
	t.Parallel()

	msg := BridgeMessage{
		Direction: "deposit",
		Amount:    big.NewInt(1000000000000000000),
	}
	if msg.Amount.Cmp(big.NewInt(1000000000000000000)) != 0 {
		t.Error("amount mismatch")
	}
}

func hashBytes(data []byte) [32]byte {
	return hashTo32(keccak256(data))
}

func hashTo32(h []byte) [32]byte {
	var result [32]byte
	copy(result[:], h)
	return result
}
