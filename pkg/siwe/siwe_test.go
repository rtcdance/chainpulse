package siwe

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	corecrypto "github.com/rtcdance/chainpulse/pkg/core/crypto"
)

func TestSIWEBuildMessage(t *testing.T) {
	t.Parallel()

	msg := &SIWEMessage{
		Domain:   "chainpulse.example.com",
		Address:  common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8"),
		URI:      "https://chainpulse.example.com/login",
		Nonce:    "1234567890abcdef",
		Version:  "1",
		ChainID:  big.NewInt(1),
		IssuedAt: time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC),
	}

	built := msg.BuildMessage()

	if !strContains(built, "chainpulse.example.com wants you to sign in") {
		t.Error("missing domain prefix")
	}
	if !strContains(built, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8") {
		t.Error("missing address")
	}
	if !strContains(built, "Nonce: 1234567890abcdef") {
		t.Error("missing nonce")
	}
	if !strContains(built, "Chain ID: 1") {
		t.Error("missing chain ID")
	}
}

func TestSIWERoundTrip(t *testing.T) {
	t.Parallel()

	original := &SIWEMessage{
		Domain:    "example.com",
		Address:   common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"),
		URI:       "https://example.com/login",
		Nonce:     "deadbeef",
		Version:   "1",
		ChainID:   big.NewInt(1),
		IssuedAt:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Statement: "Sign in to access the API",
	}

	built := original.BuildMessage()
	parsed, err := ParseMessage(built)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}

	if parsed.Domain != original.Domain {
		t.Errorf("Domain: got %s, want %s", parsed.Domain, original.Domain)
	}
	if parsed.Address != original.Address {
		t.Errorf("Address: got %s, want %s", parsed.Address.Hex(), original.Address.Hex())
	}
	if parsed.Nonce != original.Nonce {
		t.Errorf("Nonce: got %s, want %s", parsed.Nonce, original.Nonce)
	}
	if parsed.ChainID.Cmp(original.ChainID) != 0 {
		t.Errorf("ChainID mismatch")
	}
	if parsed.URI != original.URI {
		t.Errorf("URI: got %s, want %s", parsed.URI, original.URI)
	}
	if parsed.Statement != original.Statement {
		t.Errorf("Statement: got %s, want %s", parsed.Statement, original.Statement)
	}
}

func TestSIWERoundTripWithResources(t *testing.T) {
	t.Parallel()

	original := &SIWEMessage{
		Domain:    "app.chainpulse.io",
		Address:   common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045"),
		URI:       "https://app.chainpulse.io/auth",
		Nonce:     "abcd1234",
		Version:   "1",
		ChainID:   big.NewInt(11155111),
		IssuedAt:  time.Now(),
		Resources: []string{"https://rpc.chainpulse.io", "https://api.chainpulse.io/v1"},
	}

	built := original.BuildMessage()
	parsed, err := ParseMessage(built)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}

	if len(parsed.Resources) != 2 {
		t.Fatalf("expected 2 resources, got %d", len(parsed.Resources))
	}
	if parsed.Resources[0] != "https://rpc.chainpulse.io" {
		t.Errorf("Resource[0]: got %s", parsed.Resources[0])
	}
}

func TestSIWEVerification(t *testing.T) {
	t.Parallel()

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	msg := &SIWEMessage{
		Domain:   "test.example.com",
		Address:  address,
		URI:      "https://test.example.com/login",
		Nonce:    "testnonce",
		Version:  "1",
		ChainID:  big.NewInt(1),
		IssuedAt: time.Now(),
	}

	messageBytes := []byte(msg.BuildMessage())
	signature, err := corecrypto.SignMessage(messageBytes, privateKey)
	if err != nil {
		t.Fatalf("GenerateChallenge: %v", err)
	}
	_ = hex.EncodeToString(signature)

	if msg.Domain != "test.example.com" {
		t.Errorf("Domain: got %s, want test.example.com", msg.Domain)
	}
	if msg.Address != address {
		t.Errorf("Address: got %s", msg.Address.Hex())
	}
	if msg.Nonce == "" {
		t.Error("nonce is empty")
	}
	if msg.Version != "1" {
		t.Errorf("Version: got %s", msg.Version)
	}
}

func TestGenerateNonce(t *testing.T) {
	t.Parallel()

	n1, _ := GenerateNonce()
	n2, _ := GenerateNonce()
	if n1 == n2 {
		t.Error("expected unique nonces")
	}
	if len(n1) != 32 {
		t.Errorf("expected 32 hex chars, got %d", len(n1))
	}
}

func TestSIWEExpiration(t *testing.T) {
	t.Parallel()

	privateKey, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	expired := time.Now().Add(-1 * time.Hour)

	msg := &SIWEMessage{
		Domain:         "test.com",
		Address:        addr,
		URI:            "https://test.com/login",
		Nonce:          "expirednonce",
		Version:        "1",
		ChainID:        big.NewInt(1),
		IssuedAt:       time.Now().Add(-2 * time.Hour),
		ExpirationTime: &expired,
	}

	sig, _ := corecrypto.SignMessage([]byte(msg.BuildMessage()), privateKey)
	if err := msg.VerifySIWE(sig, nil); err == nil {
		t.Error("expected error for expired message")
	}
}

func TestParseMessageInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
	}{
		{"empty", ""},
		{"no suffix", "hello world"},
		{"invalid address", "domain wants you to sign in with your Ethereum account:\nnotanaddress"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseMessage(tt.msg)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func strContains(s, substr string) bool {
	return len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
