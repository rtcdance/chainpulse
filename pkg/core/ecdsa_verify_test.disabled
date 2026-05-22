package core

import (
	"crypto/ecdsa"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestEthSignHash(t *testing.T) {
	t.Parallel()
	hash := EthSignHash([]byte("Hello!"))
	if hash == (common.Hash{}) {
		t.Error("expected non-zero hash")
	}
	hash2 := EthSignHash([]byte("Hello!"))
	if hash != hash2 {
		t.Error("expected deterministic hash")
	}
}

func TestSignAndRecover(t *testing.T) {
	t.Parallel()
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	expected := crypto.PubkeyToAddress(privateKey.PublicKey)
	sig, err := SignMessage([]byte("auth"), privateKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if len(sig) != 65 {
		t.Fatalf("expected 65-byte signature, got %d", len(sig))
	}
	recovered, err := RecoverAddress([]byte("auth"), sig)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if recovered != expected {
		t.Errorf("recovered %s, expected %s", recovered.Hex(), expected.Hex())
	}
}

func TestVerifySignature(t *testing.T) {
	t.Parallel()
	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	sig, _ := SignMessage([]byte("msg"), privateKey)
	if err := VerifySignature([]byte("msg"), sig, addr); err != nil {
		t.Errorf("expected valid: %v", err)
	}
	wrong := common.HexToAddress("0x0000000000000000000000000000000000000001")
	if err := VerifySignature([]byte("msg"), sig, wrong); err == nil {
		t.Error("expected error for wrong address")
	}
}

func TestVerifySignature_InvalidLength(t *testing.T) {
	t.Parallel()
	_, err := RecoverAddress([]byte("msg"), []byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for invalid signature length")
	}
}

func TestRecoverAddressFromHash(t *testing.T) {
	t.Parallel()
	privateKey, _ := crypto.GenerateKey()
	expected := crypto.PubkeyToAddress(privateKey.PublicKey)
	sig, _ := SignMessage([]byte("data"), privateKey)
	hash := EthSignHash([]byte("data"))
	recovered, err := RecoverAddressFromHash(hash, sig)
	if err != nil {
		t.Fatalf("RecoverAddressFromHash: %v", err)
	}
	if recovered != expected {
		t.Errorf("recovered %s, expected %s", recovered.Hex(), expected.Hex())
	}
}

func TestPublicKeyToAddress(t *testing.T) {
	t.Parallel()
	privateKey, _ := crypto.GenerateKey()
	pub := &privateKey.PublicKey
	if PublicKeyToAddress(pub) != crypto.PubkeyToAddress(*pub) {
		t.Error("PublicKeyToAddress mismatch")
	}
}

var _ = ecdsa.PrivateKey{}
var _ = rand.Reader
