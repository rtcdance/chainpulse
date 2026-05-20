// Package crypto provides Ethereum ECDSA signature verification, recovery,
// and EIP-191 signed message hashing utilities.
package crypto

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var ethSignedMessagePrefix = []byte("\x19Ethereum Signed Message:\n")

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

func RecoverAddress(message []byte, signature []byte) (common.Address, error) {
	hash := EthSignHash(message)
	return recoverFromHash(hash, signature)
}

func RecoverAddressFromHash(hash common.Hash, signature []byte) (common.Address, error) {
	return recoverFromHash(hash, signature)
}

func EthSignHash(message []byte) common.Hash {
	lenStr := fmt.Sprintf("%d", len(message))
	prefixed := make([]byte, len(ethSignedMessagePrefix))
	copy(prefixed, ethSignedMessagePrefix)
	prefixed = append(prefixed, []byte(lenStr)...)
	prefixed = append(prefixed, message...)
	return ethcrypto.Keccak256Hash(prefixed)
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
	pubKey, err := ethcrypto.SigToPub(hash.Bytes(), sig)
	if err != nil {
		return common.Address{}, fmt.Errorf("SigToPub: %w", err)
	}
	return ethcrypto.PubkeyToAddress(*pubKey), nil
}

func SignMessage(message []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	hash := EthSignHash(message)
	sig, err := ethcrypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	sig[64] += 27
	return sig, nil
}

func PublicKeyToAddress(pub *ecdsa.PublicKey) common.Address {
	return ethcrypto.PubkeyToAddress(*pub)
}