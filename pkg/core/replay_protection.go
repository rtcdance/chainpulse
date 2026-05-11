package core

import (
	"fmt"
	"math/big"
)

// SignerType represents the transaction signing algorithm.
type SignerType int

const (
	// SignerHomestead is the pre-EIP-155 signer without replay protection.
	// V values are 27 or 28.
	SignerHomestead SignerType = iota

	// SignerEIP155 is the EIP-155 signer with chain ID replay protection.
	// V values are 2*chainID + 35 or 2*chainID + 36.
	SignerEIP155
)

// String returns a human-readable name for the signer type.
func (s SignerType) String() string {
	switch s {
	case SignerHomestead:
		return "Homestead"
	case SignerEIP155:
		return "EIP-155"
	default:
		return "Unknown"
	}
}

// ValidateChainIDReplayProtection checks that a transaction's chain ID
// matches the expected chain, preventing cross-chain replay attacks.
// Returns nil if valid, or an error describing the mismatch.
//
// A nil txChainID indicates a pre-EIP-155 transaction that lacks replay
// protection — this returns an error unless expectedChainID is also 0
// (indicating a pre-EIP-155 chain is expected).
func ValidateChainIDReplayProtection(txChainID *big.Int, expectedChainID int) error {
	if expectedChainID == 0 {
		// Unknown or pre-EIP-155 target chain — skip validation
		return nil
	}

	if txChainID == nil {
		return fmt.Errorf("transaction lacks chain ID (pre-EIP-155 signing), expected chain %d: vulnerable to cross-chain replay", expectedChainID)
	}

	txID := txChainID.Int64()
	if txID != int64(expectedChainID) {
		return fmt.Errorf("chain ID mismatch: transaction signed for chain %d, expected chain %d", txID, expectedChainID)
	}

	return nil
}

// InferEIP155SignerType determines which signer to use based on chain ID:
//   - chain ID = 0 or nil: Homestead signer (pre-EIP-155, no replay protection)
//   - chain ID > 0: EIP-155 signer (with replay protection)
func InferEIP155SignerType(chainID *big.Int) SignerType {
	if chainID == nil || chainID.Sign() == 0 {
		return SignerHomestead
	}
	return SignerEIP155
}

// IsReplayVulnerable checks if a transaction is vulnerable to replay attacks
// based on the signature's V value.
//
// EIP-155 encodes the chain ID in V as: V = 2*chainID + 35 or V = 2*chainID + 36
// Pre-EIP-155 uses V = 27 or V = 28.
// Any V value of 27 or 28 indicates a replay-vulnerable transaction.
func IsReplayVulnerable(v uint64) bool {
	return v == 27 || v == 28
}

// ExtractChainIDFromV extracts the chain ID from an EIP-155 signature V value.
// Returns nil if V indicates a pre-EIP-155 (Homestead) signature.
//
// EIP-155: V = 2*chainID + 35 (even y) or V = 2*chainID + 36 (odd y)
// So: chainID = (V - 35) / 2
func ExtractChainIDFromV(v uint64) *big.Int {
	if v <= 28 {
		// Pre-EIP-155: V is 27 or 28
		return nil
	}

	// EIP-155: chainID = (V - 35) / 2
	chainID := new(big.Int).SetUint64(v)
	chainID.Sub(chainID, big.NewInt(35))
	chainID.Div(chainID, big.NewInt(2))
	return chainID
}

// ValidateSignatureV validates that a transaction's V value is consistent
// with the expected chain ID. Returns an error if the V value doesn't
// encode the correct chain ID.
func ValidateSignatureV(v uint64, expectedChainID int) error {
	if expectedChainID == 0 {
		// No chain ID enforcement
		return nil
	}

	if v <= 28 {
		return fmt.Errorf("signature V=%d indicates pre-EIP-155 signing (no replay protection), expected chain %d", v, expectedChainID)
	}

	extractedID := ExtractChainIDFromV(v)
	if extractedID == nil || extractedID.Int64() != int64(expectedChainID) {
		return fmt.Errorf("signature V=%d encodes chain ID %v, expected chain %d", v, extractedID, expectedChainID)
	}

	return nil
}
