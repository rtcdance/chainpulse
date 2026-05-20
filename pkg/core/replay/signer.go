// Package replay provides EIP-155 chain ID replay protection validation for
// Ethereum transaction signatures.
//
// These functions were originally defined in pkg/core and are re-exported
// there for backward compatibility.
package replay

import (
	"fmt"
	"math/big"
)

// SignerType represents the transaction signing algorithm.
type SignerType int

const (
	// SignerHomestead is the pre-EIP-155 signer without replay protection.
	SignerHomestead SignerType = iota
	// SignerEIP155 is the EIP-155 signer with chain ID replay protection.
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
func ValidateChainIDReplayProtection(txChainID *big.Int, expectedChainID int) error {
	if expectedChainID == 0 {
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

// InferEIP155SignerType determines which signer to use based on chain ID.
func InferEIP155SignerType(chainID *big.Int) SignerType {
	if chainID == nil || chainID.Sign() == 0 {
		return SignerHomestead
	}
	return SignerEIP155
}

// IsReplayVulnerable checks if a transaction is vulnerable to replay attacks.
func IsReplayVulnerable(v uint64) bool {
	return v == 27 || v == 28
}

// ExtractChainIDFromV extracts the chain ID from an EIP-155 signature V value.
func ExtractChainIDFromV(v uint64) *big.Int {
	if v <= 28 {
		return nil
	}
	chainID := new(big.Int).SetUint64(v)
	chainID.Sub(chainID, big.NewInt(35))
	chainID.Div(chainID, big.NewInt(2))
	return chainID
}

// ValidateSignatureV validates that a transaction's V value is consistent
// with the expected chain ID.
func ValidateSignatureV(v uint64, expectedChainID int) error {
	if expectedChainID == 0 {
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