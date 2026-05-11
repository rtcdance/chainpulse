package core

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// ─── L1→L2 Deposit Verification ─────────────────────────────────────────────

// DepositProof verifies that an L1→L2 deposit was correctly bridged.
// Optimistic rollups (Optimism, Arbitrum) use different deposit mechanisms,
// but all require verifying that the L1 transaction was included and the
// L2 state was updated accordingly.
type DepositProof struct {
	L1TxHash      [32]byte `json:"l1_tx_hash"`
	L1BlockNumber uint64   `json:"l1_block_number"`
	L2TxHash      [32]byte `json:"l2_tx_hash"`
	L2BlockNumber uint64   `json:"l2_block_number"`
	// MessagePasser storage slot on L2 proving the deposit was processed
	StorageSlot   [32]byte `json:"storage_slot"`
	StorageValue  [32]byte `json:"storage_value"`
}

// VerifyDepositInclusion performs basic validation of a deposit proof.
func (d *DepositProof) VerifyDepositInclusion() error {
	if d.L1TxHash == [32]byte{} {
		return fmt.Errorf("L1 transaction hash is empty")
	}
	if d.L2TxHash == [32]byte{} {
		return fmt.Errorf("L2 transaction hash is empty")
	}
	if d.L1BlockNumber == 0 {
		return fmt.Errorf("L1 block number is zero")
	}
	if d.L2BlockNumber == 0 {
		return fmt.Errorf("L2 block number is zero")
	}
	if d.L2BlockNumber < d.L1BlockNumber {
		return fmt.Errorf("L2 block (%d) cannot precede L1 block (%d)", d.L2BlockNumber, d.L1BlockNumber)
	}
	// If a storage slot is provided, the corresponding value must also be present.
	// Full verification (eth_getProof against L2 message passer) is the caller's
	// responsibility — this validates structural completeness of the proof.
	if d.StorageSlot != [32]byte{} && d.StorageValue == [32]byte{} {
		return fmt.Errorf("storage slot provided but storage value is empty — proof is incomplete")
	}
	return nil
}

// ─── L2→L1 Withdrawal Proof ─────────────────────────────────────────────────

// WithdrawalProof verifies that an L2→L1 withdrawal can be proven against
// the L1 bridge contract. This uses merkle proofs against the L2 output root
// that was posted to L1.
type WithdrawalProof struct {
	L2TxHash        [32]byte   `json:"l2_tx_hash"`
	L2BlockNumber   uint64     `json:"l2_block_number"`
	OutputRoot      [32]byte   `json:"output_root"`       // L2 output root posted to L1
	OutputRootIndex uint64     `json:"output_root_index"` // index in the output root list
	MerkleProof     [][32]byte `json:"merkle_proof"`      // merkle proof from withdrawal to output root
	L1BridgeAddress [20]byte   `json:"l1_bridge_address"`
	WithdrawalHash  [32]byte   `json:"withdrawal_hash"`   // merkle leaf: keccak256 of the withdrawal transaction
	L1BlockNumber   uint64     `json:"l1_block_number"`   // L1 block where output root was submitted (0 = unknown)
}

// VerifyWithdrawalProof validates a withdrawal proof. It checks that required
// fields are present and, if WithdrawalHash is provided, verifies the merkle
// proof cryptographically against the output root.
func (w *WithdrawalProof) VerifyWithdrawalProof() error {
	if w.L2TxHash == [32]byte{} {
		return fmt.Errorf("L2 transaction hash is empty")
	}
	if w.OutputRoot == [32]byte{} {
		return fmt.Errorf("output root is empty")
	}
	if len(w.MerkleProof) == 0 {
		return fmt.Errorf("merkle proof is empty")
	}
	if w.L1BridgeAddress == [20]byte{} {
		return fmt.Errorf("L1 bridge address is empty")
	}

	// Verify merkle proof if the withdrawal leaf hash is provided.
	// When WithdrawalHash is zero-valued (not yet populated by the caller),
	// the merkle check is skipped — this maintains backward compatibility.
	if w.WithdrawalHash != [32]byte{} {
		if !VerifyMerkleProof(w.WithdrawalHash, w.MerkleProof, w.OutputRoot, nil) {
			return fmt.Errorf("merkle proof verification failed: leaf %x does not prove to root %x",
				w.WithdrawalHash[:4], w.OutputRoot[:4])
		}
	}

	return nil
}

// RequiresL1FinalityCheck returns true when the withdrawal proof references
// an L1 block, meaning the caller should verify that the L1 block containing
// the output root submission is finalized before marking the withdrawal as proven.
// For optimistic rollups, this means the L1 challenge window must have elapsed.
func (w *WithdrawalProof) RequiresL1FinalityCheck() bool {
	return w.L1BlockNumber > 0
}

// VerifyMerkleProof verifies a merkle proof against a known root.
// proofFlags[i] indicates whether the sibling at level i is on the left side
// (true = sibling is left, current is right; false = current is left, sibling is right).
// If proofFlags is nil or shorter than the proof, remaining levels default to
// current-left ordering (false) for backward compatibility.
// Production L2 withdrawal proofs (Optimism, Polygon) require position-dependent
// ordering because a leaf in the right subtree must hash as sibling||current.
func VerifyMerkleProof(leaf [32]byte, proof [][32]byte, root [32]byte, proofFlags []bool) bool {
	current := leaf
	for i, sibling := range proof {
		siblingIsLeft := false
		if i < len(proofFlags) {
			siblingIsLeft = proofFlags[i]
		}
		var combined []byte
		if siblingIsLeft {
			combined = append(sibling[:], current[:]...)
		} else {
			combined = append(current[:], sibling[:]...)
		}
		hash := keccak256(combined)
		current = [32]byte(hash)
	}
	return current == root
}

// ─── State Root Verification ────────────────────────────────────────────────

// StateRootVerifier verifies L2 state roots against L1 contract storage.
// Rollups post their state roots (or output roots) to L1 contracts,
// and verifiers can check that L2 data matches what was posted on L1.
type StateRootVerifier struct {
	chainID    string
	l1Contract [20]byte // L1 rollup contract address
}

// NewStateRootVerifier creates a new state root verifier for a chain.
func NewStateRootVerifier(chainID string, l1Contract [20]byte) *StateRootVerifier {
	return &StateRootVerifier{
		chainID:    chainID,
		l1Contract: l1Contract,
	}
}

// L2OutputRoot represents a state root posted to L1.
type L2OutputRoot struct {
	Index           uint64   `json:"index"`
	L2BlockNumber   uint64   `json:"l2_block_number"`
	OutputRoot      [32]byte `json:"output_root"`
	L1Timestamp      uint64   `json:"l1_timestamp"`
	SubmissionTxHash [32]byte `json:"submission_tx_hash"`
}

// VerifyStateRoot checks if an L2 state root matches the one posted on L1.
func (v *StateRootVerifier) VerifyStateRoot(l2Root L2OutputRoot, expectedRoot [32]byte) error {
	if l2Root.OutputRoot != expectedRoot {
		return fmt.Errorf("state root mismatch: got %x, expected %x",
			l2Root.OutputRoot[:4], expectedRoot[:4])
	}
	return nil
}

// WithdrawalDelay calculates the minimum withdrawal delay for an L2.
// Optimism: ~7 days (1 week challenge period)
// Arbitrum: ~7 days
// Base: ~7 days (same as Optimism, uses OP stack)
func WithdrawalDelay(chainID string) (time.Duration, error) {
	oneWeek := 7 * 24 * time.Hour
	delays := map[string]time.Duration{
		// Symbolic names (backward compatible)
		"optimism":      oneWeek,
		"arbitrum":      oneWeek,
		"base":          oneWeek,
		"polygon-zkevm": oneWeek,
		// Numeric chain IDs (EIP-155 standard)
		"10":    oneWeek, // Optimism
		"42161": oneWeek, // Arbitrum One
		"8453":  oneWeek, // Base
		"1101":  oneWeek, // Polygon zkEVM
	}

	delay, ok := delays[chainID]
	if !ok {
		return 0, fmt.Errorf("unknown chain for withdrawal delay: %s", chainID)
	}
	return delay, nil
}

// BridgeMessage represents a cross-chain message.
type BridgeMessage struct {
	Direction     string   `json:"direction"` // "deposit" or "withdrawal"
	FromChain     string   `json:"from_chain"`
	ToChain       string   `json:"to_chain"`
	Amount        *big.Int `json:"amount"`
	Sender        [20]byte `json:"sender"`
	Recipient     [20]byte `json:"recipient"`
	TxHash        [32]byte `json:"tx_hash"`
	BlockNumber   uint64   `json:"block_number"`
}

// IsDeposit returns true if this is an L1→L2 deposit message.
func (m *BridgeMessage) IsDeposit() bool {
	return m.Direction == "deposit"
}

// keccak256 wraps go-ethereum's Keccak256 for merkle proof computation.
func keccak256(data []byte) []byte {
	return crypto.Keccak256(data)
}
