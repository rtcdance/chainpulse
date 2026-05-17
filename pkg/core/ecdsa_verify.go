package core

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var ethSignedMessagePrefix = []byte("\x19Ethereum Signed Message:\n")

// VerifySignature recovers the signer address from an EIP-191 signed message
// and verifies it matches the expected address. Returns an error if signature
// recovery fails or the recovered address does not match.
func VerifySignature(message []byte, signature []byte, expectedAddress common.Address) error {
	recovered, err := RecoverAddress(message, signature)
	if err != nil {
		return fmt.Errorf("recover signer: %w", err)
	}
	if recovered != expectedAddress {
		return fmt.Errorf("signature: recovered %s, expected %s", recovered.Hex(), expectedAddress.Hex())
	}
	return nil
}

// RecoverAddress recovers the Ethereum address that signed the given message
// using EIP-191 personal_sign format (prefixed with "\x19Ethereum Signed Message:\n").
func RecoverAddress(message []byte, signature []byte) (common.Address, error) {
	hash := EthSignHash(message)
	return recoverFromHash(hash, signature)
}

// RecoverAddressFromHash recovers the Ethereum address from a pre-computed hash
// and a 65-byte ECDSA signature (R || S || V).
func RecoverAddressFromHash(hash common.Hash, signature []byte) (common.Address, error) {
	return recoverFromHash(hash, signature)
}

// EthSignHash computes the Ethereum signed message hash (EIP-191).
// The message is prefixed with "\x19Ethereum Signed Message:\n" + len(message)
// before being hashed with Keccak-256. This matches eth_sign / personal_sign behavior.
func EthSignHash(message []byte) common.Hash {
	lenStr := fmt.Sprintf("%d", len(message))
	prefixed := make([]byte, len(ethSignedMessagePrefix))
	copy(prefixed, ethSignedMessagePrefix)
	prefixed = append(prefixed, []byte(lenStr)...)
	prefixed = append(prefixed, message...)
	return crypto.Keccak256Hash(prefixed)
}

func recoverFromHash(hash common.Hash, signature []byte) (common.Address, error) {
	if len(signature) != 65 {
		return common.Address{}, fmt.Errorf("signature length %d != 65", len(signature))
	}
	sig := make([]byte, 65)
	copy(sig, signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	pubKey, err := crypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return common.Address{}, fmt.Errorf("SigToPub: %w", err)
	}
	return crypto.PubkeyToAddress(*pubKey), nil
}

// SignMessage produces a 65-byte ECDSA signature (R || S || V) in EIP-191 format
// using the given private key. The V value is adjusted to Ethereum's convention (27/28).
func SignMessage(message []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	hash := EthSignHash(message)
	sig, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	sig[64] += 27
	return sig, nil
}

// PublicKeyToAddress derives the canonical Ethereum address from a secp256k1 public key.
func PublicKeyToAddress(pub *ecdsa.PublicKey) common.Address {
	return crypto.PubkeyToAddress(*pub)
}
